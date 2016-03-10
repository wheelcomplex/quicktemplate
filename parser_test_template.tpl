This is a test template file.
All the lines outside func and code are just comments.

Optional imports must be at the top of template
{% import (
	"fmt"
	"strconv"
)
%}

Arbitrary go code may be inserted here. For instance, type definition:
{% code
type FooArgs struct {
	S string
	N int	
}
%}

Now define an exported function template
{% func Foo(a []FooArgs) %}
	<h1>Hello, I'm Foo!</h1>
	<div>
		My args are:
		{% if len(a) == 0 %}
			no args!
		{% elseif len(a) == 1 %}
			a single arg: {%= printArgs(0, a[0]) %}
		{% else %}
			<ul>
			{% for i, aa := range a %}
				{% if i >= 42 %}
					There are other args, but only the first 42 of them are shown
					{% break %}
				{% endif %}
				{%= printArgs(i, aa) %}
				Arbitrary Go code may be inserted here: {% code	str := strconv.Itoa(i+42) %}
				str = {%s fmt.Sprintf("this html will be escaped <b>%s</b>", str) %}
			{% endfor %}
			</ul>
		{% endif %}
	</div>
	{% plain %}
		Arbitrary tags are treated as plaintext inside plain.
		For instance, {% foo %} {% bar %} {% for %}
		{% func %} {% code %} {% return %} {% break %} {% comment %}
		and even {% unclosed tag
	{% endplain %}
	{% collapsespace %}
		Leading and trailing space between template tags is collapsed
		inside collapsespace unless {%space%} is used: {%space%}
	{% endcollapsespace %}
{% endfunc %}

Now define private printArgs, which is used in Foo
{% func printArgs(i int, a FooArgs) %}
	{% if i == 0 %}
		Hide args for i = 0
		{% return %}
	{% endif %}
	<li>
		a[{%d i %}] = {S: {%q a.S %}, N: {%d a.N %}}<br>
	</li>
{% endfunc %}


unused code may be commented:
{% comment %}
{% func UnusedFunc(n int) %}
	foobar
{% endfunc %}
{% endcomment %}
