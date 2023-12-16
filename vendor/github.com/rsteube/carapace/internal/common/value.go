// Package common code
package common

import (
	"sort"
	"strings"

	"github.com/rsteube/carapace/pkg/match"
	"github.com/rsteube/carapace/pkg/style"
)

// FromInvokedAction provides access to RawValues within an InvokedAction.
// It is intended for testing purposes in Sandbox (circumventing dependency issues).
var FromInvokedAction func(action interface{}) (Meta, RawValues)

// RawValue represents a completion candidate.
type RawValue struct {
	Value       string `json:"value"`
	Display     string `json:"display"`
	Description string `json:"description,omitempty"`
	Style       string `json:"style,omitempty"`
	Tag         string `json:"tag,omitempty"`
}

// TrimmedDescription returns the trimmed description.
func (r RawValue) TrimmedDescription() string {
	maxLength := 80
	description := strings.SplitN(r.Description, "\n", 2)[0]
	description = strings.TrimSpace(description)
	if len([]rune(description)) > maxLength {
		description = string([]rune(description)[:maxLength-3]) + "..."
	}
	return description
}

// RawValues is an alias for []RawValue.
type RawValues []RawValue

// RawValuesFrom creates RawValues from given values.
func RawValuesFrom(values ...string) RawValues {
	rawValues := make([]RawValue, len(values))
	for index, val := range values {
		rawValues[index] = RawValue{Value: val, Display: val, Style: style.Default}
	}
	return rawValues
}

func (r RawValues) Unique() RawValues {
	uniqueRawValues := make(map[string]RawValue)
	for _, value := range r {
		uniqueRawValues[value.Value] = value
	}

	rawValues := make([]RawValue, 0, len(uniqueRawValues))
	for _, value := range uniqueRawValues {
		rawValues = append(rawValues, value)
	}
	sort.Sort(ByDisplay(rawValues))
	return rawValues
}

func (r RawValues) contains(s string) bool {
	for _, value := range r {
		if value.Value == s {
			return true
		}
	}
	return false
}

// Filter filters values.
func (r RawValues) Filter(values ...string) RawValues {
	toremove := make(map[string]bool)
	for _, v := range values {
		toremove[v] = true
	}
	filtered := make([]RawValue, 0)
	for _, rawValue := range r {
		if _, ok := toremove[rawValue.Value]; !ok {
			filtered = append(filtered, rawValue)
		}
	}
	return filtered
}

// Retain retains given values.
func (r RawValues) Retain(values ...string) RawValues {
	toretain := make(map[string]bool)
	for _, v := range values {
		toretain[v] = true
	}
	filtered := make([]RawValue, 0)
	for _, rawValue := range r {
		if _, ok := toretain[rawValue.Value]; ok {
			filtered = append(filtered, rawValue)
		}
	}
	return filtered
}

// Decolor clears style for all values.
func (r RawValues) Decolor() RawValues {
	rawValues := make(RawValues, len(r))
	for index, value := range r {
		value.Style = ""
		rawValues[index] = value
	}
	return rawValues
}

// FilterPrefix filters values with given prefix.
func (r RawValues) FilterPrefix(prefix string) RawValues {
	filtered := make(RawValues, 0)
	for _, r := range r {
		if match.HasPrefix(r.Value, prefix) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (r RawValues) EachTag(f func(tag string, values RawValues)) {
	tagGroups := make(map[string]RawValues)
	for _, val := range r {
		if _, exists := tagGroups[val.Tag]; !exists {
			tagGroups[val.Tag] = make(RawValues, 0)
		}
		tagGroups[val.Tag] = append(tagGroups[val.Tag], val)
	}

	tags := make([]string, 0)
	for tag := range tagGroups {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		f(tag, tagGroups[tag])
	}
}

// ByValue alias to filter by value.
type ByValue []RawValue

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// ByDisplay alias to filter by display.
type ByDisplay []RawValue

func (a ByDisplay) Len() int           { return len(a) }
func (a ByDisplay) Less(i, j int) bool { return a[i].Display < a[j].Display }
func (a ByDisplay) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
