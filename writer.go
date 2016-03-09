package quicktemplate

import (
	"fmt"
	"io"
	"strconv"
	"sync"
)

type Writer struct {
	e QWriter
	n QWriter
}

func (qw *Writer) W() io.Writer {
	return qw.n.w
}

func (qw *Writer) E() *QWriter {
	return &qw.e
}

func (qw *Writer) N() *QWriter {
	return &qw.n
}

func AcquireWriter(w io.Writer) *Writer {
	var qw *Writer
	v := writerPool.Get()
	if v == nil {
		qw = &Writer{}
	}
	qw = v.(*Writer)
	qw.e.w = acquireHTMLEscapeWriter(w)
	qw.n.w = w
	return qw
}

func ReleaseWriter(qw *Writer) {
	releaseHTMLEscapeWriter(qw.e.w)
	qw.e.w = nil
	qw.n.w = nil
	writerPool.Put(qw)
}

var writerPool sync.Pool

type QWriter struct {
	w io.Writer
}

func (w *QWriter) S(s string) {
	w.w.Write(unsafeStrToBytes(s))
}

func (w *QWriter) Z(z []byte) {
	w.w.Write(z)
}

func (w *QWriter) D(n int) {
	bb := AcquireByteBuffer()
	bb.b = strconv.AppendInt(bb.b, int64(n), 10)
	w.w.Write(bb.b)
	ReleaseByteBuffer(bb)
}

func (w *QWriter) F(f float64) {
	bb := AcquireByteBuffer()
	bb.b = strconv.AppendFloat(bb.b, f, 'f', -1, 64)
	w.w.Write(bb.b)
	ReleaseByteBuffer(bb)
}

func (w *QWriter) Q(s string) {
	bb := AcquireByteBuffer()
	bb.b = strconv.AppendQuote(bb.b, s)
	w.w.Write(bb.b)
	ReleaseByteBuffer(bb)
}

func (w *QWriter) V(v interface{}) {
	fmt.Fprintf(w.w, "%v", v)
}

func acquireHTMLEscapeWriter(w io.Writer) io.Writer {
	return w
}

func releaseHTMLEscapeWriter(w io.Writer) {
}
