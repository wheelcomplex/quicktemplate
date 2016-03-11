package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/valyala/quicktemplate"
)

func TestParseFailure(t *testing.T) {
	// unknown tag
	testParseFailure(t, "{% foobar %}")

	// unexpected tag outside func
	testParseFailure(t, "aaa{% for %}bbb{%endfor%}")
	testParseFailure(t, "{% return %}")
	testParseFailure(t, "{% break %}")
	testParseFailure(t, "{% if 1==1 %}aaa{%endif%}")
	testParseFailure(t, "{%s abc %}")
	testParseFailure(t, "{%v= aaaa(xx) %}")
	testParseFailure(t, "{%= a() %}")

	// import after func and/or code
	testParseFailure(t, `{%code var i = 0 %}{%import "fmt"%}`)
	testParseFailure(t, `{%func f()%}{%endfunc%}{%import "fmt"%}`)

	// missing endfunc
	testParseFailure(t, "{%func a() %}aaaa")

	// empty func name
	testParseFailure(t, "{% func () %}aaa{% endfunc %}")
	testParseFailure(t, "{% func (a int, b string) %}aaa{% endfunc %}")

	// empty func arguments
	testParseFailure(t, "{% func aaa %}aaa{% endfunc %}")

	// func with anonymous argument
	testParseFailure(t, "{% func a(x int, string) %}{%endfunc%}")

	// func with incorrect arguments' list
	testParseFailure(t, "{% func x(foo, bar) %}{%endfunc%}")
	testParseFailure(t, "{% func x(foo bar baz) %}{%endfunc%}")

	// empty if condition
	testParseFailure(t, "{% func a() %}{% if    %}aaaa{% endif %}{% endfunc %}")

	// missing endif
	testParseFailure(t, "{%func a() %}{%if foo %}aaa{% endfunc %}")

	// missing endfor
	testParseFailure(t, "{%func a()%}{%for %}aaa{%endfunc%}")

	// break outside for
	testParseFailure(t, "{%func a()%}{%break%}{%endfunc%}")

	// invalid if condition
	testParseFailure(t, "{%func a()%}{%if a = b %}{%endif%}{%endfunc%}")
	testParseFailure(t, "{%func f()%}{%if a { %}{%endif%}{%endfunc%}")

	// invalid for
	testParseFailure(t, "{%func a()%}{%for a = b %}{%endfor%}{%endfunc%}")
	testParseFailure(t, "{%func f()%}{%for { %}{%endfor%}{%endfunc%}")

	// invalid code inside func
	testParseFailure(t, "{%func f()%}{%code } %}{%endfunc%}")
	testParseFailure(t, "{%func f()%}{%code { %}{%endfunc%}")
}

func TestParserSuccess(t *testing.T) {
	// empty template
	testParseSuccess(t, "")

	// template without code and funcs
	testParseSuccess(t, "foobar\nbaz")

	// template with code
	testParseSuccess(t, "{%code var a struct {}\nconst n = 123%}")

	// import
	testParseSuccess(t, `{%import "foobar"%}`)
	testParseSuccess(t, `{% import (
	"foo"
	"bar"
)%}`)
	testParseSuccess(t, `{%import "foo"%}{%import "bar"%}`)

	// func
	testParseSuccess(t, "{%func a()%}{%endfunc%}")

	// func with with condition
	testParseSuccess(t, "{%func a(x bool)%}{%if x%}foobar{%endif%}{%endfunc%}")

	// func with complex arguments
	testParseSuccess(t, "{%func f(h1, h2 func(x, y int) string, d int)%}{%endfunc%}")

	// for
	testParseSuccess(t, "{%func a()%}{%for%}aaa{%endfor%}{%endfunc%}")

	// return
	testParseSuccess(t, "{%func a()%}{%return%}{%endfunc%}")

	// nested for
	testParseSuccess(t, "{%func a()%}{%for i := 0; i < 10; i++ %}{%for j := 0; j < i; j++%}aaa{%endfor%}{%endfor%}{%endfunc%}")

	// plain containing arbitrary tags
	testParseSuccess(t, "{%func f()%}{%plain%}This {%endfunc%} is ignored{%endplain%}{%endfunc%}")

	// comment with arbitrary tags
	testParseSuccess(t, "{%func f()%}{%comment%}This {%endfunc%} is ignored{%endcomment%}{%endfunc%}")

	// complex if
	testParseSuccess(t, "{%func a()%}{%if n, err := w.Write(p); err != nil %}{%endif%}{%endfunc%}")

	// complex for
	testParseSuccess(t, "{%func a()%}{%for i, n := 0, len(s); i < n && f(i); i++ %}{%endfor%}{%endfunc%}")

	// complex code inside func
	testParseSuccess(t, `{%func f()%}{%code
		type A struct{}
		var aa []A
		for i := 0; i < 10; i++ {
			aa = append(aa, &A{})
			if i == 42 {
				break
			}
		}
		return
	%}{%endfunc%}`)

	// break inside for loop
	testParseSuccess(t, `{%func f()%}{%for%}{%code
		if a() {
			break
		} else {
			return
		}
	%}{%endfor%}{%endfunc%}`)
}

func testParseFailure(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	w := &bytes.Buffer{}
	if err := parse(w, r, "memory/foobar.tpl", "memory"); err == nil {
		t.Fatalf("expecting error when parsing %q", str)
	}
}

func testParseSuccess(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	w := &bytes.Buffer{}
	if err := parse(w, r, "memory/foobar.tpl", "memory"); err != nil {
		t.Fatalf("unexpected error when parsing %q: %s", str, err)
	}
}

func TestParseFile(t *testing.T) {
	filename := "templates/test.qtpl"
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("cannot open file %q: %s", filename, err)
	}
	defer f.Close()

	packageName, err := getPackageName(filename)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	w := quicktemplate.AcquireByteBuffer()
	if err := parse(w, f, filename, packageName); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expectedFilename := filename + ".compiled"
	data, err := ioutil.ReadFile(expectedFilename)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !bytes.Equal(w.B, data) {
		t.Fatalf("unexpected output: %q. Expecting %q", w.B, data)
	}

	quicktemplate.ReleaseByteBuffer(w)
}
