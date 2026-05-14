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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/backends/healthcheck"
)

func TestPool_Stop_ConcurrentDoubleClose(t *testing.T) {
	const iterations = 200
	for i := range iterations {
		s := &healthcheck.Status{}
		tgt := NewTarget(http.NotFoundHandler(), s, nil)
		p := New(Targets{tgt}, 1)

		var start sync.WaitGroup
		start.Add(1)
		var done sync.WaitGroup
		done.Add(2)

		var panics atomic.Int32
		stop := func() {
			defer done.Done()
			defer func() {
				if r := recover(); r != nil {
					panics.Add(1)
				}
			}()
			start.Wait()
			p.Stop()
		}
		go stop()
		go stop()
		start.Done()
		done.Wait()

		if panics.Load() > 0 {
			t.Fatalf("iteration %d: Stop panicked on concurrent invocation (close of closed channel)", i)
		}
	}
}
