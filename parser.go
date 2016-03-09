package quicktemplate

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

type parser struct {
	s        *Scanner
	w        io.Writer
	prefix   string
	forDepth int
}

func Parse(w io.Writer, r io.Reader) {
	p := &parser{
		s: NewScanner(r),
		w: w,
	}
	p.parseTemplate()
}

func (p *parser) parseTemplate() {
	s := p.s
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			// just skip top-level text
		case TagName:
			switch string(t.Value) {
			case "code":
				p.parseCode()
			case "func":
				p.parseFunc()
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

func (p *parser) parseFunc() {
	s := p.s
	t := expectTagContents(s)
	fname, fargs, fargsNoTypes := parseFnameFargs(s, t.Value)
	p.emitFuncStart(fname, fargs)
	p.prefix += "\t"
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			if p.tryParseCommonTags(t.Value) {
				continue
			}
			switch string(t.Value) {
			case "endfunc":
				skipTagContents(s)
				p.emitFuncEnd(fname, fargs, fargsNoTypes)
				p.prefix = p.prefix[1:]
				return
			default:
				log.Fatalf("unexpected tag found inside func: %s at %s", t.Value, s.Context())
			}
		default:
			log.Fatalf("unexpected token found %s when parsing func at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse func: %s", err)
	} else {
		log.Fatalf("cannot find endfunc tag at %s", s.Context())
	}
}

func (p *parser) parseFor() {
	s := p.s
	w := p.w
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sfor %s {\n", p.prefix, t.Value)
	p.prefix += "\t"
	p.forDepth++
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			if p.tryParseCommonTags(t.Value) {
				continue
			}
			switch string(t.Value) {
			case "endfor":
				skipTagContents(s)
				p.forDepth--
				p.prefix = p.prefix[1:]
				fmt.Fprintf(w, "%s}\n", p.prefix)
				return
			default:
				log.Fatalf("unexpected tag found inside for loop: %s at %s", t.Value, s.Context())
			}
		default:
			log.Fatalf("unexpected token found %s when parsing for loop at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse for loop: %s", err)
	} else {
		log.Fatalf("cannot find endfor tag at %s", s.Context())
	}
}

func (p *parser) parseIf() {
	s := p.s
	w := p.w
	t := expectTagContents(s)
	fmt.Fprintf(w, "%sif %s {\n", p.prefix, t.Value)
	p.prefix += "\t"
	elseUsed := false
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			if p.tryParseCommonTags(t.Value) {
				continue
			}
			switch string(t.Value) {
			case "endif":
				skipTagContents(s)
				p.prefix = p.prefix[1:]
				fmt.Fprintf(w, "%s}\n", p.prefix)
				return
			case "else":
				if elseUsed {
					log.Fatalf("duplicate else branch found at %s", s.Context())
				}
				skipTagContents(s)
				fmt.Fprintf(w, "%s} else {\n", p.prefix[1:])
				elseUsed = true
			case "elseif":
				if elseUsed {
					log.Fatalf("unexpected elseif branch found after else branch at %s", s.Context())
				}
				t = expectTagContents(s)
				fmt.Fprintf(w, "%s} else if %s {\n", p.prefix[1:], t.Value)
			default:
				log.Fatalf("unexpected tag found inside if condition: %s at %s", t.Value, s.Context())
			}
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse if branch: %s", err)
	} else {
		log.Fatalf("cannot find endif tag at %s", s.Context())
	}
}

func (p *parser) tryParseCommonTags(tagName []byte) bool {
	s := p.s
	w := p.w
	prefix := p.prefix
	switch string(tagName) {
	case "s":
		t := expectTagContents(s)
		fmt.Fprintf(w, "%sqw.E.S(%s)\n", prefix, t.Value)
	case "v":
		t := expectTagContents(s)
		fmt.Fprintf(w, "%sqw.E.V(%s)\n", prefix, t.Value)
	case "d":
		t := expectTagContents(s)
		fmt.Fprintf(w, "%sqw.D(%s)\n", prefix, t.Value)
	case "f":
		t := expectTagContents(s)
		fmt.Fprintf(w, "%sqw.F(%s)\n", prefix, t.Value)
	case "return":
		skipTagContents(s)
		fmt.Fprintf(w, "%squicktemplate.ReleaseWriter(qw)\n", prefix)
		fmt.Fprintf(w, "%sreturn\n", prefix)
	case "break":
		if p.forDepth <= 0 {
			log.Fatalf("found break tag outside for loop at %s", s.Context())
		}
		skipTagContents(s)
		fmt.Fprintf(w, "%sbreak\n", prefix)
	case "code":
		p.parseCode()
	case "for":
		p.parseFor()
	case "if":
		p.parseIf()
	default:
		return false
	}
	return true
}

func (p *parser) parseCode() {
	t := expectTagContents(p.s)
	fmt.Fprintf(p.w, "%s%s\n", p.prefix, t.Value)
}

func (p *parser) emitText(text []byte) {
	w := p.w
	prefix := p.prefix
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

func (p *parser) emitFuncStart(fname, fargs string) {
	fmt.Fprintf(p.w, `
func %sStream(w *io.Writer, %s) {
	qw := quicktemplate.AcquireWriter(w)
`,
		fname, fargs)
}

func (p *parser) emitFuncEnd(fname, fargs, fargsNoTypes string) {
	fmt.Fprintf(p.w, `
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

func skipTagContents(s *Scanner) {
	expectTagContents(s)
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
