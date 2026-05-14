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
	"strconv"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

func TestNLM_AllTruncated_Returns502(t *testing.T) {
	const maxBytes = 8
	const bodySize = 4096

	oversized := func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			body := make([]byte, bodySize)
			for i := range body {
				body[i] = 'a'
			}
			w.Header().Set("Content-Length", strconv.Itoa(bodySize))
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		})
	}

	hs := []http.Handler{oversized(), oversized(), oversized()}
	p, _, _ := albpool.NewHealthy(hs)
	defer p.Stop()
	p.SetHealthy(hs)

	h := &handler{maxCaptureBytes: maxBytes}
	h.SetPool(p)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, albpool.NewParentGET(t))

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when all members are truncated, got %d (body len=%d)",
			w.Code, w.Body.Len())
	}
}

func TestNLM_AllFailed_Returns502(t *testing.T) {
	hs := make([]http.Handler, 3)
	for i := range hs {
		hs[i] = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("upstream blew up")
		})
	}

	p, _, _ := albpool.NewHealthy(hs)
	defer p.Stop()
	p.SetHealthy(hs)

	h := &handler{}
	h.SetPool(p)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, albpool.NewParentGET(t))

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when all members fail, got %d (body len=%d)",
			w.Code, w.Body.Len())
	}
}
