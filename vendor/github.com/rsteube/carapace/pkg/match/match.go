package match

import (
	"os"
	"strconv"
	"strings"
)

type Match int

const (
	CASE_SENSITIVE Match = iota
	CASE_INSENSITIVE
)

func (m Match) Equal(s, t string) bool {
	if m == CASE_INSENSITIVE {
		strings.EqualFold(s, t)
	}
	return s == t

}

func (m Match) HasPrefix(s, prefix string) bool {
	if m == CASE_INSENSITIVE {
		return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
	}
	return strings.HasPrefix(s, prefix)
}

func (m Match) TrimPrefix(s, prefix string) string {
	if m.HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

var match = CASE_SENSITIVE

func init() {
	switch os.Getenv("CARAPACE_MATCH") {
	case "CASE_INSENSITIVE", strconv.Itoa(int(CASE_INSENSITIVE)):
		match = CASE_INSENSITIVE
	}
}

func Equal(s, t string) bool {
	return match.Equal(s, t)
}

func HasPrefix(s, prefix string) bool {
	return match.HasPrefix(s, prefix)
}

func TrimPrefix(s, prefix string) string {
	return match.TrimPrefix(s, prefix)
}
