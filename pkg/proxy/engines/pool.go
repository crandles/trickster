package engines

import (
	"bytes"
	"sync"
)

const maxMarshalBufSize = 1 << 20 // 1MB

var marshalBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

func getMarshalBuf() []byte {
	bp := marshalBufPool.Get().(*[]byte)
	return (*bp)[:0]
}

func putMarshalBuf(b []byte) {
	if cap(b) > maxMarshalBufSize {
		return
	}
	marshalBufPool.Put(&b)
}

var decompBufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func getDecompBuf() *bytes.Buffer {
	buf := decompBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putDecompBuf(buf *bytes.Buffer) {
	if buf == nil || buf.Cap() > maxMarshalBufSize {
		return
	}
	buf.Reset()
	decompBufPool.Put(buf)
}
