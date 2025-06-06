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

import "testing"

func TestClone(t *testing.T) {

	const expectedMS = 87
	const expectedLen = 1

	o := &Options{
		InstantRound: expectedMS,
		Labels:       map[string]string{"test": "trickster"},
	}

	o2 := o.Clone()
	if o2.InstantRound != expectedMS {
		t.Errorf("expected %d got %d", expectedMS, o2.InstantRound)
	}
	if len(o2.Labels) != expectedLen {
		t.Errorf("expected %d got %d", expectedLen, len(o2.Labels))
	}

}
