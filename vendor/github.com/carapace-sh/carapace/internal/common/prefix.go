package common

import (
	"encoding/json"
	"sort"
	"strings"
)

type PrefixMatcher struct {
	string
}

func (pm *PrefixMatcher) Add(prefixes ...rune) {
	if strings.Contains(pm.string, "*") || strings.Contains(string(prefixes), "*") {
		pm.string = "*"
		return
	}

	unique := []rune(pm.string)
	for _, r := range prefixes {
		if !strings.Contains(pm.string, string(r)) {
			unique = append(unique, r)
		}
	}
	sort.Sort(ByRune(unique))
	pm.string = string(unique)
}

func (pm *PrefixMatcher) Merge(other PrefixMatcher) {
	for _, r := range other.string {
		pm.Add(r)
	}
}

func (pm PrefixMatcher) Matches(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range pm.string {
		if r == '*' || strings.HasPrefix(s, string(r)) {
			return true
		}
	}
	return false
}

func (pm PrefixMatcher) MarshalJSON() ([]byte, error) {
	return json.Marshal(pm.string)
}

func (pm *PrefixMatcher) UnmarshalJSON(data []byte) (err error) {
	if err = json.Unmarshal(data, &pm.string); err != nil {
		return err
	}
	return
}
