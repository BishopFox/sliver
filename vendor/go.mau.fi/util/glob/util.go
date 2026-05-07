package glob

import (
	"regexp"
	"strings"
)

var redundantStarRegex = regexp.MustCompile(`\*{2,}`)
var maybeRedundantQuestionRegex = regexp.MustCompile(`[*?]{2,}`)
var wildcardRegex = regexp.MustCompile(`[*?]+`)

func SplitPattern(pattern string) []string {
	indexes := wildcardRegex.FindAllStringIndex(pattern, -1)
	if len(indexes) == 0 {
		return []string{pattern}
	}
	parts := make([]string, 0, len(indexes)+1)
	start := 0
	for _, part := range indexes {
		end := part[0]
		if end > start {
			parts = append(parts, pattern[start:end])
		}
		parts = append(parts, pattern[part[0]:part[1]])
		start = part[1]
	}
	if start < len(pattern) {
		parts = append(parts, pattern[start:])
	}
	return parts
}

func Simplify(pattern string) string {
	pattern = redundantStarRegex.ReplaceAllString(pattern, "*")
	pattern = maybeRedundantQuestionRegex.ReplaceAllStringFunc(pattern, func(s string) string {
		if !strings.ContainsRune(s, '*') {
			return s
		}
		return strings.Repeat("?", strings.Count(s, "?")) + "*"
	})
	return pattern
}
