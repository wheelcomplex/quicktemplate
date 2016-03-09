package quicktemplate

import (
	"bytes"
	"testing"
)

func TestParseFailure(t *testing.T) {
	// unexpected tag
	testParseFailure(t, "{% foobar %}")
	testParseFailure(t, "aaa{% for %}bbb")

	// empty func name
	testParseFailure(t, "{% func () %}aaa{% endfunc %}")
	testParseFailure(t, "{% func (a int, b string) %}aaa{% endfunc %}")

	// empty if condition
	testParseFailure(t, "{% if    %}aaaa{% endif %}")
}

func testParseFailure(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	w := &bytes.Buffer{}
	if err := Parse(w, r, "memory"); err == nil {
		t.Fatalf("expecting error when parsing %q", str)
	}
}

func TestParse(t *testing.T) {
	s := `
this is a sample template
{% code
import (
	"foo"
	"bar"
)
%}

{% stripspace %}

this is a sample func
{% func foobar (  s string , 
 x int, a *Foo ) %}
	{%comment%}this %}{% is a comment{%endcomment%}
	he` + "`" + `llo, {%s s %}
	{% code panic("foobar") %} aaa {% return %}
	{% plain %}
		aaa {% ` + "`" + `foo %} bar
	{% endplain %}
	{% for _, c := range s %}
		c = {%d= c %}
		{% if c == 'a' %}
			break {% break %}
		{% elseif c == 'b' %}
			return {% return %}
		{% else %}
			{%= echo(s) %}
		{% endif %}
	{% endfor %}
bbb
{% endfunc %}

{% func echo(s string) %}
	s={%s s %}
{% endfunc %}

{% endstripspace %}

this is a tail`

	r := bytes.NewBufferString(s)
	w := &bytes.Buffer{}
	if err := Parse(w, r, "memory"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	t.Fatalf("result\n%s\n", w.Bytes())
}
