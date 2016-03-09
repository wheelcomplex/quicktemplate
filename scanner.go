package quicktemplate

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Token ids
const (
	Text = iota
	TagName
	TagContents
)

var tokenStrMap = map[int]string{
	Text:        "Text",
	TagName:     "TagName",
	TagContents: "TagContents",
}

func tokenIDToStr(id int) string {
	str := tokenStrMap[id]
	if str == "" {
		panic(fmt.Sprintf("unknown tokenID=%d", id))
	}
	return str
}

type Token struct {
	ID    int
	Value []byte
}

func (t *Token) init(id int) {
	t.ID = id
	t.Value = t.Value[:0]
}

func (t *Token) String() string {
	return fmt.Sprintf("Token %q, value %q", tokenIDToStr(t.ID), t.Value)
}

type Scanner struct {
	r   *bufio.Reader
	t   Token
	c   byte
	err error

	filePath string

	line    int
	lineStr []byte

	nextTokenID int

	capture       bool
	capturedValue []byte
}

func NewScanner(r io.Reader, filePath string) *Scanner {
	return &Scanner{
		r:        bufio.NewReader(r),
		filePath: filePath,
	}
}

func (s *Scanner) Next() bool {
	for {
		if !s.scanToken() {
			return false
		}
		switch s.t.ID {
		case Text:
			if len(s.t.Value) == 0 {
				// skip empty text
				continue
			}
		case TagName:
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
			}
		}
		return true
	}
}

func (s *Scanner) readPlain() bool {
	s.startCapture()
	ok := s.skipUntilTag("endplain")
	v := s.stopCapture()
	s.t.init(Text)
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

func (s *Scanner) skipComment() bool {
	return s.skipUntilTag("endcomment")
}

func (s *Scanner) skipUntilTag(tagName string) bool {
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
		s.nextTokenID = Text
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

func (s *Scanner) scanToken() bool {
	switch s.nextTokenID {
	case Text:
		return s.readText()
	case TagName:
		return s.readTagName()
	case TagContents:
		return s.readTagContents()
	default:
		panic(fmt.Sprintf("BUG: unknown nextTokenID %d", s.nextTokenID))
	}
}

func (s *Scanner) readText() bool {
	s.t.init(Text)
	for {
		if !s.nextByte() {
			return len(s.t.Value) > 0
		}
		if s.c != '{' {
			s.appendByte()
			continue
		}
		if !s.nextByte() {
			s.appendByte()
			return true
		}
		if s.c == '%' {
			s.nextTokenID = TagName
			return true
		}
		s.unreadByte('{')
		s.appendByte()
	}
}

func (s *Scanner) readTagName() bool {
	s.t.init(TagName)
	s.skipSpace()
	for {
		if s.isSpace() || s.c == '%' {
			if s.c == '%' {
				s.unreadByte('~')
			}
			s.nextTokenID = TagContents
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

func (s *Scanner) readTagContents() bool {
	s.t.init(TagContents)
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
			s.nextTokenID = Text
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

func (s *Scanner) skipSpace() {
	for s.nextByte() && s.isSpace() {
	}
}

func (s *Scanner) isSpace() bool {
	return isSpace(s.c)
}

func (s *Scanner) nextByte() bool {
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

func (s *Scanner) startCapture() {
	s.capture = true
	s.capturedValue = s.capturedValue[:0]
}

func (s *Scanner) stopCapture() []byte {
	s.capture = false
	v := s.capturedValue
	s.capturedValue = s.capturedValue[:0]
	return v
}

func (s *Scanner) Token() *Token {
	return &s.t
}

func (s *Scanner) LastError() error {
	if s.err == nil {
		return nil
	}
	if s.err == io.ErrUnexpectedEOF && s.t.ID == Text {
		return nil
	}

	return fmt.Errorf("error when reading %s at %s: %s",
		tokenIDToStr(s.t.ID), s.Context(), s.err)
}

func (s *Scanner) appendByte() {
	s.t.Value = append(s.t.Value, s.c)
}

func (s *Scanner) unreadByte(c byte) {
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

func (s *Scanner) Context() string {
	var lineStr string
	v := s.lineStr
	if len(v) <= 40 {
		lineStr = fmt.Sprintf("%q", v)
	} else {
		lineStr = fmt.Sprintf("%q ... %q", v[:20], v[len(v)-20:])
	}
	return fmt.Sprintf("file %q, line %d, pos %d, str %s", s.filePath, s.line+1, len(s.lineStr), lineStr)
}
