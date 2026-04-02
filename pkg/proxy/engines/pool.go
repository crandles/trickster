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
