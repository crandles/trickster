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

package index

import "sync"

const maxObjectMsgBufSize = 1 << 20 // 1MB

var objectPool = sync.Pool{
	New: func() any {
		return &Object{}
	},
}

func getObject() *Object {
	o := objectPool.Get().(*Object)
	o.Key = ""
	o.Value = nil
	o.Size = 0
	o.ReferenceValue = nil
	return o
}

func putObject(o *Object) {
	if o == nil {
		return
	}
	o.Value = nil
	o.ReferenceValue = nil
	objectPool.Put(o)
}

var objectMsgBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

func getObjectMsgBuf() []byte {
	bp := objectMsgBufPool.Get().(*[]byte)
	return (*bp)[:0]
}

func putObjectMsgBuf(b []byte) {
	if cap(b) > maxObjectMsgBufSize {
		return
	}
	objectMsgBufPool.Put(&b)
}
