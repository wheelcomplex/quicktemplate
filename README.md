[![Build Status](https://travis-ci.org/valyala/quicktemplate.svg)](https://travis-ci.org/valyala/quicktemplate)
[![GoDoc](https://godoc.org/github.com/valyala/quicktemplate?status.svg)](http://godoc.org/github.com/valyala/quicktemplate)
[![Coverage](http://gocover.io/_badge/github.com/valyala/quicktemplate)](http://gocover.io/github.com/valyala/quicktemplate)
[![Go Report](http://goreportcard.com/badge/valyala/quicktemplate)](http://goreportcard.com/report/valyala/quicktemplate)

# quicktemplate
Fast, powerful, yet easy to use html template engine for Go.

# Features

  * [Extremely fast](#performance-comparison-with-htmltemplate).
    Templates are converted into Go code and then compiled.
  * Easy to use. See [quickstart](#quick-start) and [examples](https://github.com/valyala/quicktemplate/tree/master/examples)
    for details.
  * Powerful. Arbitrary Go code may be embedded into and mixed with templates.
    Be careful with this power - do not query db and/or external resources from
    templates :)
  * Templates are compiled into a single binary, so there is no need in copying
    template files to the server.

# Drawbacks

  * Templates cannot be updated on the fly on the server, since they
    are compiled into a single binary.

# Performance comparison with html/template

Quicktemplate is more than 20x faster than [html/template](https://golang.org/pkg/html/template/).
The following simple template is used:

  * [html/template version](https://github.com/valyala/quicktemplate/blob/master/testdata/templates/bench.tpl)
  * [quicktemplate version](https://github.com/valyala/quicktemplate/blob/master/testdata/templates/bench.qtpl)

Benchmark results:

```
$ go test -bench=Template -benchmem
BenchmarkQuickTemplate1-4  	10000000	       158 ns/op	       0 B/op	       0 allocs/op
BenchmarkQuickTemplate10-4 	 2000000	       604 ns/op	       0 B/op	       0 allocs/op
BenchmarkQuickTemplate100-4	  300000	      5498 ns/op	       0 B/op	       0 allocs/op
BenchmarkHTMLTemplate1-4   	  500000	      2807 ns/op	     752 B/op	      23 allocs/op
BenchmarkHTMLTemplate10-4  	  100000	     13527 ns/op	    3521 B/op	     117 allocs/op
BenchmarkHTMLTemplate100-4 	   10000	    133503 ns/op	   34499 B/op	    1152 allocs/op
```

# Security

By default all the template placeholders are html-escaped.

# Examples

See [examples](https://github.com/valyala/quicktemplate/tree/master/examples).

# Quick start

Let's start with a minimal template example:

```qtpl
Hello is a simple template function.
{% func Hello(name string) %}
	Hello, {%s name %}!
{% endfunc %}
```

Save this file into `templates` folder under the name `hello.qtpl`
and run [qtc](https://github.com/valyala/quicktemplate/tree/master/qtc)
inside this folder. `qtc` may be installed by issuing:

```
go get -u github.com/valyala/quicktemplate/qtc
```

If all went ok, `hello.qtpl.go` file must appear in the `templates` folder.
This file contains Go code for `hello.qtpl`. Let's use it!

Create a file main.go outside `templates` folder and put the following
code there:

```go
package main

import (
	"fmt"

	"./templates"
)

func main() {
	fmt.Printf("%s\n", templates.Hello("Foo"))
	fmt.Printf("%s\n", templates.Hello("Bar"))
}
```

Then run `go run`. If all went ok, you'll see something like this:

```

	Hello, Foo!


	Hello, Bar!

```

Let's create more complex template, which calls other template functions,
contains loops, conditions, breaks and returns.
Put the following template into `templates/greetings.qtpl`:

```qtpl

Greetings greets up to 42 names.
It also greets John differently comparing to others.
{% func Greetings(names []string) %}
	{% if len(names) == 0 %}
		Nobody to greet :(
		{% return %}
	{% endif %}

	{% for i, name := range names %}
		{% if i == 42 %}
			I'm tired to greet so many people...
			{% break %}
		{% elseif name == "John" %}
			{%= sayHi("John") %}
		{% else %}
			{%= Hello(name) %}
		{% endif %}
	{% endfor %}
{% endfunc %}

sayHi is unexported, since it starts with lowercase letter.
{% func sayHi(name string) %}
	Hi, {%s name %}
{% endfunc %}
```

Run `qtc` inside `templates` folder - it should create `greetings.qtpl.go`.
Now put the following code into `main.go`:

```go
package main

import (
	"bytes"
	"fmt"

	"./templates"
)

func main() {
	names := []string{"Kate", "Go", "John", "Brad"}

	// qtc creates Write* function for each template function.
	// Such functions accept io.Writer as first parameter:
	var buf bytes.Buffer
	templates.WriteGreetings(&buf, names)

	fmt.Printf("buf=\n%s", buf.Bytes())
}
```

Careful readers may notice different output tags were used in these
templates: `{%s name %}` and `{%= Hello(name) %}`. What's the difference?
The `{%s x %}` is used for printing html-safe strings, while `{%= F() %}`
is used for embedding template function calls. Quicktemplate supports also
other output tags:

  * `{%d num %}` for integers
  * `{%f float %}` for float64
  * `{%z bytes %}` for byte slices
  * `{%q str %}` for json-compatible quoted strings.
  * `{%v anything %}` is equivalent to `%v` in printf-like functions.

All these output tags produce html-safe output, i.e. they escape `<` to `&lt;`,
`>` to `&gt;`, etc. If you don't want html-safe output, then just put `=` after
the tag. For example: `{%s= "<h1>This h1 won't be escaped</h1>" %}`.

As you may notice `{%= F() %}` and `{%s= F() %}` produce the same output for `{% func F() %}`.
But the first one is optimized for speed - it avoids memory allocations and copy.
So stick to it when embedding template function calls.

All the ouptut tags except of `{%= F() %}` may contain arbitrary valid
Go expression instead of just identifier. For example:

```qtpl
Import fmt for fmt.Sprintf()
{% import "fmt" %}

FmtFunc uses fmt.Sprintf() inside output tag
{% func FmtFunc(s string) %}
	{%s fmt.Sprintf("FmtFunc accepted %q string", s) %}
{% endfunc %}
```

There are other useful tags supported by quicktemplate:

  * `{% comment %}This is a comment and won't trap into the output{% endcomment %}`
  * `{% plain %}This will {% trap into %} {% the %} output {% unmodified %}{% endplain %}`
  * `{% collapsespace %}   Space between   {%s " tags " %}   will be collapsed.  {% endcollapsespace %}`
  * `{% collapsespace %}{% space %}{% newline %}This is prepended with space and newline   {% endcollapsespace %}`
  * `{% code %}`:

    ```qtpl
    {% code
    // arbitrary Go code may be embedded here!
    type FooArg struct {
        Name string
        Age int
    }
    %}
    ```

  * `{% import %}`:

    ```qtpl
    Import external packages.
    {% import "foo/bar" %}
    {% import (
        "foo"
        bar "baz/baa"
    ) %}
    ```

  * `{% interface %}`:

    ```qtpl
    Interfaces allow powerful templates' inheritance.
    {%
    interface Page {
        Title()
        Body(s string, n int)
        Footer()
    }
    %}
    ```

    See [basicserver example](https://github.com/valyala/quicktemplate/tree/master/examples/basicserver)
    for details.
