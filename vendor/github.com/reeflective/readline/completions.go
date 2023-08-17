package readline

import (
	"fmt"

	"github.com/reeflective/readline/internal/completion"
)

// Completion represents a completion candidate.
type Completion = completion.Candidate

// Completions holds all completions candidates and their associated data,
// including usage strings, messages, and suffix matchers for autoremoval.
// Some of those additional settings will apply to all contained candidates,
// except when these candidates have their own corresponding settings.
type Completions struct {
	values   completion.RawValues
	messages completion.Messages
	noSpace  completion.SuffixMatcher
	usage    string
	listLong map[string]bool
	noSort   map[string]bool
	listSep  map[string]string
	pad      map[string]bool
	escapes  map[string]bool

	// Initially this will be set to the part of the current word
	// from the beginning of the word up to the position of the cursor.
	// It may be altered to give a prefix for all matches.
	PREFIX string

	// Initially this will be set to the part of the current word,
	// starting from the cursor position up to the end of the word.
	// It may be altered so that inserted completions don't overwrite
	// entirely any suffix when completing in the middle of a word.
	SUFFIX string
}

// CompleteValues completes arbitrary keywords (values).
func CompleteValues(values ...string) Completions {
	vals := make([]Completion, 0, len(values))
	for _, val := range values {
		vals = append(vals, Completion{Value: val, Display: val, Description: ""})
	}

	return Completions{values: vals}
}

// CompleteStyledValues is like CompleteValues but also accepts a style.
func CompleteStyledValues(values ...string) Completions {
	if length := len(values); length%2 != 0 {
		return CompleteMessage("invalid amount of arguments [CompleteStyledValues]: %v", length)
	}

	vals := make([]Completion, 0, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		vals = append(vals, Completion{Value: values[i], Display: values[i], Description: "", Style: values[i+1]})
	}

	return Completions{values: vals}
}

// CompleteValuesDescribed completes arbitrary key (values) with an additional description (value, description pairs).
func CompleteValuesDescribed(values ...string) Completions {
	if length := len(values); length%2 != 0 {
		return CompleteMessage("invalid amount of arguments [CompleteValuesDescribed]: %v", length)
	}

	vals := make([]Completion, 0, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		vals = append(vals, Completion{Value: values[i], Display: values[i], Description: values[i+1]})
	}

	return Completions{values: vals}
}

// CompleteStyledValuesDescribed is like CompleteValues but also accepts a style.
func CompleteStyledValuesDescribed(values ...string) Completions {
	if length := len(values); length%3 != 0 {
		return CompleteMessage("invalid amount of arguments [CompleteStyledValuesDescribed]: %v", length)
	}

	vals := make([]Completion, 0, len(values)/3)
	for i := 0; i < len(values); i += 3 {
		vals = append(vals, Completion{Value: values[i], Display: values[i], Description: values[i+1], Style: values[i+2]})
	}

	return Completions{values: vals}
}

// CompleteMessage ads a help message to display along with
// or in places where no completions can be generated.
func CompleteMessage(msg string, args ...any) Completions {
	comps := Completions{}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	comps.messages.Add(msg)

	return comps
}

// CompleteRaw directly accepts a list of prepared Completion values.
func CompleteRaw(values []Completion) Completions {
	return Completions{values: completion.RawValues(values)}
}

// Message displays a help messages in places where no completions can be generated.
func Message(msg string, args ...any) Completions {
	comps := Completions{}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	comps.messages.Add(msg)

	return comps
}

// Suppress suppresses specific error messages using regular expressions.
func (c Completions) Suppress(expr ...string) Completions {
	if err := c.messages.Suppress(expr...); err != nil {
		return CompleteMessage(err.Error())
	}

	return c
}

// NoSpace disables space suffix for given characters (or all if none are given).
// These suffixes will be used for all completions that have not specified their
// own suffix-matching patterns.
// This is used for slash-autoremoval in path completions, comma-separated completions, etc.
func (c Completions) NoSpace(suffixes ...rune) Completions {
	if len(suffixes) == 0 {
		c.noSpace.Add('*')
	}

	c.noSpace.Add(suffixes...)

	return c
}

// Prefix adds a prefix to values (only the ones inserted, not the display values)
//
//	a := CompleteValues("melon", "drop", "fall").Invoke(c)
//	b := a.Prefix("water") // ["watermelon", "waterdrop", "waterfall"] but display still ["melon", "drop", "fall"]
func (c Completions) Prefix(prefix string) Completions {
	for index, val := range c.values {
		c.values[index].Value = prefix + val.Value
	}

	return c
}

// Suffix adds a suffx to values (only the ones inserted, not the display values)
//
//	a := CompleteValues("apple", "melon", "orange").Invoke(c)
//	b := a.Suffix("juice") // ["applejuice", "melonjuice", "orangejuice"] but display still ["apple", "melon", "orange"]
func (c Completions) Suffix(suffix string) Completions {
	for index, val := range c.values {
		c.values[index].Value = val.Value + suffix
	}

	return c
}

// Usage sets the usage.
func (c Completions) Usage(usage string, args ...any) Completions {
	return c.UsageF(func() string {
		return fmt.Sprintf(usage, args...)
	})
}

// UsageF sets the usage using a function.
func (c Completions) UsageF(f func() string) Completions {
	if usage := f(); usage != "" {
		c.usage = usage
	}

	return c
}

// Style sets the style, accepting cterm color codes, eg. 255, 30, etc.
//
//	CompleteValues("yes").Style("35")
//	CompleteValues("no").Style("255")
func (c Completions) Style(style string) Completions {
	return c.StyleF(func(s string) string {
		return style
	})
}

// StyleR sets the style using a reference
//
//	CompleteValues("value").StyleR(&style.Value)
//	CompleteValues("description").StyleR(&style.Value)
func (c Completions) StyleR(style *string) Completions {
	if style != nil {
		return c.Style(*style)
	}

	return c
}

// StyleF sets the style using a function
//
//	CompleteValues("dir/", "test.txt").StyleF(myStyleFunc)
//	CompleteValues("true", "false").StyleF(styleForKeyword)
func (c Completions) StyleF(f func(s string) string) Completions {
	for index, v := range c.values {
		c.values[index].Style = f(v.Value)
	}

	return c
}

// Tag sets the tag.
//
//	CompleteValues("192.168.1.1", "127.0.0.1").Tag("interfaces").
func (c Completions) Tag(tag string) Completions {
	return c.TagF(func(value string) string {
		return tag
	})
}

// TagF sets the tag using a function.
//
//	CompleteValues("192.168.1.1", "127.0.0.1").TagF(func(value string) string {
//		return "interfaces"
//	})
func (c Completions) TagF(f func(value string) string) Completions {
	for index, v := range c.values {
		c.values[index].Tag = f(v.Value)
	}

	return c
}

// DisplayList forces the completions to be list below each other as a list.
// A series of tags can be passed to restrict this to these tags. If empty,
// will be applied to all completions.
func (c Completions) DisplayList(tags ...string) Completions {
	if c.listLong == nil {
		c.listLong = make(map[string]bool)
	}

	if len(tags) == 0 {
		c.listLong["*"] = true
	}

	for _, tag := range tags {
		c.listLong[tag] = true
	}

	return c
}

// ListSeparator accepts a custom separator to use between the candidates and their descriptions.
// If more than one separator is given, the list is considered to be a map of tag:separators, in
// which case it will fail if the list has an odd number of values.
//
// If one only one value is given, will apply to all completions (and their tags if any).
// If no value is given, no modifications will be made.
func (c Completions) ListSeparator(seps ...string) Completions {
	if c.listSep == nil {
		c.listSep = make(map[string]string)
	}

	if length := len(seps); len(seps) > 1 && length%2 != 0 {
		return CompleteMessage("invalid amount of arguments (ListSeparator): %v", length)
	}

	if len(seps) == 1 {
		if len(c.listSep) == 0 {
			c.listSep["*"] = seps[0]
		} else {
			for tag := range c.listSep {
				c.listSep[tag] = seps[0]
			}
		}
	} else {
		for i := 0; i < len(seps); i += 2 {
			c.listSep[seps[i]] = seps[i+1]
		}
	}

	return c
}

// NoSort forces the completions not to sort the completions in alphabetical order.
// A series of tags can be passed to restrict this to these tags. If empty, will be
// applied to all completions.
func (c Completions) NoSort(tags ...string) Completions {
	if c.noSort == nil {
		c.noSort = make(map[string]bool)
	}

	if len(tags) == 0 {
		c.noSort["*"] = true
	}

	for _, tag := range tags {
		c.noSort[tag] = true
	}

	return c
}

// Filter filters given values (this should be done before any call
// to Prefix/Suffix as those alter the values being filtered)
//
//	a := CompleteValues("A", "B", "C").Invoke(c)
//	b := a.Filter([]string{"B"}) // ["A", "C"]
func (c Completions) Filter(values []string) Completions {
	c.values = c.values.Filter(values...)
	return c
}

// JustifyDescriptions accepts a list of tags for which descriptions (if any), will be left justified.
// If no arguments are given, description justification (padding) will apply to all tags.
func (c Completions) JustifyDescriptions(tags ...string) Completions {
	if c.pad == nil {
		c.pad = make(map[string]bool)
	}

	if len(tags) == 0 {
		c.pad["*"] = true
	}

	for _, tag := range tags {
		c.pad[tag] = true
	}

	return c
}

// PreserveEscapes forces the completion engine to keep all escaped characters in
// the inserted completion (c.Value of the Completion type). By default, those are
// stripped out and only kept in the completion.Display. If no arguments are given,
// escape sequence preservation will apply to all tags.
//
// This has very few use cases: one of them might be when you want to read a string
// from the readline shell that might include color sequences to be preserved.
// In such cases, this function gives a double advantage: the resulting completion
// is still "color-displayed" in the input line, and returned to the readline with
// them. A classic example is where you want to read a prompt string configuration.
//
// Note that this option might have various undefined behaviors when it comes to
// completion prefix matching, insertion, removal and related things.
func (c Completions) PreserveEscapes(tags ...string) Completions {
	if c.escapes == nil {
		c.escapes = make(map[string]bool)
	}

	if len(tags) == 0 {
		c.escapes["*"] = true
	}

	for _, tag := range tags {
		c.escapes[tag] = true
	}

	return c
}

// Merge merges Completions (existing values are overwritten)
//
//	a := CompleteValues("A", "B").Invoke(c)
//	b := CompleteValues("B", "C").Invoke(c)
//	c := a.Merge(b) // ["A", "B", "C"]
func (c Completions) Merge(others ...Completions) Completions {
	uniqueRawValues := make(map[string]Completion)

	for _, other := range append([]Completions{c}, others...) {
		for _, c := range other.values {
			uniqueRawValues[c.Value] = c
		}
	}

	for _, other := range others {
		c.merge(other)
	}

	rawValues := make([]Completion, 0, len(uniqueRawValues))
	for _, c := range uniqueRawValues {
		rawValues = append(rawValues, c)
	}

	c.values = rawValues

	return c
}

// EachValue runs a function on each value, overwriting with the returned one.
func (c *Completions) EachValue(tagF func(comp Completion) Completion) {
	for index, v := range c.values {
		c.values[index] = tagF(v)
	}
}

func (c *Completions) merge(other Completions) {
	if other.usage != "" {
		c.usage = other.usage
	}

	c.noSpace.Merge(other.noSpace)
	c.messages.Merge(other.messages)

	for tag := range other.listLong {
		if _, found := c.listLong[tag]; !found {
			c.listLong[tag] = true
		}
	}

	for tag := range other.noSort {
		if _, found := c.noSort[tag]; !found {
			c.noSort[tag] = true
		}
	}

	for tag := range other.listSep {
		if _, found := c.listSep[tag]; !found {
			c.listSep[tag] = other.listSep[tag]
		}
	}

	for tag := range other.pad {
		if _, found := c.pad[tag]; !found {
			c.pad[tag] = other.pad[tag]
		}
	}
}

func (c *Completions) convert() completion.Values {
	comps := completion.AddRaw(c.values)

	comps.Messages = c.messages
	comps.NoSpace = c.noSpace
	comps.Usage = c.usage
	comps.ListLong = c.listLong
	comps.NoSort = c.noSort
	comps.ListSep = c.listSep
	comps.Pad = c.pad
	comps.Escapes = c.escapes

	comps.PREFIX = c.PREFIX
	comps.SUFFIX = c.SUFFIX

	return comps
}
