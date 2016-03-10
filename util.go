package quicktemplate

import (
	"path/filepath"
	"reflect"
	"unicode"
	"unsafe"
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

func isUpper(c byte) bool {
	return unicode.IsUpper(rune(c))
}

func unsafeStrToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func getPackageName(filename string) (string, error) {
	filenameAbs, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	dir, _ := filepath.Split(filenameAbs)
	return filepath.Base(dir), nil
}
