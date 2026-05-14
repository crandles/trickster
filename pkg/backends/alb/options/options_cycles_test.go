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

package options

import (
	"strings"
	"testing"
)

func TestValidateNoCycles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		albs    map[string]*Options
		wantErr bool
		errSub  string
	}{
		{
			name: "direct 2-cycle a->b->a",
			albs: map[string]*Options{
				"a": {Pool: []string{"b"}},
				"b": {Pool: []string{"a"}},
			},
			wantErr: true,
			errSub:  "cycle",
		},
		{
			name: "self-loop a->a",
			albs: map[string]*Options{
				"a": {Pool: []string{"a"}},
			},
			wantErr: true,
			errSub:  "cycle",
		},
		{
			name: "3-cycle a->b->c->a",
			albs: map[string]*Options{
				"a": {Pool: []string{"b"}},
				"b": {Pool: []string{"c"}},
				"c": {Pool: []string{"a"}},
			},
			wantErr: true,
			errSub:  "cycle",
		},
		{
			name: "acyclic DAG: alb_root targets alb_mid targets prom1 (leaf)",
			albs: map[string]*Options{
				"alb_root": {Pool: []string{"alb_mid", "prom2"}},
				"alb_mid":  {Pool: []string{"prom1"}},
			},
			wantErr: false,
		},
		{
			name: "two disjoint ALBs, no cycle",
			albs: map[string]*Options{
				"alb1": {Pool: []string{"prom1"}},
				"alb2": {Pool: []string{"prom2"}},
			},
			wantErr: false,
		},
		{
			name: "diamond: a->b, a->c, b->d, c->d (DAG, not a cycle)",
			albs: map[string]*Options{
				"a": {Pool: []string{"b", "c"}},
				"b": {Pool: []string{"d"}},
				"c": {Pool: []string{"d"}},
				"d": {Pool: []string{"prom1"}},
			},
			wantErr: false,
		},
		{
			name:    "empty",
			albs:    map[string]*Options{},
			wantErr: false,
		},
		{
			name: "10-length cycle a0->a1->...->a9->a0",
			albs: map[string]*Options{
				"a0": {Pool: []string{"a1"}},
				"a1": {Pool: []string{"a2"}},
				"a2": {Pool: []string{"a3"}},
				"a3": {Pool: []string{"a4"}},
				"a4": {Pool: []string{"a5"}},
				"a5": {Pool: []string{"a6"}},
				"a6": {Pool: []string{"a7"}},
				"a7": {Pool: []string{"a8"}},
				"a8": {Pool: []string{"a9"}},
				"a9": {Pool: []string{"a0"}},
			},
			wantErr: true,
			errSub:  "cycle",
		},
		{
			// pins the leaf-skip in ValidateNoCycles: pool members absent
			// from the albs map are leaves and must not drive recursion. The
			// shape mixes a real ALB target with several non-ALB names that
			// would otherwise be unreachable; a regression that recursed into
			// missing keys would either panic on nil deref or fail to detect
			// the alb_root -> alb_mid edge.
			name: "alb pool mixes alb target with non-alb leaves",
			albs: map[string]*Options{
				"alb_root": {Pool: []string{"prom_a", "alb_mid", "prom_b"}},
				"alb_mid":  {Pool: []string{"prom_c"}},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateNoCycles(tc.albs)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
