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

// Package safego is a tiny shared helper for spawning goroutines whose
// panics must not crash the process. It hoists the `go { defer recover()
// { handler(r) } }` boilerplate so each consumer only writes the panic
// handler (typically log + metric increment). Callers retain full
// control over the metric name, labels, and any side effects (eg
// flipping an "exited" flag) by passing a closure.
//
// The helper does not pick a counter or log message; that's intentional.
// Each consumer has its own observability surface and labels, and one-
// size-fits-all has driven the prior ad-hoc duplication this package
// replaces.
package safego

import "runtime/debug"

// PanicHandler is invoked when Go or Run recovers a panic. r is the
// value passed to panic(); stack is the goroutine's stack trace at the
// point of recovery.
type PanicHandler func(r any, stack []byte)

// Go starts fn in a new goroutine. On panic, handler is invoked with
// the recovered value and stack trace. Use Run if fn is already running
// inside a goroutine body.
func Go(handler PanicHandler, fn func()) {
	go Run(handler, fn)
}

// Run executes fn synchronously, recovering panics into handler.
func Run(handler PanicHandler, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			handler(r, debug.Stack())
		}
	}()
	fn()
}
