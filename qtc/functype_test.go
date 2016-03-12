package main

import (
	"testing"
)

func TestParseFuncCallSuccess(t *testing.T) {
	// func without args
	testParseFuncCallSuccess(t, "f()", "streamf(qw)")

	// func with args
	testParseFuncCallSuccess(t, "Foo(a, b)", "streamFoo(qw, a, b)")

	// method without args
	testParseFuncCallSuccess(t, "a.f()", "a.streamf(qw)")

	// method with args
	testParseFuncCallSuccess(t, "a.f(xx)", "a.streamf(qw, xx)")

	// chained method
	testParseFuncCallSuccess(t, "foo.bar.Baz(x, y)", "foo.bar.streamBaz(qw, x, y)")

	// complex args
	testParseFuncCallSuccess(t, `as.ffs.SS(
		func(x int, y string) {
			panic("foobar")
		},
		map[string]int{
			"foo":1,
			"bar":2,
		},
		qwe)`,
		`as.ffs.streamSS(qw, 
		func(x int, y string) {
			panic("foobar")
		},
		map[string]int{
			"foo":1,
			"bar":2,
		},
		qwe)`)
}

func TestParseFuncCallFailure(t *testing.T) {
	testParseFuncCallFailure(t, "")

	// non-func
	testParseFuncCallFailure(t, "foobar")
	testParseFuncCallFailure(t, "a, b, c")
	testParseFuncCallFailure(t, "{}")
	testParseFuncCallFailure(t, "(a)")
	testParseFuncCallFailure(t, "(f())")

	// inline func
	testParseFuncCallFailure(t, "func() {}()")
	testParseFuncCallFailure(t, "func a() {}()")

	// nonempty tail after func call
	testParseFuncCallFailure(t, "f(); f1()")
	testParseFuncCallFailure(t, "f()\nf1()")
	testParseFuncCallFailure(t, "f()\n for {}")
}

func testParseFuncCallFailure(t *testing.T, s string) {
	_, err := parseFuncCall([]byte(s))
	if err == nil {
		t.Fatalf("expecting non-nil error when parsing %q", s)
	}
}

func testParseFuncCallSuccess(t *testing.T, s, callStream string) {
	f, err := parseFuncCall([]byte(s))
	if err != nil {
		t.Fatalf("unexpected error when parsing %q: %s", s, err)
	}
	cs := f.CallStream("qw")
	if cs != callStream {
		t.Fatalf("unexpected CallStream: %q. Expecting %q. s=%q", cs, callStream, s)
	}
}

func TestParseFuncDefSuccess(t *testing.T) {
	// private func without args
	testParseFuncDefSuccess(t, "xx()", "xx() string",
		"streamxx(qw *quicktemplate.Writer)", "streamxx(qw)",
		"writexx(qww io.Writer)", "writexx(qww)")

	// public func with a single arg
	testParseFuncDefSuccess(t, "F(a int)", "F(a int) string",
		"streamF(qw *quicktemplate.Writer, a int)", "streamF(qw, a)",
		"WriteF(qww io.Writer, a int)", "WriteF(qww, a)")

	// public method without args
	testParseFuncDefSuccess(t, "(f *foo) M()", "(f *foo) M() string",
		"(f *foo) streamM(qw *quicktemplate.Writer)", "f.streamM(qw)",
		"(f *foo) WriteM(qww io.Writer)", "f.WriteM(qww)")

	// private method with three args
	testParseFuncDefSuccess(t, "(f *Foo) bar(x, y string, z int)", "(f *Foo) bar(x, y string, z int) string",
		"(f *Foo) streambar(qw *quicktemplate.Writer, x, y string, z int)", "f.streambar(qw, x, y, z)",
		"(f *Foo) writebar(qww io.Writer, x, y string, z int)", "f.writebar(qww, x, y, z)")

	// method with complex args
	testParseFuncDefSuccess(t, "(t TPL) Head(h1, h2 func(x, y int), h3 map[int]struct{})", "(t TPL) Head(h1, h2 func(x, y int), h3 map[int]struct{}) string",
		"(t TPL) streamHead(qw *quicktemplate.Writer, h1, h2 func(x, y int), h3 map[int]struct{})", "t.streamHead(qw, h1, h2, h3)",
		"(t TPL) WriteHead(qww io.Writer, h1, h2 func(x, y int), h3 map[int]struct{})", "t.WriteHead(qww, h1, h2, h3)")
}

func TestParseFuncDefFailure(t *testing.T) {
	testParseFuncDefFailure(t, "")

	// invalid syntax
	testParseFuncDefFailure(t, "foobar")
	testParseFuncDefFailure(t, "f() {")
	testParseFuncDefFailure(t, "for {}")

	// missing func name
	testParseFuncDefFailure(t, "()")
	testParseFuncDefFailure(t, "(a int, b string)")

	// missing method name
	testParseFuncDefFailure(t, "(x XX) ()")
	testParseFuncDefFailure(t, "(x XX) (y, z string)")

	// reserved variable name
	testParseFuncDefFailure(t, "f(qww []byte)")
	testParseFuncDefFailure(t, "f(qw int)")
	testParseFuncDefFailure(t, "(qs *Soo) f()")
	testParseFuncDefFailure(t, "(qb Boo) f()")
	testParseFuncDefFailure(t, "(x XX) f(qww int, qw string)")

	// func with return values
	testParseFuncDefFailure(t, "f() string")
	testParseFuncDefFailure(t, "f() (int, string)")
	testParseFuncDefFailure(t, "(x XX) f() string")
	testParseFuncDefFailure(t, "(x XX) f(a int) (int, string)")
}

func testParseFuncDefFailure(t *testing.T, s string) {
	f, err := parseFuncDef([]byte(s))
	if err == nil {
		t.Fatalf("expecting error when parsing %q. got %#v", s, f)
	}
}

func testParseFuncDefSuccess(t *testing.T, s, defString, defStream, callStream, defWrite, callWrite string) {
	f, err := parseFuncDef([]byte(s))
	if err != nil {
		t.Fatalf("cannot parse %q: %s", s, err)
	}
	ds := f.DefString()
	if ds != defString {
		t.Fatalf("unexpected DefString: %q. Expecting %q. s=%q", ds, defString, s)
	}
	ds = f.DefStream("qw")
	if ds != defStream {
		t.Fatalf("unexpected DefStream: %q. Expecting %q. s=%q", ds, defStream, s)
	}
	cs := f.CallStream("qw")
	if cs != callStream {
		t.Fatalf("unexpected CallStream: %q. Expecting %q. s=%q", cs, callStream, s)
	}
	dw := f.DefWrite("qww")
	if dw != defWrite {
		t.Fatalf("unexpected DefWrite: %q. Expecting %q. s=%q", dw, defWrite, s)
	}
	cw := f.CallWrite("qww")
	if cw != callWrite {
		t.Fatalf("unexpected CallWrite: %q. Expecting %q. s=%q", cw, callWrite, s)
	}
}
