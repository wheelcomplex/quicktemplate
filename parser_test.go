package quicktemplate

import (
	"bytes"
	"testing"
)

func TestParse(t *testing.T) {
	s := `
this is a sample template
{% code
import (
	"foo"
	"bar"
)
%}

this is a sample func
{% func foobar (  s string , 
 x int, a *Foo ) %}
	{%comment%}this %}{% is a comment{%endcomment%}
	he` + "`" + `llo, {%s s %}
	{% code panic("foobar") %} aaa {% return %}
	{% plain %}aaa {% ` + "`" + `foo %} bar{% endplain %}
	{% for _, c := range s %}
		c = {%d c %}
		{% if c == 'a' %}
			break {% break %}
		{% elseif c == 'b' %}
			return {% return %}
		{% else %}
			foobar
		{% endif %}
	{% endfor %}
bbb
{% endfunc %}

this is a tail`

	r := bytes.NewBufferString(s)
	w := &bytes.Buffer{}
	Parse(w, r)

	t.Fatalf("result\n%s\n", w.Bytes())
}
