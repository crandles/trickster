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
	"strings"
	"testing"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

func TestNewestLastModifiedSkipsTruncated(t *testing.T) {
	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	smallBody := "ok"
	bigBody := strings.Repeat("X", 4096)

	smallH := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headers.NameLastModified, older.UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(smallBody))
	})
	bigH := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headers.NameLastModified, newer.UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", "4096")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bigBody))
	})

	p, _, _ := albpool.NewHealthy([]http.Handler{smallH, bigH})
	defer p.Stop()
	p.RefreshHealthy()
	albpool.WaitHealthy(t, p, 2)

	h := &handler{maxCaptureBytes: 10}
	h.SetPool(p)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, albpool.NewParentGET(t))

	if got := w.Body.String(); got != smallBody {
		t.Fatalf("expected truncated upstream to be disqualified; want body %q, got %q (len=%d)",
			smallBody, got, len(got))
	}
	if cl := w.Header().Get("Content-Length"); cl != "" && cl != "2" {
		t.Errorf("served Content-Length %q does not match served body length %d", cl, w.Body.Len())
	}
}
