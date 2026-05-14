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

package switcher

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSwitchHandlerConcurrentUpdateReadIsRaceFree(t *testing.T) {
	t.Parallel()

	initial := http.NewServeMux()
	sh := NewSwitchHandler(initial)

	var stop atomic.Bool
	var wg sync.WaitGroup

	const writers, readers = 8, 8

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				sh.Update(http.NewServeMux())
			}
		}()
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for !stop.Load() {
				w := httptest.NewRecorder()
				sh.ServeHTTP(w, r)
			}
		}()
	}

	time.Sleep(500 * time.Millisecond)
	stop.Store(true)
	wg.Wait()
}
