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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/backends/alb/pool"
	"github.com/trickstercache/trickster/v2/pkg/backends/healthcheck"
)

func mkFlapTarget(t *testing.T) *pool.Target {
	t.Helper()
	st := &healthcheck.Status{}
	st.Set(healthcheck.StatusPassing)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return pool.NewTarget(h, st, nil)
}

// TestAllRacesPerTargetHealthFlip races per-target hcStatus flips against
// in-flight fanout.All invocations. The assertion is data-race freedom under
// -race; success rate of the fanouts is not checked.
func TestAllRacesPerTargetHealthFlip(t *testing.T) {
	t.Parallel()

	const numTargets = 6
	targets := make(pool.Targets, numTargets)
	for i := range numTargets {
		targets[i] = mkFlapTarget(t)
	}

	stop := make(chan struct{})
	var flipperIters, fanoutIters atomic.Int64

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		toggle := int32(healthcheck.StatusFailing)
		for {
			select {
			case <-stop:
				return
			default:
			}
			for _, tgt := range targets {
				tgt.HealthStatus().Set(toggle)
			}
			if toggle == healthcheck.StatusFailing {
				toggle = healthcheck.StatusPassing
			} else {
				toggle = healthcheck.StatusFailing
			}
			flipperIters.Add(1)
		}
	}()

	go func() {
		defer wg.Done()
		parent := newParentGET(t)
		ctx := context.Background()
		cfg := Config{Mechanism: "test"}
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, _ = All(ctx, parent, targets, cfg)
			fanoutIters.Add(1)
		}
	}()

	// success = ran enough to exercise data races, not a perf benchmark.
	// Generous deadline gives CI scheduler tail-latency room; low floor means
	// "ran enough to be meaningful" rather than "ran fast enough to hit N".
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			close(stop)
			wg.Wait()
			if fanoutIters.Load() == 0 {
				t.Fatal("no fanout iterations completed")
			}
			if flipperIters.Load() == 0 {
				t.Fatal("no flipper iterations completed")
			}
			return
		default:
			if fanoutIters.Load() >= 50 {
				close(stop)
				wg.Wait()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}
}
