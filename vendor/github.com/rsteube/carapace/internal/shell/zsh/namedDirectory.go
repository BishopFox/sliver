package zsh

import (
	"strings"

	"github.com/rsteube/carapace/internal/env"
)

type namedDirectories map[string]string

// NamedDirectories provides rudimentary named directory support as these aren't expanded by zsh in the `${words}` provided to the compdef function
var NamedDirectories = make(namedDirectories)

func (nd *namedDirectories) match(s string) string {
	if strings.HasPrefix(s, "~") && !strings.HasPrefix(s, "~/") && strings.Contains(s, "/") {
		return NamedDirectories[strings.SplitN(s, "/", 2)[0][1:]]
	}
	return ""
}

// Matches checks if given string has a known named directory prefix
func (nd *namedDirectories) Matches(s string) bool {
	return nd.match(s) != ""
}

// Replace replaces a known named directory prefix with the actual folder
func (nd *namedDirectories) Replace(s string) string {
	if match := nd.match(s); match != "" {
		if !strings.HasSuffix(match, "/") {
			match = match + "/"
		}
		return match + strings.SplitN(s, "/", 2)[1]
	}
	return s
}

func init() {
	if hashDirs := env.Hashdirs(); hashDirs != "" {
		for _, line := range strings.Split(hashDirs, "\n") {
			if splitted := strings.SplitN(line, "=", 2); len(splitted) == 2 {
				NamedDirectories[splitted[0]] = splitted[1]
			}
		}
	}
}
