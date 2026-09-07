package glob

import (
	"strings"

	"go.mau.fi/util/exstrings"
)

// ExactGlob is the result of [Compile] when the pattern contains no glob characters.
// It uses [strings.EqualFold] to match.
type ExactGlob string

func (eg ExactGlob) Match(s string) bool {
	return strings.EqualFold(s, string(eg))
}

// SuffixGlob is the result of [Compile] when the pattern only has one `*` at the beginning.
// It uses [exstrings.HasSuffixFold] to match, which is a case-insensitive version of [strings.HasSuffix].
type SuffixGlob string

func (sg SuffixGlob) Match(s string) bool {
	return exstrings.HasSuffixFold(s, string(sg))
}

// PrefixGlob is the result of [Compile] when the pattern only has one `*` at the end.
// It uses [exstrings.HasPrefixFold] to match, which is a case-insensitive version of [strings.HasPrefix].
type PrefixGlob string

func (pg PrefixGlob) Match(s string) bool {
	return exstrings.HasPrefixFold(s, string(pg))
}

// ContainsGlob is the result of [Compile] when the pattern has two `*`s, one at the beginning and one at the end.
// It uses [exstrings.ContainsFold] to match, which is a case-insensitive version of [strings.Contains].
//
// When there are exactly two `*`s, but they're not surrounding the string, the pattern is compiled as a [PrefixSuffixAndContainsGlob] instead.
type ContainsGlob string

func (cg ContainsGlob) Match(s string) bool {
	return exstrings.ContainsFold(s, string(cg))
}

// PrefixAndSuffixGlob is the result of [Compile] when the pattern only has one `*` in the middle.
type PrefixAndSuffixGlob struct {
	Prefix string
	Suffix string
}

func (psg PrefixAndSuffixGlob) Match(s string) bool {
	return len(s) >= len(psg.Prefix)+len(psg.Suffix) &&
		exstrings.HasPrefixFold(s, psg.Prefix) &&
		exstrings.HasSuffixFold(s, psg.Suffix)
}

// PrefixSuffixAndContainsGlob is the result of [Compile] when the pattern has two `*`s which are not surrounding the rest of the pattern.
type PrefixSuffixAndContainsGlob struct {
	Prefix   string
	Suffix   string
	Contains string
}

func (psacg PrefixSuffixAndContainsGlob) Match(s string) bool {
	if len(s) < len(psacg.Prefix)+len(psacg.Contains)+len(psacg.Suffix) {
		return false
	}
	return exstrings.HasPrefixFold(s, psacg.Prefix) &&
		exstrings.HasSuffixFold(s, psacg.Suffix) &&
		exstrings.ContainsFold(s[len(psacg.Prefix):len(s)-len(psacg.Suffix)], psacg.Contains)
}
