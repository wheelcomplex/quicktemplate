package quicktemplate

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// token ids
const (
	text = iota
	tagName
	tagContents
)

var tokenStrMap = map[int]string{
	text:        "text",
	tagName:     "tagName",
	tagContents: "tagContents",
}

func tokenIDToStr(id int) string {
	str := tokenStrMap[id]
	if str == "" {
		panic(fmt.Sprintf("unknown tokenID=%d", id))
	}
	return str
}

type token struct {
	ID    int
	Value []byte
}

func (t *token) init(id int) {
	t.ID = id
	t.Value = t.Value[:0]
}

func (t *token) String() string {
	return fmt.Sprintf("Token %q, value %q", tokenIDToStr(t.ID), t.Value)
}

type scanner struct {
	r   *bufio.Reader
	t   token
	c   byte
	err error

	filePath string

	line    int
	lineStr []byte

	nextTokenID int

	capture       bool
	capturedValue []byte

	stripSpaceDepth int
}

func newScanner(r io.Reader, filePath string) *scanner {
	return &scanner{
		r:        bufio.NewReader(r),
		filePath: filePath,
	}
}

func (s *scanner) Next() bool {
	for {
		if !s.scanToken() {
			return false
		}
		switch s.t.ID {
		case text:
			if len(s.t.Value) == 0 {
				// skip empty text
				continue
			}
		case tagName:
			switch string(s.t.Value) {
			case "comment":
				if !s.skipComment() {
					return false
				}
				continue
			case "plain":
				if !s.readPlain() {
					return false
				}
				if len(s.t.Value) == 0 {
					// skip empty text
					continue
				}
			case "stripspace":
				if !s.readTagContents() {
					return false
				}
				s.stripSpaceDepth++
				continue
			case "endstripspace":
				if s.stripSpaceDepth == 0 {
					s.err = fmt.Errorf("endstripspace tag found without the corresponding stripspace tag")
					return false
				}
				if !s.readTagContents() {
					return false
				}
				s.stripSpaceDepth--
				continue
			}
		}
		return true
	}
}

func (s *scanner) readPlain() bool {
	s.startCapture()
	ok := s.skipUntilTag("endplain")
	v := s.stopCapture()
	s.t.init(text)
	if ok {
		n := bytes.Index(v, strTagClose)
		v = v[n+len(strTagClose):]
		n = bytes.LastIndex(v, strTagOpen)
		v = v[:n]
		s.t.Value = append(s.t.Value[:0], v...)
	}
	return ok
}

var (
	strTagOpen  = []byte("{%")
	strTagClose = []byte("%}")
)

func (s *scanner) skipComment() bool {
	return s.skipUntilTag("endcomment")
}

func (s *scanner) skipUntilTag(tagName string) bool {
	ok := false
	for {
		if !s.nextByte() {
			break
		}
		if s.c != '{' {
			continue
		}
		if !s.nextByte() {
			break
		}
		if s.c != '%' {
			s.unreadByte('~')
			continue
		}
		ok = s.readTagName()
		s.nextTokenID = text
		if !ok {
			s.err = nil
			continue
		}
		if string(s.t.Value) == tagName {
			ok = s.readTagContents()
			break
		}
	}
	if !ok {
		s.err = fmt.Errorf("cannot find %q tag: %s", tagName, s.err)
	}
	return ok
}

func (s *scanner) scanToken() bool {
	switch s.nextTokenID {
	case text:
		return s.readText()
	case tagName:
		return s.readTagName()
	case tagContents:
		return s.readTagContents()
	default:
		panic(fmt.Sprintf("BUG: unknown nextTokenID %d", s.nextTokenID))
	}
}

func (s *scanner) readText() bool {
	s.t.init(text)
	ok := false
	for {
		if !s.nextByte() {
			ok = (len(s.t.Value) > 0)
			break
		}
		if s.c != '{' {
			s.appendByte()
			continue
		}
		if !s.nextByte() {
			s.appendByte()
			ok = true
			break
		}
		if s.c == '%' {
			s.nextTokenID = tagName
			ok = true
			break
		}
		s.unreadByte('{')
		s.appendByte()
	}
	if s.stripSpaceDepth > 0 {
		s.t.Value = stripSpace(s.t.Value)
	}
	return ok
}

func (s *scanner) readTagName() bool {
	s.t.init(tagName)
	s.skipSpace()
	for {
		if s.isSpace() || s.c == '%' {
			if s.c == '%' {
				s.unreadByte('~')
			}
			s.nextTokenID = tagContents
			return true
		}
		if (s.c >= 'a' && s.c <= 'z') || (s.c >= 'A' && s.c <= 'Z') || (s.c >= '0' && s.c <= '9') || s.c == '=' {
			s.appendByte()
			if !s.nextByte() {
				return false
			}
			continue
		}
		s.err = fmt.Errorf("unexpected character: '%c'", s.c)
		s.unreadByte('~')
		return false
	}
}

func (s *scanner) readTagContents() bool {
	s.t.init(tagContents)
	s.skipSpace()
	for {
		if s.c != '%' {
			s.appendByte()
			if !s.nextByte() {
				return false
			}
			continue
		}
		if !s.nextByte() {
			s.appendByte()
			return false
		}
		if s.c == '}' {
			s.nextTokenID = text
			s.t.Value = stripTrailingSpace(s.t.Value)
			return true
		}
		s.unreadByte('%')
		s.appendByte()
		if !s.nextByte() {
			return false
		}
	}
}

func (s *scanner) skipSpace() {
	for s.nextByte() && s.isSpace() {
	}
}

func (s *scanner) isSpace() bool {
	return isSpace(s.c)
}

func (s *scanner) nextByte() bool {
	if s.err != nil {
		return false
	}
	c, err := s.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		s.err = err
		return false
	}
	if c == '\n' {
		s.line++
		s.lineStr = s.lineStr[:0]
	} else {
		s.lineStr = append(s.lineStr, c)
	}
	s.c = c
	if s.capture {
		s.capturedValue = append(s.capturedValue, c)
	}
	return true
}

func (s *scanner) startCapture() {
	s.capture = true
	s.capturedValue = s.capturedValue[:0]
}

func (s *scanner) stopCapture() []byte {
	s.capture = false
	v := s.capturedValue
	s.capturedValue = s.capturedValue[:0]
	return v
}

func (s *scanner) Token() *token {
	return &s.t
}

func (s *scanner) LastError() error {
	if s.err == nil {
		return nil
	}
	if s.err == io.ErrUnexpectedEOF && s.t.ID == text {
		if s.stripSpaceDepth > 0 {
			return fmt.Errorf("missing endstripspace tag at %s", s.Context())
		}
		return nil
	}

	return fmt.Errorf("error when reading %s at %s: %s",
		tokenIDToStr(s.t.ID), s.Context(), s.err)
}

func (s *scanner) appendByte() {
	s.t.Value = append(s.t.Value, s.c)
}

func (s *scanner) unreadByte(c byte) {
	if err := s.r.UnreadByte(); err != nil {
		panic(fmt.Sprintf("BUG: bufio.Reader.UnreadByte returned non-nil error: %s", err))
	}
	if s.capture {
		s.capturedValue = s.capturedValue[:len(s.capturedValue)-1]
	}
	if s.c == '\n' {
		s.line--
		s.lineStr = s.lineStr[:0] // TODO: use correct line
	} else {
		s.lineStr = s.lineStr[:len(s.lineStr)-1]
	}
	s.c = c
}

func (s *scanner) Context() string {
	var lineStr string
	v := s.lineStr
	if len(v) <= 40 {
		lineStr = fmt.Sprintf("%q", v)
	} else {
		lineStr = fmt.Sprintf("%q ... %q", v[:20], v[len(v)-20:])
	}
	return fmt.Sprintf("file %q, line %d, pos %d, str %s", s.filePath, s.line+1, len(s.lineStr), lineStr)
}

func (s *scanner) WriteLineComment(w io.Writer) {
	fmt.Fprintf(w, "//line %s:%d\n", s.filePath, s.line+1)
}
