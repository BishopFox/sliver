package pflagfork

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/pflag"
)

// mode defines how flags are represented.
type mode int

const (
	Default         mode = iota // default behaviour
	ShorthandOnly               // only the shorthand should be used
	NameAsShorthand             // non-posix mode where the name is also added as shorthand (single `-` prefix)
)

type Flag struct {
	*pflag.Flag
}

func (f Flag) Nargs() int {
	if field := reflect.ValueOf(f.Flag).Elem().FieldByName("Nargs"); field.IsValid() && field.Kind() == reflect.Int {
		return int(field.Int())
	}
	return 0
}

func (f Flag) Mode() mode {
	if field := reflect.ValueOf(f.Flag).Elem().FieldByName("Mode"); field.IsValid() && field.Kind() == reflect.Int {
		return mode(field.Int())
	}
	return Default
}

func (f Flag) OptargDelimiter() rune {
	if field := reflect.ValueOf(f.Flag).Elem().FieldByName("OptargDelimiter"); field.IsValid() && field.Kind() == reflect.Int32 {
		return (rune(field.Int()))
	}
	return '='
}

func (f Flag) IsRepeatable() bool {
	if strings.Contains(f.Value.Type(), "Slice") ||
		strings.Contains(f.Value.Type(), "Array") ||
		f.Value.Type() == "count" {
		return true
	}
	return false
}

func (f Flag) Split(arg string) (prefix, optarg string) {
	delimiter := string(f.OptargDelimiter())
	splitted := strings.SplitN(arg, delimiter, 2)
	return splitted[0] + delimiter, splitted[1]
}

func (f Flag) Matches(arg string, posix bool) bool {
	if !strings.HasPrefix(arg, "-") { // not a flag
		return false
	}

	switch {

	case strings.HasPrefix(arg, "--"):
		name := strings.TrimPrefix(arg, "--")
		name = strings.SplitN(name, string(f.OptargDelimiter()), 2)[0]

		switch f.Mode() {
		case ShorthandOnly, NameAsShorthand:
			return false
		default:
			return name == f.Name
		}

	case !posix:
		name := strings.TrimPrefix(arg, "-")
		name = strings.SplitN(name, string(f.OptargDelimiter()), 2)[0]

		if name == "" {
			return false
		}

		switch f.Mode() {
		case ShorthandOnly:
			return name == f.Shorthand
		default:
			return name == f.Name || name == f.Shorthand
		}

	default:
		if f.Shorthand != "" {
			return strings.HasSuffix(arg, f.Shorthand)
		}
		return false
	}
}

func (f Flag) TakesValue() bool {
	switch f.Value.Type() {
	case "bool", "boolSlice", "count":
		return false
	default:
		return true
	}
}

func (f Flag) IsOptarg() bool {
	return f.NoOptDefVal != ""
}

func (f Flag) Style() string {
	switch {
	case !f.TakesValue():
		return style.Carapace.FlagNoArg
	case f.IsOptarg():
		return style.Carapace.FlagOptArg
	case f.Nargs() != 0:
		return style.Carapace.FlagMultiArg
	default:
		return style.Carapace.FlagArg
	}
}

func (f Flag) Definition() string {
	var definition string
	switch f.Mode() {
	case ShorthandOnly:
		definition = fmt.Sprintf("-%v", f.Shorthand)
	case NameAsShorthand:
		definition = fmt.Sprintf("-%v, -%v", f.Shorthand, f.Name)
	default:
		switch f.Shorthand {
		case "":
			definition = fmt.Sprintf("--%v", f.Name)
		default:
			definition = fmt.Sprintf("-%v, --%v", f.Shorthand, f.Name)
		}
	}

	if f.IsRepeatable() {
		definition += "*"
	}

	switch {
	case f.IsOptarg():
		switch f.Value.Type() {
		case "bool", "boolSlice", "count":
		default:
			definition += "?"
		}
	case f.TakesValue():
		definition += "="
	}

	return definition
}
