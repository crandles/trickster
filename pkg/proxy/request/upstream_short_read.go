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

package request

import (
	"context"
	"sync/atomic"
)

// UpstreamShortReadCapture is a sidecar that lets the proxy engine flag
// "we did not receive the full upstream body" to the ALB fanout layer
// without involving response headers (which may be modified or
// transformed by mechanism-specific rewriters before the fanout layer
// sees them). ALB sets one of these on its cloned-request context
// before dispatch. The proxy's body-copy step (proxy_request.go's
// writeResponseBody) marks it when an io.Copy from the upstream returns
// an error and the upstream had declared a Content-Length, which is
// exactly the truncated-CL signal that this PR's chaos test exercises.
// Fanout reads the flag via Tripped() and disqualifies the slot.
//
// Non-ALB callers never bind one to the context, so the proxy's lookup
// returns nil and the path is a silent no-op.
type UpstreamShortReadCapture struct {
	tripped atomic.Bool
}

// Mark sets the short-read flag. Safe for concurrent use.
func (c *UpstreamShortReadCapture) Mark() {
	c.tripped.Store(true)
}

// Tripped reports whether the proxy marked a short read.
func (c *UpstreamShortReadCapture) Tripped() bool {
	return c.tripped.Load()
}

type shortReadKey struct{}

// WithUpstreamShortReadCapture binds a fresh capture to ctx and returns
// the new context plus the capture pointer. Callers retain the pointer
// to read Tripped() after the handler returns.
func WithUpstreamShortReadCapture(ctx context.Context) (context.Context, *UpstreamShortReadCapture) {
	c := &UpstreamShortReadCapture{}
	return context.WithValue(ctx, shortReadKey{}, c), c
}

// GetUpstreamShortReadCapture returns the capture bound to ctx, or nil
// if no capture is bound (non-ALB context).
func GetUpstreamShortReadCapture(ctx context.Context) *UpstreamShortReadCapture {
	c, _ := ctx.Value(shortReadKey{}).(*UpstreamShortReadCapture)
	return c
}

// RebindUpstreamShortReadCapture copies the capture (if any) from src
// onto dst, returning the (possibly modified) dst. cloneRequestWithSpan
// uses this to preserve the sidecar across the proxy engine's internal
// request clone, which intentionally builds a fresh context.Background()
// and would otherwise drop the capture set by the ALB fanout layer.
func RebindUpstreamShortReadCapture(dst, src context.Context) context.Context {
	if c := GetUpstreamShortReadCapture(src); c != nil {
		return context.WithValue(dst, shortReadKey{}, c)
	}
	return dst
}
