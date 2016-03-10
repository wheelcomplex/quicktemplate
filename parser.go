package quicktemplate

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"io"
	"strings"
)

type parser struct {
	s                 *scanner
	w                 io.Writer
	packageName       string
	prefix            string
	forDepth          int
	skipOutputDepth   int
	importsUseEmitted bool
}

func parse(w io.Writer, r io.Reader, filePath, packageName string) error {
	p := &parser{
		s:           newScanner(r, filePath),
		w:           w,
		packageName: packageName,
	}
	return p.parseTemplate()
}

func (p *parser) parseTemplate() error {
	s := p.s
	p.Printf("package %s\n", p.packageName)
	p.Printf(`import (
	"io"

	"github.com/valyala/quicktemplate"
)
`)
	nonimportEmitted := false
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case text:
			// just skip top-level text
		case tagName:
			switch string(t.Value) {
			case "import":
				if nonimportEmitted {
					return fmt.Errorf("imports must be at the top of the template. Found at %s", s.Context())
				}
				if err := p.parseImport(); err != nil {
					return err
				}
			case "code":
				p.emitImportsUse()
				if err := p.parseCode(); err != nil {
					return err
				}
				nonimportEmitted = true
			case "func":
				p.emitImportsUse()
				if err := p.parseFunc(); err != nil {
					return err
				}
				nonimportEmitted = true
			default:
				return fmt.Errorf("unexpected tag found outside func: %q at %s", t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found %s outside func at %s", t, s.Context())
		}
	}
	p.emitImportsUse()
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse template: %s", err)
	}
	return nil
}

func (p *parser) emitImportsUse() {
	if p.importsUseEmitted {
		return
	}
	p.Printf(`var (
	_ = io.Copy
	_ = quicktemplate.AcquireByteBuffer
)
`)
	p.importsUseEmitted = true
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
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case text:
			p.emitText(t.Value)
		case tagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return fmt.Errorf("error in func %q: %s", fname, err)
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
				return nil
			default:
				return fmt.Errorf("unexpected tag found in func %q: %q at %s", fname, t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found when parsing func %q: %s at %s", fname, t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse func %q: %s", fname, err)
	}
	return fmt.Errorf("cannot find endfunc tag for func %q at %s", fname, s.Context())
}

func (p *parser) parseFor() error {
	s := p.s
	t, err := expectTagContents(s)
	if err != nil {
		return err
	}
	if err = validateForStmt(t.Value); err != nil {
		return err
	}
	p.Printf("for %s {", t.Value)
	p.prefix += "\t"
	p.forDepth++
	forStr := "for " + string(t.Value)
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case text:
			p.emitText(t.Value)
		case tagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return fmt.Errorf("error in %q: %s", forStr, err)
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
				p.Printf("}")
				return nil
			default:
				return fmt.Errorf("unexpected tag found in %q: %q at %s", forStr, t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found when parsing %q: %s at %s", forStr, t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse %q: %s", forStr, err)
	}
	return fmt.Errorf("cannot find endfor tag for %q at %s", forStr, s.Context())
}

func (p *parser) parseIf() error {
	s := p.s
	t, err := expectTagContents(s)
	if err != nil {
		return err
	}
	if len(t.Value) == 0 {
		return fmt.Errorf("empty if condition at %s", s.Context())
	}
	if err = validateIfStmt(t.Value); err != nil {
		return fmt.Errorf("error in if condition at %s: %s", s.Context(), err)
	}
	p.Printf("if %s {", t.Value)
	p.prefix += "\t"
	elseUsed := false
	ifStr := "if " + string(t.Value)
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case text:
			p.emitText(t.Value)
		case tagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return fmt.Errorf("error in %q: %s", ifStr, err)
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
				p.Printf("}")
				return nil
			case "else":
				if elseUsed {
					return fmt.Errorf("duplicate else branch found for %q at %s", ifStr, s.Context())
				}
				if err = skipTagContents(s); err != nil {
					return err
				}
				p.prefix = p.prefix[1:]
				p.Printf("} else {")
				p.prefix += "\t"
				elseUsed = true
			case "elseif":
				if elseUsed {
					return fmt.Errorf("unexpected elseif branch found after else branch for %q at %s",
						ifStr, s.Context())
				}
				t, err = expectTagContents(s)
				if err != nil {
					return err
				}
				p.prefix = p.prefix[1:]
				p.Printf("} else if %s {", t.Value)
				p.prefix += "\t"
			default:
				return fmt.Errorf("unexpected tag found in %q: %q at %s", ifStr, t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found when parsing %q: %s at %s", ifStr, t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse %q: %s", ifStr, err)
	}
	return fmt.Errorf("cannot find endif tag for %q at %s", ifStr, s.Context())
}

func (p *parser) tryParseCommonTags(tagBytes []byte) (bool, error) {
	s := p.s
	tagNameStr := string(tagBytes)
	switch tagNameStr {
	case "s", "v", "d", "f", "q", "z", "s=", "v=", "d=", "f=", "q=", "z=":
		t, err := expectTagContents(s)
		if err != nil {
			return false, err
		}
		filter := "N()."
		if len(tagNameStr) == 1 {
			switch tagNameStr {
			case "s", "v", "q", "z":
				filter = "E()."
			}
		} else {
			tagNameStr = tagNameStr[:len(tagNameStr)-1]
		}
		tagNameStr = strings.ToUpper(tagNameStr)
		p.Printf("qw.%s%s(%s)", filter, tagNameStr, t.Value)
	case "=":
		t, err := expectTagContents(s)
		if err != nil {
			return false, err
		}
		fname, fargs, err := parseFnameFargs(s, t.Value)
		if err != nil {
			return false, err
		}
		p.Printf("stream%s(qw, %s)", fname, fargs)
	case "return":
		if err := p.skipAfterTag("return"); err != nil {
			return false, err
		}
	case "break":
		if p.forDepth <= 0 {
			return false, fmt.Errorf("found break tag outside for loop at %s", s.Context())
		}
		if err := p.skipAfterTag("break"); err != nil {
			return false, err
		}
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

func (p *parser) skipAfterTag(tagStr string) error {
	s := p.s
	if err := skipTagContents(s); err != nil {
		return err
	}
	p.Printf("%s", tagStr)
	p.skipOutputDepth++
	defer func() {
		p.skipOutputDepth--
	}()
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case text:
			// skip text
		case tagName:
			ok, err := p.tryParseCommonTags(t.Value)
			if err != nil {
				return fmt.Errorf("error when parsing contents after %q: %s", tagStr, err)
			}
			if ok {
				continue
			}
			switch string(t.Value) {
			case "endfunc", "endfor", "endif", "else", "elseif":
				s.Rewind()
				return nil
			default:
				return fmt.Errorf("unexpected tag found after %q: %q at %s", tagStr, t.Value, s.Context())
			}
		default:
			return fmt.Errorf("unexpected token found when parsing contents after %q: %s at %s", tagStr, t, s.Context())
		}
	}
	if err := s.LastError(); err != nil {
		return fmt.Errorf("cannot parse contents after %q: %s", tagStr, err)
	}
	return fmt.Errorf("cannot find closing tag after %q at %s", tagStr, s.Context())
}

func (p *parser) parseImport() error {
	t, err := expectTagContents(p.s)
	if err != nil {
		return err
	}
	if len(t.Value) == 0 {
		return fmt.Errorf("empty import found at %s", p.s.Context())
	}
	p.Printf("import %s\n", t.Value)
	return nil
}

func (p *parser) parseCode() error {
	t, err := expectTagContents(p.s)
	if err != nil {
		return err
	}
	p.Printf("%s\n", t.Value)
	return nil
}

func parseFnameFargsNoTypes(s *scanner, f []byte) (string, string, string, error) {
	fname, fargs, err := parseFnameFargs(s, f)
	if err != nil {
		return "", "", "", err
	}

	// extract function arg names
	fStr := fmt.Sprintf("func (%s)", fargs)
	expr, err := goparser.ParseExpr(fStr)
	if err != nil {
		return "", "", "", fmt.Errorf("cannot parse arguments for func %q at %s: %s", fname, s.Context(), err)
	}
	ft := expr.(*ast.FuncType)
	var args []string
	for _, f := range ft.Params.List {
		if len(f.Names) == 0 {
			return "", "", "", fmt.Errorf("func %q cannot contain untyped arguments at %s", fname, s.Context())
		}
		for _, n := range f.Names {
			if n == nil {
				return "", "", "", fmt.Errorf("func %q cannot contain untyped arguments at %s", fname, s.Context())
			}
			args = append(args, n.Name)
		}
	}
	fargsNoTypes := strings.Join(args, ", ")
	return fname, fargs, fargsNoTypes, nil
}

func parseFnameFargs(s *scanner, f []byte) (string, string, error) {
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
	for len(text) > 0 {
		n := bytes.IndexByte(text, '`')
		if n < 0 {
			p.Printf("qw.N().S(`%s`)", text)
			return
		}
		p.Printf("qw.N().S(`%s`)", text[:n])
		p.Printf("qw.N().S(\"`\")")
		text = text[n+1:]
	}
}

func (p *parser) emitFuncStart(fname, fargs string) {
	p.Printf("func stream%s(qw *quicktemplate.Writer, %s) {", fname, fargs)
	p.prefix = "\t"
}

func (p *parser) emitFuncEnd(fname, fargs, fargsNoTypes string) {
	p.prefix = ""
	p.Printf("}\n")

	fPrefix := "Write"
	if !isUpper(fname[0]) {
		fPrefix = "write"
	}
	p.Printf("func %s%s(w io.Writer, %s) {", fPrefix, fname, fargs)
	p.prefix = "\t"
	p.Printf("qw := quicktemplate.AcquireWriter(w)")
	p.Printf("stream%s(qw, %s)", fname, fargsNoTypes)
	p.Printf("quicktemplate.ReleaseWriter(qw)")
	p.prefix = ""
	p.Printf("}\n")

	p.Printf("func %s(%s) string {", fname, fargs)
	p.prefix = "\t"
	p.Printf("bb := quicktemplate.AcquireByteBuffer()")
	p.Printf("%s%s(bb, %s)", fPrefix, fname, fargsNoTypes)
	p.Printf("s := string(bb.B)")
	p.Printf("quicktemplate.ReleaseByteBuffer(bb)")
	p.Printf("return s")
	p.prefix = ""
	p.Printf("}\n")
}

func (p *parser) Printf(format string, args ...interface{}) {
	if p.skipOutputDepth > 0 {
		return
	}
	w := p.w
	fmt.Fprintf(w, "%s", p.prefix)
	p.s.WriteLineComment(w)
	fmt.Fprintf(w, "%s", p.prefix)
	fmt.Fprintf(w, format, args...)
	fmt.Fprintf(w, "\n")
}

func skipTagContents(s *scanner) error {
	_, err := expectTagContents(s)
	return err
}

func expectTagContents(s *scanner) (*token, error) {
	return expectToken(s, tagContents)
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

func validateForStmt(stmt []byte) error {
	exprStr := fmt.Sprintf("func () { for %s {} }", stmt)
	_, err := goparser.ParseExpr(exprStr)
	return err
}

func validateIfStmt(stmt []byte) error {
	exprStr := fmt.Sprintf("func () { if %s {} }", stmt)
	_, err := goparser.ParseExpr(exprStr)
	return err
}
