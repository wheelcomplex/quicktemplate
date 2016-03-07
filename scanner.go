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

type Scanner struct {
	r   *bufio.Reader
	t   Token
	c   byte
	err error

	line int
	pos  int

	nextTokenID int
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r: bufio.NewReader(r),
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
			if string(s.t.Value) == "comment" {
				if !s.skipComment() {
					return false
				}
				continue
			}
		}
		return true
	}
}

func (s *Scanner) skipComment() bool {
	for {
		if !s.nextByte() {
			return false
		}
		if s.c != '{' {
			continue
		}
		if !s.nextByte() {
			return false
		}
		if s.c != '%' {
			s.unreadByte('~')
			continue
		}
		ok := s.readTagName()
		s.nextTokenID = Text
		if !ok {
			s.err = nil
			continue
		}
		if string(s.t.Value) == "endcomment" {
			if !s.readTagContents() {
				return false
			}
			return true
		}
	}
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
