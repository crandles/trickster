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

package sets

import (
	"maps"
	"slices"
)

// Set is a collection of unique elements
type Set[T comparable] map[T]struct{}

// New creates a new Set from a slice of keys.
func New[T comparable](keys []T) Set[T] {
	s := make(Set[T], len(keys))
	s.SetAll(keys)
	return s
}

func MapKeysToStringSet[V any](m map[string]V) Set[string] {
	if m == nil {
		return nil
	}
	out := make(Set[string], len(m))
	for k := range m {
		out.Set(k)
	}
	return out
}

// NewInt64Set returns a new Set[int64]
func NewInt64Set() Set[int64] {
	return make(Set[int64])
}

// NewIntSet returns a new Set[int]
func NewIntSet() Set[int] {
	return make(Set[int])
}

// NewStringSet returns a new Set[string]
func NewStringSet() Set[string] {
	return make(Set[string])
}

// NewByteSet returns a new Set[byte]
func NewByteSet() Set[byte] {
	return make(Set[byte])
}

func (s Set[T]) SetAll(vals []T) {
	for _, val := range vals {
		s.Set(val)
	}
}

// Set inserts a value into the set.
func (s Set[T]) Set(val T) {
	s[val] = struct{}{}
}

// Add inserts a value into the Set if does not already exist. True is returned
// only if the value was inserted.
func (s Set[T]) Add(val T) bool {
	if s.Contains(val) {
		return false
	}
	s.Set(val)
	return true
}

// Remove deletes a value from the set.
func (s Set[T]) Remove(val T) {
	delete(s, val)
}

// Contains checks if a value is in the set.
func (s Set[T]) Contains(val T) bool {
	_, ok := s[val]
	return ok
}

// Keys returns the set elements as a slice in an unpredictable order.
func (s Set[T]) Keys() []T {
	out := make([]T, len(s))
	var i int
	for key := range s {
		out[i] = key
		i++
	}
	return out
}

// Sorted returns the set elements as a sorted slice.
func (s Set[T]) Sorted(less func(a, b T) int) []T {
	out := s.Keys()
	slices.SortFunc(out, less)
	return out
}

// Clone returns a new independent copy of the set.
func (s Set[T]) Clone() Set[T] {
	return maps.Clone(s)
}

// Merge adds all elements from another set into this one.
func (s Set[T]) Merge(other Set[T]) {
	maps.Copy(s, other)
}
