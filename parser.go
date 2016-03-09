package quicktemplate

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type parser struct {
	s           *scanner
	w           io.Writer
	packageName string
	prefix      string
	forDepth    int
}

func parse(w io.Writer, r io.Reader, filePath string) error {
	packageName, err := getPackageName(filePath)
	if err != nil {
		return err
	}
	p := &parser{
		s:           newScanner(r, filePath),
		w:           w,
		packageName: packageName,
	}
	return p.parseTemplate()
}

func (p *parser) parseTemplate() error {
	s := p.s
	fmt.Fprintf(p.w, "package %s\n\n", p.packageName)
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			// just skip top-level text
		case TagName:
			switch string(t.Value) {
			case "code":
				if err := p.parseCode(); err != nil {
					return err
				}
			case "func":
				if err := p.parseFunc(); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unexpected tag found outside func: %s at %s", t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found %s outside func at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse template: %s", err)
	}
	return nil
}

func (p *parser) parseFunc() error {
	s := p.s
	t, err := expectTagContents(s)
	if err != nil {
		return err
	}
	fname, fargs, fargsNoTypes, err := parseFnameFargsNoTypes(s, t.Value)
	if err != nil {
		return err
	}
	p.emitFuncStart(fname, fargs)
	p.prefix += "\t"
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			switch string(t.Value) {
			case "endfunc":
				if err = skipTagContents(s); err != nil {
					return err
				}
				p.emitFuncEnd(fname, fargs, fargsNoTypes)
				p.prefix = p.prefix[1:]
				return nil
			default:
				return fmt.Errorf("unexpected tag found inside func: %s at %s", t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found %s when parsing func at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse func: %s", err)
	}
	return fmt.Errorf("cannot find endfunc tag at %s", s.Context())
}

func (p *parser) parseFor() error {
	s := p.s
	w := p.w
	t, err := expectTagContents(s)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%sfor %s {\n", p.prefix, t.Value)
	p.prefix += "\t"
	p.forDepth++
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			switch string(t.Value) {
			case "endfor":
				if err = skipTagContents(s); err != nil {
					return err
				}
				p.forDepth--
				p.prefix = p.prefix[1:]
				fmt.Fprintf(w, "%s}\n", p.prefix)
				return nil
			default:
				return fmt.Errorf("unexpected tag found inside for loop: %s at %s", t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found %s when parsing for loop at %s", t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse for loop: %s", err)
	}
	return fmt.Errorf("cannot find endfor tag at %s", s.Context())
}

func (p *parser) parseIf() error {
	s := p.s
	w := p.w
	t, err := expectTagContents(s)
	if err != nil {
		return err
	}
	if len(t.Value) == 0 {
		return fmt.Errorf("empty if condition at %s", s.Context())
	}
	fmt.Fprintf(w, "%sif %s {\n", p.prefix, t.Value)
	p.prefix += "\t"
	elseUsed := false
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			p.emitText(t.Value)
		case TagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			switch string(t.Value) {
			case "endif":
				if err = skipTagContents(s); err != nil {
					return err
				}
				p.prefix = p.prefix[1:]
				fmt.Fprintf(w, "%s}\n", p.prefix)
				return nil
			case "else":
				if elseUsed {
					return fmt.Errorf("duplicate else branch found at %s", s.Context())
				}
				if err = skipTagContents(s); err != nil {
					return err
				}
				fmt.Fprintf(w, "%s} else {\n", p.prefix[1:])
				elseUsed = true
			case "elseif":
				if elseUsed {
					return fmt.Errorf("unexpected elseif branch found after else branch at %s", s.Context())
				}
				t, err = expectTagContents(s)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "%s} else if %s {\n", p.prefix[1:], t.Value)
			default:
				return fmt.Errorf("unexpected tag found inside if condition: %s at %s", t.Value, s.Context())
			}
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse if branch: %s", err)
	}
	return fmt.Errorf("cannot find endif tag at %s", s.Context())
}

func (p *parser) tryParseCommonTags(tagName []byte) (bool, error) {
	s := p.s
	w := p.w
	prefix := p.prefix
	tagNameStr := string(tagName)
	switch tagNameStr {
	case "s", "v", "d", "f", "s=", "v=", "d=", "f=":
		t, err := expectTagContents(s)
		if err != nil {
			return false, err
		}
		filter := ""
		if len(tagNameStr) == 1 {
			filter = "e."
		} else {
			tagNameStr = tagNameStr[:len(tagNameStr)-1]
		}
		fmt.Fprintf(w, "%sqw.%s%s(%s)\n", prefix, filter, tagNameStr, t.Value)
	case "=":
		t, err := expectTagContents(s)
		if err != nil {
			return false, err
		}
		fname, fargs, err := parseFnameFargs(s, t.Value)
		if err != nil {
			return false, err
		}
		fmt.Fprintf(w, "%s%sStream(qw.w, %s)\n", prefix, fname, fargs)
	case "return":
		if err := skipTagContents(s); err != nil {
			return false, err
		}
		fmt.Fprintf(w, "%squicktemplate.ReleaseWriter(qw)\n", prefix)
		fmt.Fprintf(w, "%sreturn\n", prefix)
	case "break":
		if p.forDepth <= 0 {
			return false, fmt.Errorf("found break tag outside for loop at %s", s.Context())
		}
		if err := skipTagContents(s); err != nil {
			return false, err
		}
		fmt.Fprintf(w, "%sbreak\n", prefix)
	case "code":
		if err := p.parseCode(); err != nil {
			return false, err
		}
	case "for":
		if err := p.parseFor(); err != nil {
			return false, err
		}
	case "if":
		if err := p.parseIf(); err != nil {
			return false, err
		}
	default:
		return false, nil
	}
	return true, nil
}

func (p *parser) parseCode() error {
	t, err := expectTagContents(p.s)
	if err != nil {
		return err
	}
	fmt.Fprintf(p.w, "%s%s\n", p.prefix, t.Value)
	return nil
}

func parseFnameFargsNoTypes(s *scanner, f []byte) (string, string, string, error) {
	fname, fargs, err := parseFnameFargs(s, f)
	if err != nil {
		return "", "", "", err
	}

	var args []string
	for _, a := range strings.Split(fargs, ",") {
		a = string(stripLeadingSpace([]byte(a)))
		n := 0
		for n < len(a) && !isSpace(a[n]) {
			n++
		}
		args = append(args, a[:n])
	}
	fargsNoTypes := strings.Join(args, ", ")
	return fname, fargs, fargsNoTypes, nil
}

func parseFnameFargs(s *scanner, f []byte) (string, string, error) {
	// TODO: use real Go parser here
	n := bytes.IndexByte(f, '(')
	if n < 0 {
		return "", "", fmt.Errorf("missing '(' for function arguments at %s", s.Context())
	}
	fname := string(stripTrailingSpace(f[:n]))
	if len(fname) == 0 {
		return "", "", fmt.Errorf("empty function name at %s", s.Context())
	}

	f = f[n+1:]
	n = bytes.LastIndexByte(f, ')')
	if n < 0 {
		return "", "", fmt.Errorf("missing ')' for function arguments at %s", s.Context())
	}
	fargs := string(f[:n])
	return fname, fargs, nil
}

func (p *parser) emitText(text []byte) {
	w := p.w
	prefix := p.prefix
	for len(text) > 0 {
		n := bytes.IndexByte(text, '`')
		if n < 0 {
			fmt.Fprintf(w, "%sqw.s(`%s`)\n", prefix, text)
			return
		}
		fmt.Fprintf(w, "%sqw.s(`%s`)\n", prefix, text[:n])
		fmt.Fprintf(w, "%sqw.s(\"`\")\n", prefix)
		text = text[n+1:]
	}
}

func (p *parser) emitFuncStart(fname, fargs string) {
	fmt.Fprintf(p.w, `
func %sStream(w io.Writer, %s) {
	qw := quicktemplate.AcquireWriter(w)
`,
		fname, fargs)
}

func (p *parser) emitFuncEnd(fname, fargs, fargsNoTypes string) {
	fmt.Fprintf(p.w, "\tquicktemplate.ReleaseWriter(qw)\n"+`
}

func %s(%s) string {
	bb := quicktemplate.AcquireByteBuffer()
	%sStream(bb, %s)
	s := string(bb.Bytes())
	quicktemplate.ReleaseByteBuffer(bb)
	return s
}
`,
		fname, fargs, fname, fargsNoTypes)
}

func skipTagContents(s *scanner) error {
	_, err := expectTagContents(s)
	return err
}

func expectTagContents(s *scanner) (*token, error) {
	return expectToken(s, TagContents)
}

func expectToken(s *scanner, id int) (*token, error) {
	if !s.Next() {
		return nil, fmt.Errorf("cannot find token %s: %v", tokenIDToStr(id), s.LastError())
	}
	t := s.Token()
	if t.ID != id {
		return nil, fmt.Errorf("unexpected token found %s. Expecting %s at %s", t, tokenIDToStr(id), s.Context())
	}
	return t, nil
}

func getPackageName(filePath string) (string, error) {
	fname := filepath.Base(filePath)
	n := strings.LastIndex(fname, ".")
	if n < 0 {
		n = len(fname)
	}
	packageName := fname[:n]

	if len(packageName) == 0 {
		return "", fmt.Errorf("cannot derive package name from filePath %q", filePath)
	}
	return packageName, nil
}
