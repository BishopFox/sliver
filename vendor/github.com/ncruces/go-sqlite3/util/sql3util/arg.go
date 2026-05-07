package sql3util

import (
	"strconv"
	"strings"
	"time"
)

// NamedArg splits an named arg into a key and value,
// around an equals sign.
// Spaces are trimmed around both key and value.
func NamedArg(arg string) (key, val string) {
	key, val, _ = strings.Cut(arg, "=")
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	return
}

// Unquote unquotes a string.
//
// https://sqlite.org/lang_keywords.html
func Unquote(val string) string {
	if len(val) < 2 {
		return val
	}
	fst := val[0]
	lst := val[len(val)-1]
	rst := val[1 : len(val)-1]
	if fst == '[' && lst == ']' {
		return rst
	}
	if fst != lst {
		return val
	}
	var old, new string
	switch fst {
	default:
		return val
	case '`':
		old, new = "``", "`"
	case '"':
		old, new = `""`, `"`
	case '\'':
		old, new = `''`, `'`
	}
	return strings.ReplaceAll(rst, old, new)
}

// ParseBool parses a boolean.
//
// https://sqlite.org/pragma.html#syntax
func ParseBool(s string) (b, ok bool) {
	if len(s) == 0 {
		return false, false
	}
	if s[0] == '0' {
		return false, true
	}
	if '1' <= s[0] && s[0] <= '9' {
		return true, true
	}
	switch strings.ToLower(s) {
	case "true", "yes", "on":
		return true, true
	case "false", "no", "off":
		return false, true
	}
	return false, false
}

// ParseFloat parses a decimal floating point number.
func ParseFloat(s string) (f float64, ok bool) {
	if strings.TrimLeft(s, "+-.0123456789Ee") != "" {
		return
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// ParseTimeShift parses a time shift modifier,
// also the output of timediff.
//
// https://sqlite.org/lang_datefunc.html
func ParseTimeShift(s string) (years, months, days int, duration time.Duration, ok bool) {
	// Sign part: ±
	neg := strings.HasPrefix(s, "-")
	sign := neg || strings.HasPrefix(s, "+")
	if sign {
		s = s[1:]
	}

	if ok = len(s) >= 5; !ok {
		return // !ok
	}

	defer func() {
		if neg {
			years = -years
			months = -months
			days = -days
			duration = -duration
		}
	}()

	// Date part: YYYY-MM-DD
	if s[4] == '-' {
		if ok = sign && len(s) >= 10 && s[7] == '-'; !ok {
			return // !ok
		}
		if years, ok = parseInt(s[0:4], 0); !ok {
			return // !ok
		}
		if months, ok = parseInt(s[5:7], 12); !ok {
			return // !ok
		}
		if days, ok = parseInt(s[8:10], 31); !ok {
			return // !ok
		}
		if len(s) == 10 {
			return
		}
		if ok = s[10] == ' '; !ok {
			return // !ok
		}
		s = s[11:]
	}

	// Time part: HH:MM
	if ok = len(s) >= 5 && s[2] == ':'; !ok {
		return // !ok
	}

	var hours, minutes int
	if hours, ok = parseInt(s[0:2], 24); !ok {
		return
	}
	if minutes, ok = parseInt(s[3:5], 60); !ok {
		return
	}
	duration = time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute

	if len(s) == 5 {
		return
	}
	if ok = len(s) >= 8 && s[5] == ':'; !ok {
		return // !ok
	}

	// Seconds part: HH:MM:SS
	var seconds int
	if seconds, ok = parseInt(s[6:8], 60); !ok {
		return
	}
	duration += time.Duration(seconds) * time.Second

	if len(s) == 8 {
		return
	}
	if ok = len(s) >= 10 && s[8] == '.'; !ok {
		return // !ok
	}
	s = s[9:]

	// Nanosecond part: HH:MM:SS.SSS
	var nanos int
	if nanos, ok = parseInt(s[0:min(9, len(s))], 0); !ok {
		return
	}
	for i := len(s); i < 9; i++ {
		nanos *= 10
	}
	duration += time.Duration(nanos)

	// Subnanosecond part.
	if len(s) > 9 {
		_, ok = parseInt(s[9:], 0)
	}
	return
}

func parseInt(s string, max int) (i int, _ bool) {
	for _, r := range []byte(s) {
		r -= '0'
		if r > 9 {
			return
		}
		i = i*10 + int(r)
	}
	return i, max == 0 || i < max
}
