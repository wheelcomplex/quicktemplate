package quicktemplate

import (
	"testing"
)

func TestWriter(t *testing.T) {
	bb := AcquireByteBuffer()
	qw := AcquireWriter(bb)
	w := qw.W()
	bbNew, ok := w.(*ByteBuffer)
	if !ok {
		t.Fatalf("W() must return ByteBuffer, not %T", w)
	}
	if bbNew != bb {
		t.Fatalf("unexpected ByteBuffer returned: %p. Expecting %p", bbNew, bb)
	}

	wn := qw.N()
	we := qw.E()

	wn.S("<a></a>")
	wn.D(123)
	wn.Z([]byte("'"))
	wn.Q("foo")
	wn.J("ds")
	wn.F(1.23)
	we.V(struct{}{})

	we.S("<a></a>")
	we.D(321)
	we.Z([]byte("'"))
	we.Q("foo")
	we.J("ds")
	we.F(1.23)
	we.V(struct{}{})

	ReleaseWriter(qw)

	expectedS := `<a></a>123'"foo"ds1.23{}&lt;a&gt;&lt;/a&gt;321&#39;&quot;foo&quot;ds1.23{}`
	if string(bb.B) != expectedS {
		t.Fatalf("unexpected output: %q. Expecting %q", bb.B, expectedS)
	}

	ReleaseByteBuffer(bb)
}
