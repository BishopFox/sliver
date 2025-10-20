package carapace

import (
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/export"
	_shell "github.com/rsteube/carapace/internal/shell"
	"github.com/rsteube/carapace/pkg/match"
)

// InvokedAction is a logical alias for an Action whose (nested) callback was invoked.
type InvokedAction struct {
	Action
}

func (a InvokedAction) export() export.Export {
	return export.Export{Meta: a.meta, Values: a.rawValues}
}

// Filter filters given values.
//
//	a := carapace.ActionValues("A", "B", "C").Invoke(c)
//	b := a.Filter([]string{"B"}) // ["A", "C"]
func (a InvokedAction) Filter(values ...string) InvokedAction {
	a.rawValues = a.rawValues.Filter(values...)
	return a
}

// Merge merges InvokedActions (existing values are overwritten)
//
//	a := carapace.ActionValues("A", "B").Invoke(c)
//	b := carapace.ActionValues("B", "C").Invoke(c)
//	c := a.Merge(b) // ["A", "B", "C"]
func (a InvokedAction) Merge(others ...InvokedAction) InvokedAction {
	for _, other := range append([]InvokedAction{a}, others...) {
		a.rawValues = append(a.rawValues, other.rawValues...)
		a.meta.Merge(other.meta)
	}
	a.rawValues = a.rawValues.Unique()
	return a
}

// Prefix adds a prefix to values (only the ones inserted, not the display values)
//
//	carapace.ActionValues("melon", "drop", "fall").Invoke(c).Prefix("water")
func (a InvokedAction) Prefix(prefix string) InvokedAction {
	for index, val := range a.rawValues {
		a.rawValues[index].Value = prefix + val.Value
	}
	return a
}

// Retain retains given values.
//
//	a := carapace.ActionValues("A", "B", "C").Invoke(c)
//	b := a.Retain([]string{"A", "C"}) // ["A", "C"]
func (a InvokedAction) Retain(values ...string) InvokedAction {
	a.rawValues = a.rawValues.Retain(values...)
	return a
}

// Suffix adds a suffx to values (only the ones inserted, not the display values)
//
//	carapace.ActionValues("apple", "melon", "orange").Invoke(c).Suffix("juice")
func (a InvokedAction) Suffix(suffix string) InvokedAction {
	for index, val := range a.rawValues {
		a.rawValues[index].Value = val.Value + suffix
	}
	return a
}

// ToA casts an InvokedAction to Action.
func (a InvokedAction) ToA() Action {
	return a.Action
}

func tokenize(s string, dividers ...string) []string {
	if len(dividers) == 0 {
		return []string{s}
	}

	result := make([]string, 0)
	for _, word := range strings.SplitAfter(s, dividers[0]) {
		tokens := tokenize(strings.TrimSuffix(word, dividers[0]), dividers[1:]...)
		if len(tokens) > 0 && strings.HasSuffix(word, dividers[0]) {
			tokens[len(tokens)-1] = tokens[len(tokens)-1] + dividers[0]
		}
		result = append(result, tokens...)
	}
	return result
}

// ToMultiPartsA create an ActionMultiParts from values with given dividers
//
//	a := carapace.ActionValues("A/B/C", "A/C", "B/C", "C").Invoke(c)
//	b := a.ToMultiPartsA("/") // completes segments separately (first one is ["A/", "B/", "C"])
func (a InvokedAction) ToMultiPartsA(dividers ...string) Action {
	return ActionCallback(func(c Context) Action {
		splittedCV := tokenize(c.Value, dividers...)

		uniqueVals := make(map[string]common.RawValue)
		for _, val := range a.rawValues {
			if match.HasPrefix(val.Value, c.Value) {
				if splitted := tokenize(val.Value, dividers...); len(splitted) >= len(splittedCV) {
					v := strings.Join(splitted[:len(splittedCV)], "")
					d := splitted[len(splittedCV)-1]

					if len(splitted) == len(splittedCV) {
						uniqueVals[v] = common.RawValue{
							Value:       v,
							Display:     d,
							Description: val.Description,
							Style:       val.Style,
						}
					} else {
						uniqueVals[v] = common.RawValue{
							Value:       v,
							Display:     d,
							Description: "",
							Style:       "",
						}
					}
				}
			}
		}

		vals := make([]common.RawValue, 0)
		for _, val := range uniqueVals {
			vals = append(vals, val)
		}

		a := Action{rawValues: vals}
		for _, divider := range dividers {
			if runes := []rune(divider); len(runes) == 0 {
				a.meta.Nospace.Add('*')
				break
			} else {
				a.meta.Nospace.Add(runes[len(runes)-1])
			}
		}
		return a
	})
}

func (a InvokedAction) value(shell string, value string) string {
	return _shell.Value(shell, value, a.meta, a.rawValues)
}

func init() {
	common.FromInvokedAction = func(i interface{}) (common.Meta, common.RawValues) {
		if a, ok := i.(InvokedAction); ok {
			return a.meta, a.rawValues
		}
		return common.Meta{}, nil
	}
}
