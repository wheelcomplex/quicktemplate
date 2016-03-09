package quicktemplate

import (
	"bytes"
	"os"
	"testing"
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

	// missing endfunc
	testParseFailure(t, "{%func a() %}aaaa")

	// empty func name
	testParseFailure(t, "{% func () %}aaa{% endfunc %}")
	testParseFailure(t, "{% func (a int, b string) %}aaa{% endfunc %}")

	// empty func arguments
	testParseFailure(t, "{% func aaa %}aaa{% endfunc %}")

	// empty if condition
	testParseFailure(t, "{% func a() %}{% if    %}aaaa{% endif %}{% endfunc %}")

	// missing endif
	testParseFailure(t, "{%func a() %}{%if foo %}aaa{% endfunc %}")

	// missing endfor
	testParseFailure(t, "{%func a()%}{%for %}aaa{%endfunc%}")

	// break outside for
	testParseFailure(t, "{%func a()%}{%break%}{%endfunc%}")
}

func TestParserSuccess(t *testing.T) {
	// empty template
	testParseSuccess(t, "")

	// template without code and funcs
	testParseSuccess(t, "foobar\nbaz")

	// template with code
	testParseSuccess(t, "{%code var a struct {}\nconst n = 123%}")

	// func
	testParseSuccess(t, "{%func a()%}{%endfunc%}")

	// func with with condition
	testParseSuccess(t, "{%func a(x bool)%}{%if x%}foobar{%endif%}{%endfunc%}")

	// for
	testParseSuccess(t, "{%func a()%}{%for%}aaa{%endfor%}{%endfunc%}")

	// return
	testParseSuccess(t, "{%func a()%}{%return%}{%endfunc%}")

	// nested for
	testParseSuccess(t, "{%func a()%}{%for i := 0; i < 10; i++ %}{%for j := 0; j < i; j++%}aaa{%endfor%}{%endfor%}{%endfunc%}")
}

func testParseFailure(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	w := &bytes.Buffer{}
	if err := parse(w, r, "memory/foobar.tpl"); err == nil {
		t.Fatalf("expecting error when parsing %q", str)
	}
}

func testParseSuccess(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	w := &bytes.Buffer{}
	if err := parse(w, r, "memory/foobar.tpl"); err != nil {
		t.Fatalf("unexpected error when parsing %q: %s", str, err)
	}
}

func TestParseFile(t *testing.T) {
	filename := "parser_test_template.tpl"
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("cannot open file %q: %s", filename, err)
	}
	defer f.Close()

	w := AcquireByteBuffer()
	if err := parse(w, f, filename); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	t.Fatalf("result\n%s\n", w.B)
	ReleaseByteBuffer(w)
}
