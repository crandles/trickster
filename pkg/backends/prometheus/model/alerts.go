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

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sort"

	"github.com/trickstercache/trickster/v2/pkg/backends/providers"
	"github.com/trickstercache/trickster/v2/pkg/checksum/fnv"
	"github.com/trickstercache/trickster/v2/pkg/observability/logging"
	"github.com/trickstercache/trickster/v2/pkg/observability/logging/logger"
	"github.com/trickstercache/trickster/v2/pkg/proxy/handlers/trickster/failures"
	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/proxy/response/merge"
	"github.com/trickstercache/trickster/v2/pkg/timeseries/dataset"
)

// WFAlerts is the Wire Format Document for the /alerts endpoint
type WFAlerts struct {
	*Envelope
	Data *WFAlertData `json:"data"`
}

// WFAlertData is the Wire Format Document for the alerts list in /alerts responses
type WFAlertData struct {
	Alerts []WFAlert `json:"alerts"`
}

// WFAlert is the Wire Format Document for the alert object in /alerts responses
type WFAlert struct {
	ActiveAt    string            `json:"activeAt,omitempty"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`
	State       string            `json:"state"`
	Value       string            `json:"value,omitempty"`
}

// CalculateHash sums the FNV64a hash for the Header and stores it to the Hash member
func (a *WFAlert) CalculateHash() uint64 {
	hash := fnv.NewInlineFNV64a()
	hash.Write([]byte(dataset.Tags(a.Labels).String()))
	hash.Write([]byte("||"))
	hash.Write([]byte(dataset.Tags(a.Annotations).String()))
	return hash.Sum64()
}

// Merge merges the passed WFAlerts into the subject WFAlerts
// by preferring higher-severity states during de-duplication
func (a *WFAlerts) Merge(results ...*WFAlerts) {

	m := map[uint64]WFAlert{}

	if a.Data != nil && len(a.Data.Alerts) > 0 {
		for _, d := range a.Data.Alerts {
			m[d.CalculateHash()] = d
		}
	}

	for _, a2 := range results {
		a.Envelope.Merge(a2.Envelope)
		if a2.Data != nil && len(a2.Data.Alerts) > 0 {
			for _, d := range a2.Data.Alerts {
				h := d.CalculateHash()
				if d2, ok := m[h]; !ok ||
					((d2.State == "inactive" && (d.State == "pending" || d.State == "firing")) ||
						(d2.State == "pending" && d.State == "firing")) {
					m[h] = d
				}
			}
		}
	}

	alerts := make([]WFAlert, len(m))
	for j, k := range slices.Sorted(maps.Keys(m)) {
		alerts[j] = m[k]
	}

	a.Data.Alerts = alerts

}

// MergeAndWriteAlerts merges the provided Responses into a single prometheus Alerts data object,
// and writes it to the provided ResponseWriter
func MergeAndWriteAlerts(w http.ResponseWriter, r *http.Request, rgs merge.ResponseGates) {
	var a *WFAlerts
	responses, bestResp := gatherResponses(r, rgs, func(rg *merge.ResponseGate) bool {
		a1 := &WFAlerts{}
		err := json.Unmarshal(rg.Body(), &a1)
		if err != nil {
			logger.Error("alerts unmarshaling error",
				logging.Pairs{"provider": providers.Prometheus, "detail": err.Error()})
			return false
		}
		if a == nil {
			a = a1
		} else {
			a.Merge(a1)
		}
		return true
	})

	if a == nil || len(responses) == 0 {
		if bestResp != nil {
			h := w.Header()
			headers.Merge(h, bestResp.Header)
			w.WriteHeader(bestResp.StatusCode)
			io.Copy(w, bestResp.Body)

		} else {
			failures.HandleBadGateway(w, r)
		}
		return
	}

	sort.Ints(responses)
	statusCode := responses[0]
	a.StartMarshal(w, statusCode)

	var sep string
	w.Write([]byte(`,"data":{"alerts":[`))
	if a.Data != nil && len(a.Data.Alerts) > 0 {
		for _, alert := range a.Data.Alerts {
			fmt.Fprintf(w,
				`{"state":"%s","labels":%s,"annotations":%s`,
				alert.State, dataset.Tags(alert.Labels).JSON(),
				dataset.Tags(alert.Annotations).JSON(),
			)
			if alert.Value != "" {
				fmt.Fprintf(w, `,"value":"%s"`, alert.Value)
			}
			if alert.ActiveAt != "" {
				fmt.Fprintf(w, `,"activeAt":"%s"`, alert.ActiveAt)
			}
			w.Write([]byte("}" + sep))
			sep = ","
		}
	}
	w.Write([]byte("]}}")) // complete the alert list and the envelope
}

// helper function to gather responses from a ResponseGate
func gatherResponses(_ *http.Request, rgs merge.ResponseGates, handler func(*merge.ResponseGate) bool) ([]int, *http.Response) {
	responses := make([]int, len(rgs))
	var bestResp *http.Response
	for i, rg := range rgs {
		if rg == nil {
			continue
		}
		if rg.Resources != nil && rg.Resources.Response != nil {
			resp := rg.Resources.Response
			responses[i] = resp.StatusCode

			if resp.Body != nil {
				defer resp.Body.Close()
			}

			if resp.StatusCode < 400 {
				ok := handler(rg)
				if !ok {
					continue
				}
			}
			if bestResp == nil || resp.StatusCode < bestResp.StatusCode {
				bestResp = resp
				resp.Body = io.NopCloser(bytes.NewReader(rg.Body()))
			}
		}
	}

	return responses, bestResp
}
