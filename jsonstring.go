package quicktemplate

// appendJSONString is a synonym to strconv.AppendQuote, but works 3x faster.
func appendJSONString(dst []byte, s string) []byte {
	j := 0
	b := unsafeStrToBytes(s)
	for i, n := 0, len(b); i < n; i++ {
		switch b[i] {
		case '"':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\"`...)
			j = i + 1
		case '\\':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\\`...)
			j = i + 1
		case '\n':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\n`...)
			j = i + 1
		case '\r':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\r`...)
			j = i + 1
		case '\t':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\t`...)
			j = i + 1
		case '\f':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\u000c`...)
			j = i + 1
		case '\b':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\u0008`...)
			j = i + 1
		case '<':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\u003c`...)
			j = i + 1
		case '\'':
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\u0027`...)
			j = i + 1
		case 0:
			dst = append(dst, b[j:i]...)
			dst = append(dst, `\u0000`...)
			j = i + 1
		}
	}
	return append(dst, b[j:]...)
}
