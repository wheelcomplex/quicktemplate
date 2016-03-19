package quicktemplate

func appendURLEncode(dst []byte, src string) []byte {
	for i, n := 0, len(src); i < n; i++ {
		c := src[i]

		// See http://www.w3.org/TR/html5/forms.html#form-submission-algorithm
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '-' || c == '.' || c == '_' {
			dst = append(dst, c)
		} else {
			if c == ' ' {
				dst = append(dst, '+')
			} else {
				dst = append(dst, '%', hexCharUpper(c>>4), hexCharUpper(c&15))
			}
		}
	}
	return dst
}

func hexCharUpper(c byte) byte {
	if c < 10 {
		return '0' + c
	}
	return c - 10 + 'A'
}
