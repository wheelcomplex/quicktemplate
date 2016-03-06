package quicktemplate

import (
	"bufio"
	"fmt"
	"io"
)

// Token ids
const (
	Text = iota
	TagName
	TagContents
)

func tokenIDToStr(id int) string {
	switch id {
	case Text:
		return "Text"
	case TagName:
		return "TagName"
	case TagContents:
		return "TagContents"
	default:
		panic(fmt.Sprintf("unknown tokenID=%d", id))
	}
}

type Token struct {
	ID    int
	Value []byte
}

func (t *Token) init(id int) {
	t.ID = id
	t.Value = t.Value[:0]
}

type Scanner struct {
	r   *bufio.Reader
	t   Token
	c   byte
	err error

	line int
	pos  int

	nextToken int
}

func NewScanner(r *bufio.Reader) *Scanner {
	return &Scanner{
		r: r,
	}
}

func (s *Scanner) Next() bool {
	switch s.nextToken {
	case Text:
		if !s.readText() {
			return false
		}
		if len(s.t.Value) > 0 {
			return true
		}
		return s.readTagName()
	case TagName:
		return s.readTagName()
	case TagContents:
		return s.readTagContents()
	default:
		panic(fmt.Sprintf("BUG: unknown nextToken %d", s.nextToken))
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
			s.nextToken = TagName
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
			if len(s.t.Value) == 0 {
				panic("BUG: empty tag name")
			}
			if s.c == '%' {
				s.unreadByte('~')
			}
			s.nextToken = TagContents
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
			s.nextToken = Text
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

func stripTrailingSpace(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if !isSpace(b[i]) {
			return b[:i+1]
		}
	}
	return b[:0]
}

func (s *Scanner) skipSpace() {
	for s.nextByte() && s.isSpace() {
	}
}

func (s *Scanner) isSpace() bool {
	return isSpace(s.c)
}

func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
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
		s.pos = 0
	} else {
		s.pos++
	}
	s.c = c
	return true
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

	var tStr string
	v := s.t.Value
	if len(v) <= 40 {
		tStr = fmt.Sprintf("%q", v)
	} else {
		tStr = fmt.Sprintf("%q ... %q", v[:20], v[len(v)-20:])
	}
	return fmt.Errorf("error when reading %s at line %d, position %d: %s. Token %s",
		tokenIDToStr(s.t.ID), s.line+1, s.pos+1, s.err, tStr)
}

func (s *Scanner) appendByte() {
	s.t.Value = append(s.t.Value, s.c)
}

func (s *Scanner) unreadByte(c byte) {
	if err := s.r.UnreadByte(); err != nil {
		panic(fmt.Sprintf("BUG: bufio.Reader.UnreadByte returned non-nil error: %s", err))
	}
	s.c = c
}
