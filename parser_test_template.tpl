This is a test template file.
All the lines outside func and code are just comments.

Insert usual Go code. For instance, import certain packages
{% code
import (
	"fmt"
	"strconv"
)
%}

Now insert type definition
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
				{%= printArgs(i, aa) %}
			{% endfor %}
			</ul>
		{% endif %}
	</div>
	{% plain %}
		Arbitrary tags are treated as plaintext inside plain.
		For instance, {% foo %} {% bar %} {% for %} {% func %}
		and even unclosed tag {% unclosed
	{% endplain %}
	{% stripspace %}
		Space between template tags is collapsed inside stripspace.
	{% endstripspace %}
{% endfunc %}

Now define private printArgs, which is used in Foo
{% func printArgs(i int, a FooArgs) %}
	<li>
		a[{%d i %}] = {S: {%q aa.S %}, N: {%d aa.N %}}<br>
	</li>
{% endfunc %}


unused code may be commented:
{% comment %}
{% func UnusedFunc(n int) %}
	foobar
{% endfunc %}
{% endcomment %}
