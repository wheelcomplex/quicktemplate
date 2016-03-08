package quicktemplate

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

func Parse(w io.Writer, r io.Reader) {
	s := NewScanner(r)
	parseTemplate(s, w)
}

func parseTemplate(s *Scanner, w io.Writer) {
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			// just skip top-level text
		case TagName:
			switch string(t.Value) {
			case "code":
				parseCode(s, w, "")
			case "func":
				parseFunc(s, w)
			default:
				log.Fatalf("unexpected tag found outside func: %s at %s", t.Value, s.Context())
			}
		default:
			log.Fatalf("unexpected token found %s when parsing template at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse template: %s", err)
	}
}

func parseFunc(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fname, fargs, fargsNoTypes := parseFnameFargs(s, t.Value)
	emitFuncStart(w, fname, fargs)
	prefix := "\t"
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			emitText(w, t.Value, prefix)
		case TagName:
			switch string(t.Value) {
			case "endfunc":
				expectTagContents(s)
				emitFuncEnd(w, fname, fargs, fargsNoTypes)
				return
			case "s":
				parseS(s, w, prefix)
			case "d":
				parseD(s, w, prefix)
			case "f":
				parseF(s, w, prefix)
			case "code":
				parseCode(s, w, prefix)
			case "return":
				parseReturn(s, w, prefix)
			case "for":
				parseFor(s, w, prefix)
			default:
				log.Fatalf("unexpected tag found inside func: %s at %s", t.Value, s.Context())
			}
		default:
			log.Fatalf("unexpected token found %s when parsing func at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse func: %s", err)
	}
}

func parseFor(s *Scanner, w io.Writer, prefix string) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sfor %s {\n", prefix, t.Value)
	prefix += "\t"
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			emitText(w, t.Value, prefix)
		case TagName:
			switch string(t.Value) {
			case "endfor":
				expectTagContents(s)
				fmt.Fprintf(w, "%s}\n", prefix[1:])
				return
			case "s":
				parseS(s, w, prefix)
			case "d":
				parseD(s, w, prefix)
			case "f":
				parseF(s, w, prefix)
			case "code":
				parseCode(s, w, prefix)
			case "return":
				parseReturn(s, w, prefix)
			case "for":
				parseFor(s, w, prefix)
			default:
				log.Fatalf("unexpected tag found inside for loop: %s at %s", t.Value, s.Context())
			}
		default:
			log.Fatalf("unexpected token found %s when parsing for loop at %s", t, s.Context())
		}
	}
}

func parseReturn(s *Scanner, w io.Writer, prefix string) {
	expectTagContents(s)
	fmt.Fprintf(w, "%squicktemplate.ReleaseWriter(qw)\n", prefix)
	fmt.Fprintf(w, "%sreturn\n", prefix)
}

func emitFuncStart(w io.Writer, fname, fargs string) {
	fmt.Fprintf(w, `
func %sStream(w *io.Writer, %s) {
	qw := quicktemplate.AcquireWriter(w)
`,
		fname, fargs)
}

func emitFuncEnd(w io.Writer, fname, fargs, fargsNoTypes string) {
	fmt.Fprintf(w, `
	quicktemplate.ReleaseWriter(qw)
}

func %s(%s) string {
	bb := quicktemplate.AcquireByteBuffer()
	%sStream(bb, %s)
	s := string(bb.Bytes())
	quicktemplate.ReleaseByteBuffer(bb)
	return s
}`,
		fname, fargs, fname, fargsNoTypes)
}

func parseFnameFargs(s *Scanner, f []byte) (string, string, string) {
	// TODO: use real Go parser here

	n := bytes.IndexByte(f, '(')
	if n < 0 {
		log.Fatalf("missing '(' for function arguments at %s", s.Context())
	}
	fname := string(stripTrailingSpace(f[:n]))

	f = f[n+1:]
	n = bytes.LastIndexByte(f, ')')
	if n < 0 {
		log.Fatalf("missing ')' for function arguments at %s", s.Context())
	}
	fargs := string(f[:n])

	var args []string
	for _, a := range strings.Split(fargs, ",") {
		a = string(stripLeadingSpace([]byte(a)))
		n = 0
		for n < len(a) && !isSpace(a[n]) {
			n++
		}
		args = append(args, a[:n])
	}
	fargsNoTypes := strings.Join(args, ", ")
	return fname, fargs, fargsNoTypes
}

func parseCode(s *Scanner, w io.Writer, prefix string) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%s%s\n", prefix, t.Value)
}

func parseS(s *Scanner, w io.Writer, prefix string) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sqw.E.S(%s)\n", prefix, t.Value)
}

func parseD(s *Scanner, w io.Writer, prefix string) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sqw.D(%s)\n", prefix, t.Value)
}

func parseF(s *Scanner, w io.Writer, prefix string) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sqw.F(%s)\n", prefix, t.Value)
}

func emitText(w io.Writer, text []byte, prefix string) {
	for len(text) > 0 {
		n := bytes.IndexByte(text, '`')
		if n < 0 {
			fmt.Fprintf(w, "%sqw.E.S(`%s`)\n", prefix, text)
			return
		}
		fmt.Fprintf(w, "%sqw.E.S(`%s`)\n", prefix, text[:n])
		fmt.Fprintf(w, "%sqw.E.S(\"`\")\n", prefix)
		text = text[n+1:]
	}
}

func expectTagContents(s *Scanner) *Token {
	return expectToken(s, TagContents)
}

func expectToken(s *Scanner, id int) *Token {
	if !s.Next() {
		log.Fatalf("cannot find token %s: %v", tokenIDToStr(id), s.LastError())
	}
	t := s.Token()
	if t.ID != id {
		log.Fatalf("unexpected token found %s. Expecting %s", t, tokenIDToStr(id))
	}
	return t
}
