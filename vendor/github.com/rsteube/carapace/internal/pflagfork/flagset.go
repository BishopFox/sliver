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

func (f FlagSet) IsPosix() bool {
	if method := reflect.ValueOf(f.FlagSet).MethodByName("IsPosix"); method.IsValid() {
		if values := method.Call([]reflect.Value{}); len(values) == 1 && values[0].Kind() == reflect.Bool {
			return values[0].Bool()
		}
	}
	return true
}

func (f FlagSet) IsShorthandSeries(arg string) bool {
	re := regexp.MustCompile("^-(?P<shorthand>[^-=]+)")
	return re.MatchString(arg) && f.IsPosix()
}

func (f FlagSet) IsMutuallyExclusive(flag *pflag.Flag) bool {
	if groups, ok := flag.Annotations["cobra_annotation_mutually_exclusive"]; ok {
		for _, group := range groups {
			for _, name := range strings.Split(group, " ") {
				if other := f.Lookup(name); other != nil && other.Changed {
					return true
				}
			}
		}
	}
	return false
}

func (f *FlagSet) VisitAll(fn func(*Flag)) {
	f.FlagSet.VisitAll(func(flag *pflag.Flag) {
		fn(&Flag{flag})
	})
}

func (fs FlagSet) LookupArg(arg string) (result *Flag) {
	isPosix := fs.IsPosix()
	fs.VisitAll(func(f *Flag) {
		if result != nil {
			return
		}

		if f.Matches(arg, isPosix) {
			result = f
		}
	})
	return
}
