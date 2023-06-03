package completion

import (
	"sort"
	"strings"
)

// SuffixMatcher is a type managing suffixes for a given list of completions.
type SuffixMatcher struct {
	string
	pos int // Used to know if the saved suffix matcher is deprecated
}

// Add adds new suffixes to the matcher.
func (sm *SuffixMatcher) Add(suffixes ...rune) {
	if strings.Contains(sm.string, "*") || strings.Contains(string(suffixes), "*") {
		sm.string = "*"

		return
	}

	unique := []rune(sm.string)

	for _, r := range suffixes {
		if !strings.Contains(sm.string, string(r)) {
			unique = append(unique, r)
		}
	}

	sort.Sort(byRune(unique))
	sm.string = string(unique)
}

// Merge merges two suffix matchers.
func (sm *SuffixMatcher) Merge(other SuffixMatcher) {
	for _, r := range other.string {
		sm.Add(r)
	}
}

// Matches returns true if the given string matches one of the suffixes.
func (sm SuffixMatcher) Matches(s string) bool {
	for _, r := range sm.string {
		if r == '*' || strings.HasSuffix(s, string(r)) {
			return true
		}
	}

	return false
}

type byRune []rune

func (r byRune) Len() int           { return len(r) }
func (r byRune) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r byRune) Less(i, j int) bool { return r[i] < r[j] }
