package glob

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

type RegexGlob struct {
	regex *regexp.Regexp
}

func (rg *RegexGlob) Match(s string) bool {
	return rg.regex.MatchString(s)
}

type swWriter interface {
	io.StringWriter
	io.Writer
}

// ToRegexPattern converts a glob pattern to a regex pattern and writes it to the given buffer.
//
// Only errors returned by the Write calls are returned by this function
// (so if a non-erroring writer is used, this function will always return nil).
func ToRegexPattern(pattern string, buf swWriter) error {
	for _, part := range SplitPattern(pattern) {
		if strings.ContainsRune(part, '*') || strings.ContainsRune(part, '?') {
			questions := strings.Count(part, "?")
			star := strings.ContainsRune(part, '*')
			if star {
				if questions > 0 {
					_, err := fmt.Fprintf(buf, ".{%d,}", questions)
					if err != nil {
						return err
					}
				} else {
					_, err := buf.WriteString(".*")
					if err != nil {
						return err
					}
				}
			} else if questions > 0 {
				_, err := fmt.Fprintf(buf, ".{%d}", questions)
				if err != nil {
					return err
				}
			}
		} else {
			_, err := buf.WriteString(regexp.QuoteMeta(part))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CompileRegex compiles the given glob pattern into a regex pattern.
//
// If you want the raw regex string, use [ToRegexPattern] instead
func CompileRegex(pattern string) (*RegexGlob, error) {
	var buf strings.Builder
	buf.WriteRune('^')
	err := ToRegexPattern(pattern, &buf)
	if err != nil {
		// This will never actually happen
		return nil, err
	}
	buf.WriteRune('$')
	regex, err := regexp.Compile(buf.String())
	if err != nil {
		return nil, err
	}
	return &RegexGlob{regex}, nil
}
