package util

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidPortableFileName checks a final artifact's single path component before
// it crosses from a builder or server to an operator's filesystem. It uses the
// common Windows restrictions even on Unix so a recorded build can be saved
// by clients on either platform.
func ValidPortableFileName(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 255 || !utf8.ValidString(name) ||
		strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") {
		return false
	}
	for _, char := range name {
		if unicode.IsControl(char) || unicode.IsSpace(char) && char != ' ' ||
			strings.ContainsRune(`<>:"/\|?*`, char) {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}
