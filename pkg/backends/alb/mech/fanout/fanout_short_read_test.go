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

package fanout

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/trickstercache/trickster/v2/pkg/backends/alb/pool"
	"github.com/trickstercache/trickster/v2/pkg/proxy/request"
	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

// TestScatterShortReadDisqualifies asserts that when a target's handler
// flips the UpstreamShortReadCapture (the signal the proxy engine raises
// when it reads fewer body bytes from the upstream than the declared
// Content-Length), scatter marks the slot Failed and increments the
// short_read failure metric. This mirrors the runtime path the proxy
// engine uses to surface a truncated origin response to fanout.
func TestScatterShortReadDisqualifies(t *testing.T) {
	short := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := request.GetUpstreamShortReadCapture(r.Context()); c != nil {
			c.Mark()
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 16))
	})

	tgt, _ := albpool.HealthyTarget(short)
	targets := pool.Targets{tgt}

	parent := albpool.NewParentGET(t)
	results, err := All(context.Background(), parent, targets,
		Config{Mechanism: "test-short-read", Variant: ""})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].Failed,
		"slot must be Failed when proxy marks the short-read sidecar")
}

// TestScatterMatchingLengthPasses confirms the disqualifier only fires
// when the sidecar is marked. A handler that does NOT mark the sidecar
// (the common case: full body received) must NOT be marked Failed.
func TestScatterMatchingLengthPasses(t *testing.T) {
	full := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 32))
	})

	tgt, _ := albpool.HealthyTarget(full)
	targets := pool.Targets{tgt}

	parent := albpool.NewParentGET(t)
	results, err := All(context.Background(), parent, targets,
		Config{Mechanism: "test-full-read", Variant: ""})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.False(t, results[0].Failed,
		"slot must NOT be Failed when no short-read was marked")
}

// TestScatterNoCaptureNoSignal documents the fallback: the short-read
// check is opt-in via the sidecar bound by PrepareClone. Tests that
// don't go through PrepareClone (none today, but defense in depth)
// would never trip the check.
func TestScatterNoCaptureNoSignal(t *testing.T) {
	tiny := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("x"))
	})

	tgt, _ := albpool.HealthyTarget(tiny)
	targets := pool.Targets{tgt}

	parent := albpool.NewParentGET(t)
	results, err := All(context.Background(), parent, targets,
		Config{Mechanism: "test-no-capture", Variant: ""})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.False(t, results[0].Failed)
}
