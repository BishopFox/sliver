/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2018 Roland Singer [roland.singer@deserbit.com]
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package grumble

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type parseFlagFunc func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error)
type defaultFlagFunc func(res FlagMap)

type flagItem struct {
	Short           string
	Long            string
	Help            string
	HelpArgs        string
	HelpShowDefault bool
	Default         interface{}
}

// Flags holds all the registered flags.
type Flags struct {
	parsers  []parseFlagFunc
	defaults map[string]defaultFlagFunc
	list     []*flagItem
}

// empty returns true, if the flags are empty.
func (f *Flags) empty() bool {
	return len(f.list) == 0
}

// sort the flags by their name.
func (f *Flags) sort() {
	sort.Slice(f.list, func(i, j int) bool {
		return f.list[i].Long < f.list[j].Long
	})
}

func (f *Flags) register(
	short, long, help, helpArgs string,
	helpShowDefault bool,
	defaultValue interface{},
	df defaultFlagFunc,
	pf parseFlagFunc,
) {
	// Validate.
	if len(short) > 1 {
		panic(fmt.Errorf("invalid short flag: '%s': must be a single character", short))
	} else if strings.HasPrefix(short, "-") {
		panic(fmt.Errorf("invalid short flag: '%s': must not start with a '-'", short))
	} else if len(long) == 0 {
		panic(fmt.Errorf("empty long flag: short='%s'", short))
	} else if strings.HasPrefix(long, "-") {
		panic(fmt.Errorf("invalid long flag: '%s': must not start with a '-'", long))
	} else if len(help) == 0 {
		panic(fmt.Errorf("empty flag help message for flag: '%s'", long))
	}

	// Check, that both short and long are unique.
	// Short flags are empty if not set.
	for _, fi := range f.list {
		if fi.Short != "" && short != "" && fi.Short == short {
			panic(fmt.Errorf("flag shortcut '%s' registered twice", short))
		}
		if fi.Long == long {
			panic(fmt.Errorf("flag '%s' registered twice", long))
		}
	}

	f.list = append(f.list, &flagItem{
		Short:           short,
		Long:            long,
		Help:            help,
		HelpShowDefault: helpShowDefault,
		HelpArgs:        helpArgs,
		Default:         defaultValue,
	})

	if f.defaults == nil {
		f.defaults = make(map[string]defaultFlagFunc)
	}
	f.defaults[long] = df

	f.parsers = append(f.parsers, pf)
}

func (f *Flags) match(flag, short, long string) bool {
	return (len(short) > 0 && flag == "-"+short) ||
		(len(long) > 0 && flag == "--"+long)
}

func (f *Flags) parse(args []string, res FlagMap) ([]string, error) {
	var err error
	var parsed bool

	// Parse all leading flags.
Loop:
	for len(args) > 0 {
		a := args[0]
		if !strings.HasPrefix(a, "-") {
			break Loop
		}
		args = args[1:]

		// A double dash (--) is used to signify the end of command options,
		// after which only positional arguments are accepted.
		if a == "--" {
			break Loop
		}

		pos := strings.Index(a, "=")
		equalVal := ""
		if pos > 0 {
			equalVal = a[pos+1:]
			a = a[:pos]
		}

		for _, p := range f.parsers {
			args, parsed, err = p(a, equalVal, args, res)
			if err != nil {
				return nil, err
			} else if parsed {
				continue Loop
			}
		}
		return nil, fmt.Errorf("invalid flag: %s", a)
	}

	// Finally set all the default values for not passed flags.
	if f.defaults == nil {
		return args, nil
	}

	for _, i := range f.list {
		if _, ok := res[i.Long]; ok {
			continue
		}
		df, ok := f.defaults[i.Long]
		if !ok {
			return nil, fmt.Errorf("invalid flag: missing default function: %s", i.Long)
		}
		df(res)
	}

	return args, nil
}

// StringL same as String, but without a shorthand.
func (f *Flags) StringL(long, defaultValue, help string) {
	f.String("", long, defaultValue, help)
}

// String registers a string flag.
func (f *Flags) String(short, long, defaultValue, help string) {
	f.register(short, long, help, "string", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			if len(equalVal) > 0 {
				res[long] = &FlagMapItem{
					Value:     trimQuotes(equalVal),
					IsDefault: false,
				}
				return args, true, nil
			}
			if len(args) == 0 {
				return args, false, fmt.Errorf("missing string value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     args[0],
				IsDefault: false,
			}
			args = args[1:]
			return args, true, nil
		})
}

// BoolL same as Bool, but without a shorthand.
func (f *Flags) BoolL(long string, defaultValue bool, help string) {
	f.Bool("", long, defaultValue, help)
}

// Bool registers a boolean flag.
func (f *Flags) Bool(short, long string, defaultValue bool, help string) {
	f.register(short, long, help, "", false, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			if len(equalVal) > 0 {
				b, err := strconv.ParseBool(equalVal)
				if err != nil {
					return args, false, fmt.Errorf("invalid boolean value for flag: %s", flag)
				}
				res[long] = &FlagMapItem{
					Value:     b,
					IsDefault: false,
				}
				return args, true, nil
			}
			res[long] = &FlagMapItem{
				Value:     true,
				IsDefault: false,
			}
			return args, true, nil
		})
}

// IntL same as Int, but without a shorthand.
func (f *Flags) IntL(long string, defaultValue int, help string) {
	f.Int("", long, defaultValue, help)
}

// Int registers an int flag.
func (f *Flags) Int(short, long string, defaultValue int, help string) {
	f.register(short, long, help, "int", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing int value for flag: %s", flag)
			}
			i, err := strconv.Atoi(vStr)
			if err != nil {
				return args, false, fmt.Errorf("invalid int value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     i,
				IsDefault: false,
			}
			return args, true, nil
		})
}

// Int64L same as Int64, but without a shorthand.
func (f *Flags) Int64L(long string, defaultValue int64, help string) {
	f.Int64("", long, defaultValue, help)
}

// Int64 registers an int64 flag.
func (f *Flags) Int64(short, long string, defaultValue int64, help string) {
	f.register(short, long, help, "int", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing int value for flag: %s", flag)
			}
			i, err := strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return args, false, fmt.Errorf("invalid int value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     i,
				IsDefault: false,
			}
			return args, true, nil
		})
}

// UintL same as Uint, but without a shorthand.
func (f *Flags) UintL(long string, defaultValue uint, help string) {
	f.Uint("", long, defaultValue, help)
}

// Uint registers an uint flag.
func (f *Flags) Uint(short, long string, defaultValue uint, help string) {
	f.register(short, long, help, "uint", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing uint value for flag: %s", flag)
			}
			i, err := strconv.ParseUint(vStr, 10, 64)
			if err != nil {
				return args, false, fmt.Errorf("invalid uint value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     uint(i),
				IsDefault: false,
			}
			return args, true, nil
		})
}

// Uint64L same as Uint64, but without a shorthand.
func (f *Flags) Uint64L(long string, defaultValue uint64, help string) {
	f.Uint64("", long, defaultValue, help)
}

// Uint64 registers an uint64 flag.
func (f *Flags) Uint64(short, long string, defaultValue uint64, help string) {
	f.register(short, long, help, "uint", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing uint value for flag: %s", flag)
			}
			i, err := strconv.ParseUint(vStr, 10, 64)
			if err != nil {
				return args, false, fmt.Errorf("invalid uint value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     i,
				IsDefault: false,
			}
			return args, true, nil
		})
}

// Float64L same as Float64, but without a shorthand.
func (f *Flags) Float64L(long string, defaultValue float64, help string) {
	f.Float64("", long, defaultValue, help)
}

// Float64 registers an float64 flag.
func (f *Flags) Float64(short, long string, defaultValue float64, help string) {
	f.register(short, long, help, "float", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing float value for flag: %s", flag)
			}
			i, err := strconv.ParseFloat(vStr, 64)
			if err != nil {
				return args, false, fmt.Errorf("invalid float value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     i,
				IsDefault: false,
			}
			return args, true, nil
		})
}

// DurationL same as Duration, but without a shorthand.
func (f *Flags) DurationL(long string, defaultValue time.Duration, help string) {
	f.Duration("", long, defaultValue, help)
}

// Duration registers a duration flag.
func (f *Flags) Duration(short, long string, defaultValue time.Duration, help string) {
	f.register(short, long, help, "duration", true, defaultValue,
		func(res FlagMap) {
			res[long] = &FlagMapItem{
				Value:     defaultValue,
				IsDefault: true,
			}
		},
		func(flag, equalVal string, args []string, res FlagMap) ([]string, bool, error) {
			if !f.match(flag, short, long) {
				return args, false, nil
			}
			var vStr string
			if len(equalVal) > 0 {
				vStr = equalVal
			} else if len(args) > 0 {
				vStr = args[0]
				args = args[1:]
			} else {
				return args, false, fmt.Errorf("missing duration value for flag: %s", flag)
			}
			d, err := time.ParseDuration(vStr)
			if err != nil {
				return args, false, fmt.Errorf("invalid duration value for flag: %s", flag)
			}
			res[long] = &FlagMapItem{
				Value:     d,
				IsDefault: false,
			}
			return args, true, nil
		})
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
