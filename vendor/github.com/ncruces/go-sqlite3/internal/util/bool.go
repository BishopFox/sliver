package util

import "strings"

func ParseBool(s string) (b, ok bool) {
	if len(s) == 0 {
		return false, false
	}
	if s[0] == '0' {
		return false, true
	}
	if '1' <= s[0] && s[0] <= '9' {
		return true, true
	}
	switch strings.ToLower(s) {
	case "true", "yes", "on":
		return true, true
	case "false", "no", "off":
		return false, true
	}
	return false, false
}
