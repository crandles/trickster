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

package pool

import (
	"net/http"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/backends/healthcheck"
)

// TestPool_StopThenSetHealthy_NoRefreshOverwrite reproduces a race where the
// initial scheduleRefresh queued by New() could fire RefreshHealthy after Stop
// returned, overwriting healthyHandlers and clobbering a subsequent
// SetHealthy call. Stop must wait for the refresh worker so that any caller
// using the Stop-then-SetHealthy pattern observes deterministic state.
func TestPool_StopThenSetHealthy_NoRefreshOverwrite(t *testing.T) {
	const iterations = 500
	for i := range iterations {
		s := &healthcheck.Status{}
		tgt := NewTarget(http.NotFoundHandler(), s, nil)
		p := New(Targets{tgt}, 1)
		p.Stop()
		h := []http.Handler{http.NotFoundHandler(), http.NotFoundHandler()}
		p.SetHealthy(h)
		if got := len(p.Targets()); got != 2 {
			t.Fatalf("iteration %d: Targets: expected 2 got %d "+
				"(RefreshHealthy ran after Stop returned)", i, got)
		}
		if got := len(p.(*pool).snapshot()); got != 2 {
			t.Fatalf("iteration %d: snapshot: expected 2 got %d", i, got)
		}
	}
}
