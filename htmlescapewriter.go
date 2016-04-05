package quicktemplate

import (
	"io"
	"sync"
)

func acquireHTMLEscapeWriter(w io.Writer) io.Writer {
	v := htmlEscapeWriterPool.Get()
	if v == nil {
		v = &htmlEscapeWriter{}
	}
	hw := v.(*htmlEscapeWriter)
	hw.w = w
	return hw
}

func releaseHTMLEscapeWriter(w io.Writer) {
	hw := w.(*htmlEscapeWriter)
	hw.w = nil
	htmlEscapeWriterPool.Put(hw)
}

var htmlEscapeWriterPool sync.Pool

type htmlEscapeWriter struct {
	w io.Writer
}

func (w *htmlEscapeWriter) Write(p []byte) (int, error) {
	i := 0
	ww := w.w
	var (
		b   []byte
		err error
		n   int
	)
	for j, c := range p {
		b = nil
		switch c {
		case '<':
			b = strLT
		case '>':
			b = strGT
		case '"':
			b = strQuot
		case '\'':
			b = strApos
		case '&':
			b = strAmp
		}
		if b != nil {
			if n, err = ww.Write(p[i:j]); err != nil {
				return i + n, err
			}
			if n, err = ww.Write(b); err != nil {
				return j, err
			}
			i = j + 1
		}
	}
	if n, err = ww.Write(p[i:]); err != nil {
		return i + n, err
	}
	return len(p), nil
}

var (
	strLT   = []byte("&lt;")
	strGT   = []byte("&gt;")
	strQuot = []byte("&quot;")
	strApos = []byte("&#39;")
	strAmp  = []byte("&amp;")
)
