package quicktemplate

import (
	"sync"
)

type ByteBuffer struct {
	b []byte
}

func (bb *ByteBuffer) Write(p []byte) (int, error) {
	bb.b = append(bb.b, p...)
	return len(p), nil
}

func (bb *ByteBuffer) Bytes() []byte {
	return bb.b
}

func (bb *ByteBuffer) Reset() {
	bb.b = bb.b[:0]
}

func AcquireByteBuffer() *ByteBuffer {
	v := byteBufferPool.Get()
	if v == nil {
		return &ByteBuffer{}
	}
	return v.(*ByteBuffer)
}

func ReleaseByteBuffer(bb *ByteBuffer) {
	bb.Reset()
	byteBufferPool.Put(bb)
}

var byteBufferPool sync.Pool
