package quicktemplate

import (
	"testing"
)

func BenchmarkQWriterVString(b *testing.B) {
	v := createTestS(100)
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.V(v)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterVInt(b *testing.B) {
	v := 1233455
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.V(v)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterQ(b *testing.B) {
	s := createTestS(100)
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.Q(s)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterJ(b *testing.B) {
	s := createTestS(100)
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.J(s)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterF(b *testing.B) {
	f := 123.456
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.F(f)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterD(b *testing.B) {
	n := 123456
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.D(n)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func BenchmarkQWriterZ1Byte(b *testing.B) {
	benchmarkQWriterZ(b, 1)
}

func BenchmarkQWriterZ10Bytes(b *testing.B) {
	benchmarkQWriterZ(b, 10)
}

func BenchmarkQWriterZ100Byte(b *testing.B) {
	benchmarkQWriterZ(b, 100)
}

func BenchmarkQWriterZ1KByte(b *testing.B) {
	benchmarkQWriterZ(b, 1000)
}

func BenchmarkQWriterZ10KByte(b *testing.B) {
	benchmarkQWriterZ(b, 10000)
}

func BenchmarkQWriterS1Byte(b *testing.B) {
	benchmarkQWriterS(b, 1)
}

func BenchmarkQWriterS10Bytes(b *testing.B) {
	benchmarkQWriterS(b, 10)
}

func BenchmarkQWriterS100Byte(b *testing.B) {
	benchmarkQWriterS(b, 100)
}

func BenchmarkQWriterS1KByte(b *testing.B) {
	benchmarkQWriterS(b, 1000)
}

func BenchmarkQWriterS10KByte(b *testing.B) {
	benchmarkQWriterS(b, 10000)
}

func benchmarkQWriterZ(b *testing.B, size int) {
	z := createTestZ(size)
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.Z(z)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func benchmarkQWriterS(b *testing.B, size int) {
	s := createTestS(size)
	b.RunParallel(func(pb *testing.PB) {
		var w QWriter
		bb := AcquireByteBuffer()
		w.w = bb
		for pb.Next() {
			w.S(s)
			bb.Reset()
		}
		ReleaseByteBuffer(bb)
	})
}

func createTestS(size int) string {
	return string(createTestZ(size))
}

func createTestZ(size int) []byte {
	var b []byte
	for i := 0; i < size; i++ {
		b = append(b, '0'+byte(i%10))
	}
	return b
}
