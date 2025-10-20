package kong

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Tag represents the parsed state of Kong tags in a struct field tag.
type Tag struct {
	Ignored     bool // Field is ignored by Kong. ie. kong:"-"
	Cmd         bool
	Arg         bool
	Required    bool
	Optional    bool
	Name        string
	Help        string
	Type        string
	TypeName    string
	HasDefault  bool
	Default     string
	Format      string
	PlaceHolder string
	Envs        []string
	Short       rune
	Hidden      bool
	Sep         rune
	MapSep      rune
	Enum        string
	Group       string
	Xor         []string
	Vars        Vars
	Prefix      string // Optional prefix on anonymous structs. All sub-flags will have this prefix.
	EnvPrefix   string
	Embed       bool
	Aliases     []string
	Negatable   bool
	Passthrough bool

	// Storage for all tag keys for arbitrary lookups.
	items map[string][]string
}

func (t *Tag) String() string {
	out := []string{}
	for key, list := range t.items {
		for _, value := range list {
			out = append(out, fmt.Sprintf("%s:%q", key, value))
		}
	}
	return strings.Join(out, " ")
}

type tagChars struct {
	sep, quote, assign rune
	needsUnquote       bool
}

var kongChars = tagChars{sep: ',', quote: '\'', assign: '=', needsUnquote: false}
var bareChars = tagChars{sep: ' ', quote: '"', assign: ':', needsUnquote: true}

// nolint:gocyclo
func parseTagItems(tagString string, chr tagChars) (map[string][]string, error) {
	d := map[string][]string{}
	key := []rune{}
	value := []rune{}
	quotes := false
	inKey := true

	add := func() error {
		// Bare tags are quoted, therefore we need to unquote them in the same fashion reflect.Lookup() (implicitly)
		// unquotes "kong tags".
		s := string(value)

		if chr.needsUnquote && s != "" {
			if unquoted, err := strconv.Unquote(fmt.Sprintf(`"%s"`, s)); err == nil {
				s = unquoted
			} else {
				return fmt.Errorf("unquoting tag value `%s`: %w", s, err)
			}
		}

		d[string(key)] = append(d[string(key)], s)
		key = []rune{}
		value = []rune{}
		inKey = true

		return nil
	}

	runes := []rune(tagString)
	for idx := 0; idx < len(runes); idx++ {
		r := runes[idx]
		next := rune(0)
		eof := false
		if idx < len(runes)-1 {
			next = runes[idx+1]
		} else {
			eof = true
		}
		if !quotes && r == chr.sep {
			if err := add(); err != nil {
				return nil, err
			}

			continue
		}
		if r == chr.assign && inKey {
			inKey = false
			continue
		}
		if r == '\\' {
			if next == chr.quote {
				idx++

				// We need to keep the backslashes, otherwise subsequent unquoting cannot work
				if chr.needsUnquote {
					value = append(value, r)
				}

				r = chr.quote
			}
		} else if r == chr.quote {
			if quotes {
				quotes = false
				if next == chr.sep || eof {
					continue
				}
				return nil, fmt.Errorf("%v has an unexpected char at pos %v", tagString, idx)
			}
			quotes = true
			continue
		}
		if inKey {
			key = append(key, r)
		} else {
			value = append(value, r)
		}
	}
	if quotes {
		return nil, fmt.Errorf("%v is not quoted properly", tagString)
	}

	if err := add(); err != nil {
		return nil, err
	}

	return d, nil
}

func getTagInfo(ft reflect.StructField) (string, tagChars) {
	s, ok := ft.Tag.Lookup("kong")
	if ok {
		return s, kongChars
	}

	return string(ft.Tag), bareChars
}

func newEmptyTag() *Tag {
	return &Tag{items: map[string][]string{}}
}

func tagSplitFn(r rune) bool {
	return r == ',' || r == ' '
}

func parseTagString(s string) (*Tag, error) {
	items, err := parseTagItems(s, bareChars)
	if err != nil {
		return nil, err
	}
	t := &Tag{
		items: items,
	}
	err = hydrateTag(t, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", s, err)
	}
	return t, nil
}

func parseTag(parent reflect.Value, ft reflect.StructField) (*Tag, error) {
	if ft.Tag.Get("kong") == "-" {
		t := newEmptyTag()
		t.Ignored = true
		return t, nil
	}
	items, err := parseTagItems(getTagInfo(ft))
	if err != nil {
		return nil, err
	}
	t := &Tag{
		items: items,
	}
	err = hydrateTag(t, ft.Type)
	if err != nil {
		return nil, failField(parent, ft, "%s", err)
	}
	return t, nil
}

func hydrateTag(t *Tag, typ reflect.Type) error { // nolint: gocyclo
	var typeName string
	var isBool bool
	var isBoolPtr bool
	if typ != nil {
		typeName = typ.Name()
		isBool = typ.Kind() == reflect.Bool
		isBoolPtr = typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Bool
	}
	var err error
	t.Cmd = t.Has("cmd")
	t.Arg = t.Has("arg")
	required := t.Has("required")
	optional := t.Has("optional")
	if required && optional {
		return fmt.Errorf("can't specify both required and optional")
	}
	t.Required = required
	t.Optional = optional
	t.HasDefault = t.Has("default")
	t.Default = t.Get("default")
	// Arguments with defaults are always optional.
	if t.Arg && t.HasDefault {
		t.Optional = true
	} else if t.Arg && !optional { // Arguments are required unless explicitly made optional.
		t.Required = true
	}
	t.Name = t.Get("name")
	t.Help = t.Get("help")
	t.Type = t.Get("type")
	t.TypeName = typeName
	for _, env := range t.GetAll("env") {
		t.Envs = append(t.Envs, strings.FieldsFunc(env, tagSplitFn)...)
	}
	t.Short, err = t.GetRune("short")
	if err != nil && t.Get("short") != "" {
		return fmt.Errorf("invalid short flag name %q: %s", t.Get("short"), err)
	}
	t.Hidden = t.Has("hidden")
	t.Format = t.Get("format")
	t.Sep, _ = t.GetSep("sep", ',')
	t.MapSep, _ = t.GetSep("mapsep", ';')
	t.Group = t.Get("group")
	for _, xor := range t.GetAll("xor") {
		t.Xor = append(t.Xor, strings.FieldsFunc(xor, tagSplitFn)...)
	}
	t.Prefix = t.Get("prefix")
	t.EnvPrefix = t.Get("envprefix")
	t.Embed = t.Has("embed")
	negatable := t.Has("negatable")
	if negatable && !isBool && !isBoolPtr {
		return fmt.Errorf("negatable can only be set on booleans")
	}
	t.Negatable = negatable
	aliases := t.Get("aliases")
	if len(aliases) > 0 {
		t.Aliases = append(t.Aliases, strings.FieldsFunc(aliases, tagSplitFn)...)
	}
	t.Vars = Vars{}
	for _, set := range t.GetAll("set") {
		parts := strings.SplitN(set, "=", 2)
		if len(parts) == 0 {
			return fmt.Errorf("set should be in the form key=value but got %q", set)
		}
		t.Vars[parts[0]] = parts[1]
	}
	t.PlaceHolder = t.Get("placeholder")
	t.Enum = t.Get("enum")
	scalarType := typ == nil || !(typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || typ.Kind() == reflect.Ptr)
	if t.Enum != "" && !(t.Required || t.HasDefault) && scalarType {
		return fmt.Errorf("enum value is only valid if it is either required or has a valid default value")
	}
	passthrough := t.Has("passthrough")
	if passthrough && !t.Arg && !t.Cmd {
		return fmt.Errorf("passthrough only makes sense for positional arguments or commands")
	}
	t.Passthrough = passthrough
	return nil
}

// Has returns true if the tag contained the given key.
func (t *Tag) Has(k string) bool {
	_, ok := t.items[k]
	return ok
}

// Get returns the value of the given tag.
//
// Note that this will return the empty string if the tag is missing.
func (t *Tag) Get(k string) string {
	values := t.items[k]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// GetAll returns all encountered values for a tag, in the case of multiple occurrences.
func (t *Tag) GetAll(k string) []string {
	return t.items[k]
}

// GetBool returns true if the given tag looks like a boolean truth string.
func (t *Tag) GetBool(k string) (bool, error) {
	return strconv.ParseBool(t.Get(k))
}

// GetFloat parses the given tag as a float64.
func (t *Tag) GetFloat(k string) (float64, error) {
	return strconv.ParseFloat(t.Get(k), 64)
}

// GetInt parses the given tag as an int64.
func (t *Tag) GetInt(k string) (int64, error) {
	return strconv.ParseInt(t.Get(k), 10, 64)
}

// GetRune parses the given tag as a rune.
func (t *Tag) GetRune(k string) (rune, error) {
	value := t.Get(k)
	r, size := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError || size < len(value) {
		return 0, errors.New("invalid rune")
	}
	return r, nil
}

// GetSep parses the given tag as a rune separator, allowing for a default or none.
// The separator is returned, or -1 if "none" is specified. If the tag value is an
// invalid utf8 sequence, the default rune is returned as well as an error. If the
// tag value is more than one rune, the first rune is returned as well as an error.
func (t *Tag) GetSep(k string, dflt rune) (rune, error) {
	tv := t.Get(k)
	if tv == "none" {
		return -1, nil
	} else if tv == "" {
		return dflt, nil
	}
	r, size := utf8.DecodeRuneInString(tv)
	if r == utf8.RuneError {
		return dflt, fmt.Errorf(`%v:"%v" has a rune error`, k, tv)
	} else if size != len(tv) {
		return r, fmt.Errorf(`%v:"%v" is more than a single rune`, k, tv)
	}
	return r, nil
}
