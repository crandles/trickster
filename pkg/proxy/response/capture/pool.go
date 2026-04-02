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
