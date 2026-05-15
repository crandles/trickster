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

package tsm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/proxy/request"
	tu "github.com/trickstercache/trickster/v2/pkg/testutil"
	"github.com/trickstercache/trickster/v2/pkg/testutil/albpool"
)

// A panicking pool member must not crash the request. RecoverFanoutPanic("tsm",
// ...) at time_series_merge.go must catch it and mark the slot failed so the
// merge surfaces the partial-failure (phit) signal.
func TestTSMPanicMemberDoesNotCrashRequest(t *testing.T) {
	p, _, _ := albpool.NewHealthy([]http.Handler{
		http.HandlerFunc(tu.BasicHTTPHandler),
		albpool.PanicHandler(),
	})
	defer p.Stop()
	albpool.WaitHealthy(t, p, 2)

	rsc := request.NewResources(nil, nil, nil, nil, nil, nil)
	rsc.IsMergeMember = true
	r := request.SetResources(albpool.NewParentGET(t), rsc)

	h := &handler{mergePaths: []string{"/"}}
	h.SetPool(p)
	w := httptest.NewRecorder()
	albpool.ServeAndWait(t, h, w, r)
}

func TestTSMPanicAllMembersDoesNotCrashRequest(t *testing.T) {
	p, _, _ := albpool.NewHealthy([]http.Handler{albpool.PanicHandler(), albpool.PanicHandler()})
	defer p.Stop()
	albpool.WaitHealthy(t, p, 2)

	rsc := request.NewResources(nil, nil, nil, nil, nil, nil)
	rsc.IsMergeMember = true
	r := request.SetResources(albpool.NewParentGET(t), rsc)

	h := &handler{mergePaths: []string{"/"}}
	h.SetPool(p)
	w := httptest.NewRecorder()
	albpool.RequireFanoutFailureDelta(t, "tsm", "", "panic", 2, func() {
		albpool.ServeAndWait(t, h, w, r)
	})
	if w.Code < 500 {
		t.Errorf("expected 5xx, got %d", w.Code)
	}
}
