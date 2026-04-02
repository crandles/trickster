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

package capture

import (
	"net/http"
	"sync"
)

const maxCaptureBodySize = 64 * 1024 // 64KB

var capturePool = sync.Pool{
	New: func() any {
		return &CaptureResponseWriter{
			header:     make(http.Header),
			statusCode: http.StatusOK,
		}
	},
}

// GetCaptureResponseWriter returns a reset CaptureResponseWriter from the pool
func GetCaptureResponseWriter() *CaptureResponseWriter {
	sw := capturePool.Get().(*CaptureResponseWriter)
	sw.statusCode = http.StatusOK
	sw.body.Reset()
	sw.len = 0
	// clear header map entries but keep the allocated map
	for k := range sw.header {
		delete(sw.header, k)
	}
	return sw
}

// PutCaptureResponseWriter returns a CaptureResponseWriter to the pool
func PutCaptureResponseWriter(sw *CaptureResponseWriter) {
	if sw == nil || sw.body.Cap() > maxCaptureBodySize {
		return
	}
	capturePool.Put(sw)
}
