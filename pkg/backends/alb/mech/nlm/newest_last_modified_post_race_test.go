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

package nlm

import (
	"net/http"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/backends/alb/pool"
	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

// NLM fanout clones the parent request N ways. For POST/PUT/PATCH the body
// must be primed before fanout, else N goroutines race on r.Body and
// rsc.RequestBody inside GetBody. Run under -race to catch any regression.
func TestNLMPostBodyFanoutIsRaceFree(t *testing.T) {
	const body = `{"query":"sum(rate(metric[5m]))","start":"2024-01-01T00:00:00Z","end":"2024-01-01T01:00:00Z","step":"15s"}`
	const lm = "Sun, 06 Nov 1994 08:49:37 GMT"
	albpool.RunPostBodyFanoutRace(t, func(p pool.Pool) http.Handler {
		h := &handler{}
		h.SetPool(p)
		return h
	}, body, 4, 16, func(w http.ResponseWriter) {
		w.Header().Set(headers.NameLastModified, lm)
	})
}
