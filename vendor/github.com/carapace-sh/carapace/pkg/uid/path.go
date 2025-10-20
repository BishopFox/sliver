package uid

import (
	"net/url"
	"strings"
)

// PathEscape is like url.PathEscape but keeps '/'.
func PathEscape(s string) string {
	segments := strings.Split(s, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
