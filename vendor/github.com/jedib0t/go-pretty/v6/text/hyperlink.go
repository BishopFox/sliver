package text

import (
	"fmt"
	"strings"
)

func Hyperlink(url, text string) string {
	url = sanitizeHyperlinkURL(url)
	if url == "" {
		return text
	}
	if text == "" {
		return url
	}
	// source https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// sanitizeHyperlinkURL strips ASCII control characters (including ESC and BEL)
// from the URL; left in place, they could terminate the OSC 8 sequence early
// and inject arbitrary escape sequences into the terminal output.
func sanitizeHyperlinkURL(url string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, url)
}
