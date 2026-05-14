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

// Targets() cached path must tolerate a target whose hcStatus is nil. The
// snapshot-builder already guards both t and t.hcStatus; the cached path
// must match that pattern as defense-in-depth against future callers that
// inject targets without driving through RefreshHealthy/SetHealthy.
func TestPoolTargetsCachedPathNilHCStatus(t *testing.T) {
	st := &healthcheck.Status{}
	st.Set(healthcheck.StatusPassing)
	good := NewTarget(http.NotFoundHandler(), st, nil)
	bad := &Target{handler: http.NotFoundHandler()} // hcStatus intentionally nil

	p := &pool{
		targets:      Targets{good},
		done:         make(chan struct{}),
		statusCh:     make(chan bool, 1),
		ch:           make(chan bool, 1),
		healthyFloor: 1,
	}

	cached := Targets{good, bad}
	p.liveTargets.Store(&cached)
	// refreshPending stays false so Targets() reads the cached path.

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Targets() panicked on nil hcStatus in cached path: %v", r)
		}
	}()
	_ = p.Targets()
}
