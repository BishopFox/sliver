// +build appengine !linux,!freebsd,!darwin,!dragonfly,!netbsd,!openbsd

package kong

import "io"

func guessWidth(w io.Writer) int {
	return 80
}
