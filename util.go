package quicktemplate

import (
	"unicode"
)

func stripLeadingSpace(b []byte) []byte {
	for len(b) > 0 && isSpace(b[0]) {
		b = b[1:]
	}
	return b
}

func stripTrailingSpace(b []byte) []byte {
	for len(b) > 0 && isSpace(b[len(b)-1]) {
		b = b[:len(b)-1]
	}
	return b
}

func stripSpace(b []byte) []byte {
	b = stripLeadingSpace(b)
	return stripTrailingSpace(b)
}

func isSpace(c byte) bool {
	return unicode.IsSpace(rune(c))
}
