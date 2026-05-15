/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trickstercache/trickster/v2/integration/internal/chaos"
)

// TestALBChaosBehaviors drives each ALB winner-takes-all mechanism (fr, nlm)
// against a 2-target pool consisting of one always-healthy stub and one
// chaos-misbehaving stub. The chaos stub's healthcheck endpoint always
// responds 200 (so it joins the dispatch set), but its data path emits the
// configured misbehavior.
//
// This test exists to make `integration/internal/chaos` a real consumer
// instead of an unused fixture library, and to drive each misbehavior
// through the real proxy + ALB path so a regression that crashes the
// process or breaks startup is caught. Stricter correctness assertions
// (eg "chaos stub never wins for FR") are deferred until the adjacent-bug
// fixes in PR 3 land; today the proxy still serves short bodies under a
// stale Content-Length (see [proxy_request.go:436]).
func TestALBChaosBehaviors(t *testing.T) {
	if testing.Short() {
		t.Skip("chaos matrix is slow; skipping in -short mode")
	}

	type cell struct {
		mech     string
		behavior string
		fn       http.HandlerFunc
	}
	cells := []cell{
		{"fr", "panic", chaos.BehaviorPanic()},
		{"fr", "truncate_stale_cl", chaos.BehaviorTruncateStaleCL(4096, 16)},
		{"fr", "slow_probe", chaos.BehaviorSlowProbe(2 * time.Second)},
		{"nlm", "panic", chaos.BehaviorPanic()},
		{"nlm", "truncate_stale_cl", chaos.BehaviorTruncateStaleCL(4096, 16)},
		{"nlm", "5xx_with_lm", chaos.Behavior5xxWithLM(http.StatusInternalServerError,
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))},
	}

	port := 25000
	for _, c := range cells {
		port += 10
		front, metrics, mgmt := port, port+1, port+2
		t.Run(fmt.Sprintf("%s_%s", c.mech, c.behavior), func(t *testing.T) {
			runChaosCell(t, c.mech, c.behavior, c.fn, front, metrics, mgmt)
		})
	}
}

func runChaosCell(t *testing.T, mech, behavior string, chaosData http.HandlerFunc,
	frontPort, metricsPort, mgmtPort int,
) {
	t.Helper()

	healthy := newPathAwareStub(t, chaos.BehaviorOK(promVectorBody("healthy")))
	misbehaver := newPathAwareStub(t, chaosData)

	cfgPath := writeChaosConfig(t, mech, frontPort, metricsPort, mgmtPort,
		healthy.URL, misbehaver.URL)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go startTrickster(t, ctx, expectedStartError{}, "-config", cfgPath)
	waitForTrickster(t, fmt.Sprintf("127.0.0.1:%d", metricsPort))

	// Let healthchecks converge. Both stubs answer 200 on /buildinfo, so both
	// join the pool; the data-path misbehavior is what the ALB must absorb.
	time.Sleep(800 * time.Millisecond)

	cli := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        4,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	const reqs = 5
	const healthyBodyLen = 100 // promVectorBody is ~140 bytes; below this implies a truncated chaos win
	var ok, nonOK, short int
	for i := range reqs {
		q := fmt.Sprintf("up + 0*%d", time.Now().UnixNano()+int64(i))
		u := fmt.Sprintf("http://127.0.0.1:%d/alb-%s/api/v1/query?query=%s",
			frontPort, mech, url.QueryEscape(q))
		resp, err := cli.Get(u)
		if err != nil {
			t.Logf("%s/%s: request %d transport err=%v", mech, behavior, i, err)
			nonOK++
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			ok++
			if len(body) < healthyBodyLen {
				short++
				t.Logf("%s/%s: short 200 body (%d bytes): %q",
					mech, behavior, len(body), string(body))
			}
		} else {
			nonOK++
		}
	}
	t.Logf("%s/%s: ok=%d nonOK=%d short=%d", mech, behavior, ok, nonOK, short)

	// Every request must get a terminal response (process is still running).
	assert.Equalf(t, reqs, ok+nonOK,
		"%s/%s: %d/%d requests dropped (transport-level failure)",
		mech, behavior, reqs-(ok+nonOK), reqs)
	// The truncate_stale_cl chaos stub declares Content-Length: 4096 and
	// writes 16 bytes. With the fanout short-read disqualifier wired up
	// (pkg/backends/alb/mech/fanout/fanout.go), FR/NLM must reject the
	// chaos slot and serve the healthy member's full body, not the
	// truncated 16-byte body.
	if behavior == "truncate_stale_cl" {
		assert.Zerof(t, short,
			"%s/%s: served %d short bodies; chaos stub should be disqualified by short-read check",
			mech, behavior, short)
	}
}

// pathAwareStub serves /api/v1/status/buildinfo with a fixed healthy 200
// response and routes all other paths to the supplied data handler. This
// lets a chaos handler misbehave only on data queries, so the pool's
// healthcheck still treats the stub as available. Panics from `data`
// propagate to net/http's default recover, which closes the connection
// without writing -- the same shape a buggy real upstream produces.
type pathAwareStub struct {
	srv *httptest.Server
	URL string
}

func newPathAwareStub(t *testing.T, data http.HandlerFunc) *pathAwareStub {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status/buildinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"version":"2.0"}}`))
	})
	mux.Handle("/", data)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &pathAwareStub{srv: srv, URL: srv.URL}
}

func promVectorBody(job string) string {
	return fmt.Sprintf(
		`{"status":"success","data":{"resultType":"vector","result":[`+
			`{"metric":{"__name__":"up","job":%q},"value":[1700000000,"1"]}]}}`,
		job)
}

func writeChaosConfig(t *testing.T, mech string, frontPort, metricsPort, mgmtPort int,
	healthyURL, chaosURL string,
) string {
	t.Helper()
	var sb strings.Builder
	fmt.Fprintf(&sb, "frontend:\n  listen_port: %d\n", frontPort)
	fmt.Fprintf(&sb, "metrics:\n  listen_port: %d\n", metricsPort)
	fmt.Fprintf(&sb, "mgmt:\n  listen_port: %d\n", mgmtPort)
	sb.WriteString("logging:\n  log_level: error\n")
	sb.WriteString("caches:\n  mem1:\n    provider: memory\n")
	sb.WriteString("backends:\n")
	for i, u := range []string{healthyURL, chaosURL} {
		fmt.Fprintf(&sb, "  prom%d:\n", i)
		sb.WriteString("    provider: prometheus\n")
		fmt.Fprintf(&sb, "    origin_url: %s\n", u)
		sb.WriteString("    cache_name: mem1\n")
		sb.WriteString("    healthcheck:\n")
		sb.WriteString("      path: /api/v1/status/buildinfo\n")
		sb.WriteString("      query: \"\"\n")
		sb.WriteString("      interval: 100ms\n")
		sb.WriteString("      timeout: 500ms\n")
		sb.WriteString("      failure_threshold: 1\n")
		sb.WriteString("      recovery_threshold: 1\n")
	}
	fmt.Fprintf(&sb, "  alb-%s:\n", mech)
	sb.WriteString("    provider: alb\n")
	sb.WriteString("    alb:\n")
	fmt.Fprintf(&sb, "      mechanism: %s\n", mech)
	sb.WriteString("      pool:\n")
	for i := range 2 {
		fmt.Fprintf(&sb, "        - prom%d\n", i)
	}
	cfgPath := filepath.Join(t.TempDir(), "trickster.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(sb.String()), 0o600))
	return cfgPath
}
