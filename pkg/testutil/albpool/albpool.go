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

// Package albpool provides shared test helpers for constructing ALB pools,
// targets, and parent requests. Helpers do not register cleanup; callers own
// pool lifetime via defer p.Stop(). Helpers never call t.Fatal from inside a
// handler or spawned goroutine; that pattern interacts badly with goleak.
package albpool

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/backends/alb/pool"
	"github.com/trickstercache/trickster/v2/pkg/backends/healthcheck"
)

// New constructs a pool with the given healthyFloor and one target per handler.
// Each target's status starts at the zero value (StatusUnchecked = 0); callers
// that want targets dispatch-ready should use NewHealthy or invoke Set on the
// returned statuses.
func New(healthyFloor int, hs []http.Handler) (pool.Pool,
	[]*pool.Target, []*healthcheck.Status,
) {
	var targets []*pool.Target
	var statuses []*healthcheck.Status
	if len(hs) > 0 {
		targets = make([]*pool.Target, 0, len(hs))
		statuses = make([]*healthcheck.Status, 0, len(hs))
		for _, h := range hs {
			hst := &healthcheck.Status{}
			statuses = append(statuses, hst)
			targets = append(targets, pool.NewTarget(h, hst, nil))
		}
	}
	p := pool.New(targets, healthyFloor)
	return p, targets, statuses
}

// NewHealthy builds a pool with healthyFloor=-1 and pre-sets every target's
// status to StatusPassing. It replaces the
//
//	p, _, st := albpool.New(-1, hs)
//	for _, s := range st { s.Set(0) }
//	time.Sleep(250 * time.Millisecond)
//
// boilerplate. Callers should still WaitHealthy if dispatch needs the live
// list to converge, or invoke p.SetHealthy to bypass refresh entirely.
func NewHealthy(handlers []http.Handler) (pool.Pool,
	[]*pool.Target, []*healthcheck.Status,
) {
	p, targets, statuses := New(-1, handlers)
	for _, s := range statuses {
		s.Set(healthcheck.StatusPassing)
	}
	return p, targets, statuses
}

// WaitHealthy polls p.Targets() until its length equals want or the 2s
// deadline elapses. On timeout it calls t.Fatalf. Replaces the
// time.Sleep(250 * time.Millisecond) idiom used to wait for health
// propagation.
func WaitHealthy(t testing.TB, p pool.Pool, want int) {
	t.Helper()
	const (
		deadline = 2 * time.Second
		interval = 2 * time.Millisecond
	)
	end := time.Now().Add(deadline)
	for {
		got := len(p.Targets())
		if got == want {
			return
		}
		if time.Now().After(end) {
			t.Fatalf("waited %s for %d healthy, got %d", deadline, want, got)
			return
		}
		time.Sleep(interval)
	}
}

// StatusHandler returns an http.Handler that writes the given status code and
// body. Replaces the inline statusHandler closures in fr/, nlm/, fanout/.
func StatusHandler(code int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	})
}

// NamedHandler returns an http.Handler that writes name as the response body
// with a 200 status. Useful when tests need to assert which target served a
// request.
func NamedHandler(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(name))
	})
}

// NewParentGET returns a GET request against the stock parent URL used by ALB
// tests. Fatal if request construction fails.
func NewParentGET(t testing.TB) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "https://trickstercache.org/", nil)
	if err != nil {
		t.Fatalf("albpool.NewParentGET: %v", err)
	}
	return r
}

// NewParentPOST returns a POST request against the stock parent URL with the
// given body. Fatal if request construction fails.
func NewParentPOST(t testing.TB, body io.Reader) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, "https://trickstercache.org/", body)
	if err != nil {
		t.Fatalf("albpool.NewParentPOST: %v", err)
	}
	return r
}

// Target builds a single *pool.Target backed by a fresh status. Replaces the
// mkTarget / mkFlapTarget helpers re-implemented across fanout tests.
func Target(h http.Handler) (*pool.Target, *healthcheck.Status) {
	hst := &healthcheck.Status{}
	return pool.NewTarget(h, hst, nil), hst
}
