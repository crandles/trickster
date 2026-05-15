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

// Package golden provides shared JSON-fixture helpers used by tests that
// maintain on-disk golden files under testdata/. Tests can opt into the
// regenerate-on-flag pattern by checking Update and calling WriteJSON in
// bootstrap tests. The package owns the -update flag so multiple consumers
// (eg pkg/backends/alb/mech/tsm + future fr/nlm body fixtures) share one
// switch.
package golden

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Update is the shared -update flag. Bootstrap tests that maintain JSON
// fixtures should check *Update and call WriteJSON when set.
var Update = flag.Bool("update", false, "rewrite testdata/*.json golden fixtures from current test output")

// LoadJSON reads testdata/<name>.json relative to the calling test's package
// directory and decodes it into out. The error message points at the
// -update flag when the file is missing.
func LoadJSON(t testing.TB, name string, out any) {
	t.Helper()
	path := filepath.Join("testdata", name+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", path, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("unmarshal golden %s: %v", path, err)
	}
}

// WriteJSON marshals v with MarshalIndent and writes it to
// testdata/<name>.json, creating the directory if necessary. Used by
// bootstrap tests under `go test -update`.
func WriteJSON(t testing.TB, name string, v any) {
	t.Helper()
	path := filepath.Join("testdata", name+".json")
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
