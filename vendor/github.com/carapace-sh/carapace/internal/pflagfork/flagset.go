package pflagfork

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

type FlagSet struct {
	*pflag.FlagSet
}

func (f FlagSet) IsInterspersed() bool {
	if fv := reflect.ValueOf(f.FlagSet).Elem().FieldByName("interspersed"); fv.IsValid() {
		return fv.Bool()
	}
	return false
}

func (f FlagSet) IsPosix() bool {
	if method := reflect.ValueOf(f.FlagSet).MethodByName("IsPosix"); method.IsValid() {
		if values := method.Call([]reflect.Value{}); len(values) == 1 && values[0].Kind() == reflect.Bool {
			return values[0].Bool()
		}
	}
	return true
}

func (f FlagSet) Prefix() rune {
	if method := reflect.ValueOf(f.FlagSet).MethodByName("Prefix"); method.IsValid() {
		if values := method.Call([]reflect.Value{}); len(values) == 1 && values[0].Kind() == reflect.Int32 {
			return rune(values[0].Int())
		}
	}
	return '-'
}

func (f FlagSet) IsShorthandSeries(arg string) bool {
	p := string(f.Prefix())
	re := regexp.MustCompile("^" + p + "(?P<shorthand>[^" + p + "].*)")
	return re.MatchString(arg) && f.IsPosix()
}

func (f FlagSet) IsMutuallyExclusive(flag *pflag.Flag) bool {
	if groups, ok := flag.Annotations["cobra_annotation_mutually_exclusive"]; ok {
		for _, group := range groups {
			for name := range strings.SplitSeq(group, " ") {
				if other := f.Lookup(name); other != nil && other.Changed && other != flag {
					return true
				}
			}
		}
	}
	return false
}

func (f *FlagSet) VisitAll(fn func(*Flag)) {
	prefix := f.Prefix()
	f.FlagSet.VisitAll(func(flag *pflag.Flag) {
		fn(&Flag{Flag: flag, FlagPrefix: prefix, Args: []string{}})
	})

}

func (fs FlagSet) LookupArg(arg string) (result *Flag) {
	isPosix := fs.IsPosix()
	prefix := string(fs.Prefix())
	doublePrefix := prefix + prefix

	switch {
	case strings.HasPrefix(arg, doublePrefix):
		return fs.lookupPosixLonghandArg(arg)
	case isPosix:
		return fs.lookupPosixShorthandArg(arg)
	case !isPosix:
		// In non-posix mode, try longhand first (single dash with name)
		// to handle cases where name overlaps with shorthand
		if result = fs.LookupNonPosixLonghandArg(arg); result != nil {
			return result
		}
		return fs.lookupNonPosixShorthandArg(arg)
	}
	return
}

func (fs FlagSet) ShorthandLookup(name string) *Flag {
	if f := fs.FlagSet.ShorthandLookup(name); f != nil {
		return &Flag{
			Flag:       f,
			FlagPrefix: fs.Prefix(),
			Args:       []string{},
		}
	}
	return nil
}

func (fs FlagSet) lookupPosixLonghandArg(arg string) (flag *Flag) {
	prefix := string(fs.Prefix())
	doublePrefix := prefix + prefix
	if !strings.HasPrefix(arg, doublePrefix) {
		return nil
	}

	fs.VisitAll(func(f *Flag) { // TODO needs to be sorted to try longest matching first
		if flag != nil || f.GetMode() != Default {
			return
		}

		splitted := strings.SplitAfterN(arg, string(f.OptargDelimiter()), 2)
		if strings.TrimSuffix(splitted[0], string(f.OptargDelimiter())) == doublePrefix+f.Name {
			// AcceptAttached does not apply to longhand flags (--flagvalue is not valid);
			// only AcceptDelimited (--flag=value) and AcceptNext (--flag value) are checked here.
			if len(splitted) > 1 && !f.AcceptsDelimited() {
				return // flag doesn't accept delimited style (--flag=value)
			}
			if len(splitted) == 1 && !f.AcceptsNext() && f.NoOptDefVal == "" {
				return // flag doesn't accept next style and has no default
			}
			flag = f
			flag.ArgPrefix = splitted[0]
			if len(splitted) > 1 {
				flag.Args = splitted[1:]
			}
		}
	})
	return
}

func (fs FlagSet) lookupPosixShorthandArg(arg string) *Flag {
	prefix := string(fs.Prefix())
	if !strings.HasPrefix(arg, prefix) || !fs.IsPosix() || len(arg) < 2 {
		return nil
	}

	for index, r := range arg[1:] {
		index += 1
		flag := fs.ShorthandLookup(string(r))

		if flag == nil {
			return flag
		}

		atEnd := len(arg) == index+1
		hasDelimiter := !atEnd && arg[index+1] == byte(flag.OptargDelimiter())
		hasAttached := !atEnd && !hasDelimiter

		// Reject argument styles the flag does not accept
		switch {
		case hasDelimiter && !flag.AcceptsDelimited():
			continue
		case hasAttached && !flag.AcceptsAttached():
			continue
		case atEnd && !flag.AcceptsNext() && !flag.AcceptsAttached():
			continue
		}

		switch {
		case atEnd:
			flag.ArgPrefix = arg
			return flag
		case hasDelimiter && len(arg) > index+2:
			flag.ArgPrefix = arg[:index+2]
			flag.Args = []string{arg[index+2:]}
			return flag
		case hasDelimiter:
			flag.ArgPrefix = arg[:index+2]
			flag.Args = []string{""}
			return flag
		case !flag.IsOptarg() && len(arg) > index+1:
			flag.ArgPrefix = arg[:index+1]
			flag.Args = []string{arg[index+1:]}
			return flag
		}
	}
	return nil
}

func (fs FlagSet) lookupNonPosixShorthandArg(arg string) (result *Flag) { // TODO pretty much duplicates longhand lookup
	prefix := string(fs.Prefix())
	if !strings.HasPrefix(arg, prefix) {
		return nil
	}

	fs.VisitAll(func(f *Flag) {
		splitted := strings.SplitAfterN(arg, string(f.OptargDelimiter()), 2)
		baseArg := strings.TrimSuffix(splitted[0], string(f.OptargDelimiter()))

		// Check ArgumentStyle constraints
		if len(splitted) > 1 && !f.AcceptsDelimited() {
			return // flag doesn't accept delimited style
		}

		if baseArg == prefix+f.Shorthand {
			candidate := f
			candidate.ArgPrefix = splitted[0]
			if len(splitted) > 1 {
				candidate.Args = splitted[1:]
			}
			if result == nil || len(candidate.ArgPrefix) > len(result.ArgPrefix) {
				result = candidate
			}
			return
		}

		// optarg flags with a non-standard delimiter (e.g. -1) accept
		// directly attached values: -rvalue matches flag "r" with arg "value"
		if f.IsOptarg() && f.DelimiterDisabled() && f.AcceptsAttached() &&
			strings.HasPrefix(arg, prefix+f.Shorthand) && len(arg) > len(prefix+f.Shorthand) {
			candidate := f
			candidate.ArgPrefix = prefix + f.Shorthand
			candidate.Args = []string{arg[len(prefix+f.Shorthand):]}
			if result == nil || len(candidate.ArgPrefix) > len(result.ArgPrefix) {
				result = candidate
			}
		}
	})
	return
}

// LookupNonPosixLonghandArg looks up a non-posix longhand argument (single prefix with name)
func (fs FlagSet) LookupNonPosixLonghandArg(arg string) (result *Flag) {
	prefix := string(fs.Prefix())
	doublePrefix := prefix + prefix
	if !strings.HasPrefix(arg, prefix) || strings.HasPrefix(arg, doublePrefix) {
		return nil
	}

	fs.VisitAll(func(f *Flag) {
		if f.GetMode() != NameAsShorthand {
			return
		}

		splitted := strings.SplitAfterN(arg, string(f.OptargDelimiter()), 2)
		baseArg := strings.TrimSuffix(splitted[0], string(f.OptargDelimiter()))

		// Check ArgumentStyle constraints
		if len(splitted) > 1 && !f.AcceptsDelimited() {
			return // flag doesn't accept delimited style
		}

		if baseArg == prefix+f.Name {
			if len(splitted) == 1 && !f.AcceptsNext() && f.NoOptDefVal == "" {
				return // flag doesn't accept next style and has no default
			}

			result = f
			result.ArgPrefix = splitted[0]
			if len(splitted) > 1 {
				result.Args = splitted[1:]
			}
			return
		}

		// optarg flags with a non-standard delimiter (e.g. -1) accept
		// directly attached values: -namevalue matches flag "name" with arg "value"
		if f.IsOptarg() && f.DelimiterDisabled() && f.AcceptsAttached() &&
			strings.HasPrefix(arg, prefix+f.Name) && len(arg) > len(prefix+f.Name) {
			candidate := f
			candidate.ArgPrefix = prefix + f.Name
			candidate.Args = []string{arg[len(prefix+f.Name):]}
			if result == nil || len(candidate.ArgPrefix) > len(result.ArgPrefix) {
				result = candidate
			}
		}
	})
	return
}
