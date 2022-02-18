package bytealg

import "strings"

func IndexByteString(s string, c byte) int {
	return strings.IndexByte(s, c)
}
