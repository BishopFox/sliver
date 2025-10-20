package carapace

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/carapace-sh/carapace/internal/common"
	"github.com/carapace-sh/carapace/internal/export"
	_shell "github.com/carapace-sh/carapace/internal/shell"
	"github.com/carapace-sh/carapace/pkg/match"
)

// InvokedAction is a logical alias for an Action whose (nested) callback was invoked.
type InvokedAction struct {
	action Action
}

func (ia InvokedAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ia.export())
}

func (ia InvokedAction) export() export.Export {
	return export.Export{Meta: ia.action.meta, Values: ia.action.rawValues}
}

// Filter filters given values.
//
//	a := carapace.ActionValues("A", "B", "C").Invoke(c)
//	b := a.Filter([]string{"B"}) // ["A", "C"]
func (ia InvokedAction) Filter(values ...string) InvokedAction {
	ia.action.rawValues = ia.action.rawValues.Filter(values...)
	return ia
}

// Merge merges InvokedActions (existing values are overwritten)
//
//	a := carapace.ActionValues("A", "B").Invoke(c)
//	b := carapace.ActionValues("B", "C").Invoke(c)
//	c := a.Merge(b) // ["A", "B", "C"]
func (ia InvokedAction) Merge(others ...InvokedAction) InvokedAction {
	for _, other := range append([]InvokedAction{ia}, others...) {
		ia.action.rawValues = append(ia.action.rawValues, other.action.rawValues...)
		ia.action.meta.Merge(other.action.meta)
	}
	ia.action.rawValues = ia.action.rawValues.Unique()
	return ia
}

// Prefix adds a prefix to values (only the ones inserted, not the display values)
//
//	carapace.ActionValues("melon", "drop", "fall").Invoke(c).Prefix("water")
func (ia InvokedAction) Prefix(prefix string) InvokedAction {
	for index, val := range ia.action.rawValues {
		ia.action.rawValues[index].Value = prefix + val.Value
	}
	if prefix != "" {
		ia.action.meta.Queries.Modify(fmt.Sprintf("prefix(%v)", prefix))
	}
	return ia
}

// Retain retains given values.
//
//	a := carapace.ActionValues("A", "B", "C").Invoke(c)
//	b := a.Retain([]string{"A", "C"}) // ["A", "C"]
func (ia InvokedAction) Retain(values ...string) InvokedAction {
	ia.action.rawValues = ia.action.rawValues.Retain(values...)
	return ia
}

// Suffix adds a suffx to values (only the ones inserted, not the display values)
//
//	carapace.ActionValues("apple", "melon", "orange").Invoke(c).Suffix("juice")
func (ia InvokedAction) Suffix(suffix string) InvokedAction {
	for index, val := range ia.action.rawValues {
		ia.action.rawValues[index].Value = val.Value + suffix
	}
	if suffix != "" {
		ia.action.meta.Queries.Modify(fmt.Sprintf("suffix(%v)", suffix))
	}
	return ia
}

// UidF TODO experimental
func (ia InvokedAction) UidF(f func(s string) (*url.URL, error)) InvokedAction {
	for index, v := range ia.action.rawValues {
		url, err := f(v.Value)
		if err != nil {
			return ActionMessage(err.Error()).Invoke(Context{})
		}
		ia.action.rawValues[index].Uid = url.String()
	}
	return ia
}

// ToA casts an InvokedAction to Action.
func (ia InvokedAction) ToA() Action {
	return ia.action
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
func (ia InvokedAction) ToMultiPartsA(dividers ...string) Action {
	return ActionCallback(func(c Context) Action {
		splittedCV := tokenize(c.Value, dividers...)

		uniqueVals := make(map[string]common.RawValue)
		for _, val := range ia.action.rawValues {
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
							Tag:         val.Tag,
							Uid:         val.Uid,
						}
					} else {
						uniqueVals[v] = common.RawValue{
							Value:       v,
							Display:     d,
							Description: "",
							Style:       "",
							Tag:         val.Tag,
							Uid:         "",
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

func (ia InvokedAction) Query(q string) InvokedAction {
	ia.action.meta.Queries.Add(q)
	return ia
}

func (ia InvokedAction) value(shell string, value string) string {
	return _shell.Value(shell, value, ia.action.meta, ia.action.rawValues)
}

func init() {
	common.FromInvokedAction = func(i interface{}) (common.Meta, common.RawValues) {
		if invoked, ok := i.(InvokedAction); ok {
			return invoked.action.meta, invoked.action.rawValues
		}
		return common.Meta{}, nil
	}
}
