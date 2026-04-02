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
