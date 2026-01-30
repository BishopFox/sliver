// Package glob implements very simple glob pattern matching used in various parts of the Matrix spec,
// such as push rules and moderation policy lists.
//
// See https://spec.matrix.org/v1.11/appendices/#glob-style-matching for more info.
package glob

import (
	"strings"
)

type Glob interface {
	Match(string) bool
}

var (
	_ Glob = ExactGlob("")
	_ Glob = PrefixGlob("")
	_ Glob = SuffixGlob("")
	_ Glob = ContainsGlob("")
	_ Glob = (*PrefixAndSuffixGlob)(nil)
	_ Glob = (*PrefixSuffixAndContainsGlob)(nil)
	_ Glob = (*RegexGlob)(nil)
)

// Compile compiles a glob pattern into an object that can be used to efficiently match strings against the pattern.
//
// Simple globs will be converted into prefix/suffix/contains checks, while complicated ones will be compiled as regex.
func Compile(pattern string) Glob {
	pattern = Simplify(pattern)
	g := compileSimple(pattern)
	if g != nil {
		return g
	}
	g, _ = CompileRegex(pattern)
	return g
}

// CompileWithImplicitContains is a wrapper for Compile which will replace exact matches with contains matches.
// i.e. if the pattern has no wildcards, it will be treated as if it was surrounded in asterisks (`foo` -> `*foo*`).
func CompileWithImplicitContains(pattern string) Glob {
	g := Compile(pattern)
	if _, isExact := g.(ExactGlob); isExact {
		return ContainsGlob(pattern)
	}
	return g
}

// CompileSimple compiles a glob pattern into one of the non-regex forms.
//
// If the pattern can't be compiled into a simple form, it returns nil.
func CompileSimple(pattern string) Glob {
	return compileSimple(Simplify(pattern))
}

func compileSimple(pattern string) Glob {
	if strings.ContainsRune(pattern, '?') {
		return nil
	}
	switch strings.Count(pattern, "*") {
	case 0:
		return ExactGlob(pattern)
	case 1:
		if strings.HasPrefix(pattern, "*") {
			return SuffixGlob(pattern[1:])
		} else if strings.HasSuffix(pattern, "*") {
			return PrefixGlob(pattern[:len(pattern)-1])
		} else {
			parts := strings.Split(pattern, "*")
			return PrefixAndSuffixGlob{
				Prefix: parts[0],
				Suffix: parts[1],
			}
		}
	case 2:
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			return ContainsGlob(pattern[1 : len(pattern)-1])
		}
		parts := strings.Split(pattern, "*")
		return PrefixSuffixAndContainsGlob{
			Prefix:   parts[0],
			Contains: parts[1],
			Suffix:   parts[2],
		}
	default:
		return nil
	}
}

var sqlCompiler = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
	`*`, `%`,
	`?`, `_`,
)

// ToSQL converts a Matrix glob pattern to a SQL LIKE pattern.
func ToSQL(pattern string) string {
	return sqlCompiler.Replace(Simplify(pattern))
}
