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
	"net/http/httptest"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

// A panicking pool member must not crash the proxy. RecoverFanoutPanic in
// the NLM fanout goroutine should swallow the panic and clear the capture
// slot so the fallback path doesn't pick a partial response.
func TestNLMPanicMemberDoesNotCrashRequest(t *testing.T) {
	lm := "Sun, 06 Nov 1994 08:49:37 GMT"
	healthy := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headers.NameLastModified, lm)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body-ok"))
	})

	p, _, _ := albpool.NewHealthy([]http.Handler{albpool.PanicHandler(), healthy})
	defer p.Stop()
	albpool.WaitHealthy(t, p, 2)

	h := &handler{}
	h.SetPool(p)
	w := httptest.NewRecorder()
	albpool.ServeAndWait(t, h, w, albpool.NewParentGET(t))

	if w.Body.String() != "body-ok" {
		t.Errorf("expected healthy member body, got %q", w.Body.String())
	}
}

func TestNLMPanicAllMembersDoesNotCrashRequest(t *testing.T) {
	p, _, _ := albpool.NewHealthy([]http.Handler{albpool.PanicHandler(), albpool.PanicHandler()})
	defer p.Stop()
	albpool.WaitHealthy(t, p, 2)

	h := &handler{}
	h.SetPool(p)
	w := httptest.NewRecorder()
	albpool.RequireFanoutFailureDelta(t, "nlm", "", "panic", 2, func() {
		albpool.ServeAndWait(t, h, w, albpool.NewParentGET(t))
	})
	if w.Code < 500 {
		t.Errorf("expected 5xx, got %d", w.Code)
	}
}
