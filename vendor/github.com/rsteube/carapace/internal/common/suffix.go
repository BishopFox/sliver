package common

import (
	"encoding/json"
	"sort"
	"strings"
)

type SuffixMatcher struct {
	string
}

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
	sort.Sort(ByRune(unique))
	sm.string = string(unique)
}

func (sm *SuffixMatcher) Merge(other SuffixMatcher) {
	for _, r := range other.string {
		sm.Add(r)
	}
}

func (sm SuffixMatcher) Matches(s string) bool {
	for _, r := range sm.string {
		if r == '*' || strings.HasSuffix(s, string(r)) {
			return true
		}
	}
	return false
}

func (sm SuffixMatcher) MarshalJSON() ([]byte, error) {
	return json.Marshal(sm.string)
}

func (sm *SuffixMatcher) UnmarshalJSON(data []byte) (err error) {
	if err = json.Unmarshal(data, &sm.string); err != nil {
		return err
	}
	return
}

type ByRune []rune

func (r ByRune) Len() int           { return len(r) }
func (r ByRune) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ByRune) Less(i, j int) bool { return r[i] < r[j] }
