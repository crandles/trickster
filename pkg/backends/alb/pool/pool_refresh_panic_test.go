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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/trickstercache/trickster/v2/pkg/observability/metrics"
)

// TestRefreshWorkerSurvivesPanic asserts that a panic inside the refresh
// worker iteration body is recovered, the panic-recovered counter increments,
// and a subsequent iteration runs normally. Exercises the runWithRecover
// contract directly so we don't need to engineer a panicking RefreshHealthy.
func TestRefreshWorkerSurvivesPanic(t *testing.T) {
	t.Parallel()

	p := &pool{}

	before := counterValue(t, metrics.ALBPoolRefreshPanicRecovered, "checkHealth")

	// First iteration panics; runWithRecover must absorb it.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic escaped runWithRecover: %v", r)
			}
		}()
		p.runWithRecover("checkHealth", func() {
			panic("simulated refresh panic")
		})
	}()

	after := counterValue(t, metrics.ALBPoolRefreshPanicRecovered, "checkHealth")
	if got := after - before; got != 1 {
		t.Fatalf("expected ALBPoolRefreshPanicRecovered{worker=checkHealth} +1, got +%v", got)
	}

	// Subsequent iteration must still execute.
	ran := false
	p.runWithRecover("checkHealth", func() {
		ran = true
	})
	if !ran {
		t.Fatal("iteration after recovered panic did not execute")
	}

	// listenStatusUpdates worker label is observed independently.
	lbefore := counterValue(t, metrics.ALBPoolRefreshPanicRecovered, "listenStatusUpdates")
	p.runWithRecover("listenStatusUpdates", func() {
		panic("simulated status panic")
	})
	lafter := counterValue(t, metrics.ALBPoolRefreshPanicRecovered, "listenStatusUpdates")
	if got := lafter - lbefore; got != 1 {
		t.Fatalf("expected ALBPoolRefreshPanicRecovered{worker=listenStatusUpdates} +1, got +%v", got)
	}
}

func counterValue(t *testing.T, vec *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()
	c, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	if m.Counter == nil || m.Counter.Value == nil {
		return 0
	}
	return *m.Counter.Value
}
