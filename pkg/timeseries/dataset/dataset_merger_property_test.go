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

package dataset

import (
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

// TestPropertyDefaultMergerPreservesWarnings is the property formulation of
// the regression fixed in the alb-cont round-3 hardening sweep: every
// per-shard Warning emitted by any input DataSet must appear in the merged
// result, regardless of how many shards or in what order they arrive.
// Multiset equality is the contract; ordering across shards is not promised.
func TestPropertyDefaultMergerPreservesWarnings(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Receiver + 1-4 inputs. The receiver's own Warnings are part of the
		// expected multiset; the merge appends each input's warnings in turn.
		recvWarn := rapid.SliceOfN(rapid.String(), 0, 4).Draw(rt, "recvWarnings")
		inputCount := rapid.IntRange(1, 4).Draw(rt, "inputs")
		inputs := make([]timeseries.Timeseries, inputCount)
		want := append([]string(nil), recvWarn...)
		for i := range inputs {
			ws := rapid.SliceOfN(rapid.String(), 0, 4).Draw(rt, "warnings")
			inputs[i] = &DataSet{Warnings: append([]string(nil), ws...)}
			want = append(want, ws...)
		}

		recv := &DataSet{Warnings: append([]string(nil), recvWarn...)}
		recv.DefaultMerger(false, inputs...)

		got := append([]string(nil), recv.Warnings...)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			rt.Fatalf("warnings multiset mismatch:\n  got:  %v\n  want: %v", recv.Warnings, want)
		}
	})
}

// TestPropertyDefaultMergerStatusPreference asserts the permutation-invariant
// part of the Status precedence at dataset.go:267-269:
//
//   - any input with Status == "success" -> result is "success"
//   - all inputs (including receiver) with Status == "" -> result is ""
//
// The middle case (mixed non-success non-empty) is order-dependent because
// DefaultMerger preserves the first-seen non-empty Status; that piece is
// covered by the existing dataset_merger_warnings_test.go cases.
func TestPropertyDefaultMergerStatusPreference(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		choices := []string{"", "success", "error", "warning"}
		recvStatus := rapid.SampledFrom(choices).Draw(rt, "recvStatus")
		inputCount := rapid.IntRange(1, 4).Draw(rt, "inputs")
		statuses := make([]string, inputCount)
		inputs := make([]timeseries.Timeseries, inputCount)
		for i := range inputs {
			statuses[i] = rapid.SampledFrom(choices).Draw(rt, "status")
			inputs[i] = &DataSet{Status: statuses[i]}
		}

		all := append([]string{recvStatus}, statuses...)
		recv := &DataSet{Status: recvStatus}
		recv.DefaultMerger(false, inputs...)

		switch {
		case slices.Contains(all, "success"):
			if recv.Status != "success" {
				rt.Fatalf("any input success -> result success; got %q from %v", recv.Status, all)
			}
		case !slices.ContainsFunc(all, func(s string) bool { return s != "" }):
			if recv.Status != "" {
				rt.Fatalf("all-empty -> result empty; got %q from %v", recv.Status, all)
			}
		}
	})
}

// TestPropertyDefaultMergerIdempotent asserts merging an empty collection is
// a no-op on Warnings and Status. Surfaces the trivial idempotence
// invariant; if a future caller relies on `Merge(empty)` resetting state,
// this property fails first.
func TestPropertyDefaultMergerIdempotent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		warnings := rapid.SliceOfN(rapid.String(), 0, 4).Draw(rt, "warnings")
		status := rapid.SampledFrom([]string{"", "success", "error"}).Draw(rt, "status")
		recv := &DataSet{
			Warnings: append([]string(nil), warnings...),
			Status:   status,
		}
		recv.DefaultMerger(false)
		if !slices.Equal(recv.Warnings, warnings) {
			rt.Fatalf("Merge(empty) changed Warnings: got %v want %v", recv.Warnings, warnings)
		}
		if recv.Status != status {
			rt.Fatalf("Merge(empty) changed Status: got %q want %q", recv.Status, status)
		}
	})
}
