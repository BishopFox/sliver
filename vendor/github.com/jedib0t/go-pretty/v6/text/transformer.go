package text

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Transformer related constants
const (
	unixTimeMinMilliseconds = int64(10000000000)
	unixTimeMinMicroseconds = unixTimeMinMilliseconds * 1000
	unixTimeMinNanoSeconds  = unixTimeMinMicroseconds * 1000
)

// Transformer related variables
var (
	colorsNumberPositive = Colors{FgHiGreen}
	colorsNumberNegative = Colors{FgHiRed}
	colorsNumberZero     = Colors{}
	colorsURL            = Colors{Underline, FgBlue}
	rfc3339Milli         = "2006-01-02T15:04:05.000Z07:00"
	rfc3339Micro         = "2006-01-02T15:04:05.000000Z07:00"

	possibleTimeLayouts = []string{
		time.RFC3339,
		rfc3339Milli, // strfmt.DateTime.String()'s default layout
		rfc3339Micro,
		time.RFC3339Nano,
	}
)

// Transformer helps format the contents of an object to the user's liking.
type Transformer func(val interface{}) string

// NewNumberTransformer returns a number Transformer that:
//   * transforms the number as directed by 'format' (ex.: %.2f)
//   * colors negative values Red
//   * colors positive values Green
func NewNumberTransformer(format string) Transformer {
	return func(val interface{}) string {
		if number, ok := val.(int); ok {
			return transformInt(format, int64(number))
		}
		if number, ok := val.(int8); ok {
			return transformInt(format, int64(number))
		}
		if number, ok := val.(int16); ok {
			return transformInt(format, int64(number))
		}
		if number, ok := val.(int32); ok {
			return transformInt(format, int64(number))
		}
		if number, ok := val.(int64); ok {
			return transformInt(format, int64(number))
		}
		if number, ok := val.(uint); ok {
			return transformUint(format, uint64(number))
		}
		if number, ok := val.(uint8); ok {
			return transformUint(format, uint64(number))
		}
		if number, ok := val.(uint16); ok {
			return transformUint(format, uint64(number))
		}
		if number, ok := val.(uint32); ok {
			return transformUint(format, uint64(number))
		}
		if number, ok := val.(uint64); ok {
			return transformUint(format, uint64(number))
		}
		if number, ok := val.(float32); ok {
			return transformFloat(format, float64(number))
		}
		if number, ok := val.(float64); ok {
			return transformFloat(format, float64(number))
		}
		return fmt.Sprint(val)
	}
}

func transformInt(format string, val int64) string {
	if val < 0 {
		return colorsNumberNegative.Sprintf("-"+format, -val)
	}
	if val > 0 {
		return colorsNumberPositive.Sprintf(format, val)
	}
	return colorsNumberZero.Sprintf(format, val)
}

func transformUint(format string, val uint64) string {
	if val > 0 {
		return colorsNumberPositive.Sprintf(format, val)
	}
	return colorsNumberZero.Sprintf(format, val)
}

func transformFloat(format string, val float64) string {
	if val < 0 {
		return colorsNumberNegative.Sprintf("-"+format, -val)
	}
	if val > 0 {
		return colorsNumberPositive.Sprintf(format, val)
	}
	return colorsNumberZero.Sprintf(format, val)
}

// NewJSONTransformer returns a Transformer that can format a JSON string or an
// object into pretty-indented JSON-strings.
func NewJSONTransformer(prefix string, indent string) Transformer {
	return func(val interface{}) string {
		if valStr, ok := val.(string); ok {
			var b bytes.Buffer
			if err := json.Indent(&b, []byte(strings.TrimSpace(valStr)), prefix, indent); err == nil {
				return string(b.Bytes())
			}
		} else if b, err := json.MarshalIndent(val, prefix, indent); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%#v", val)
	}
}

// NewTimeTransformer returns a Transformer that can format a timestamp (a
// time.Time) into a well-defined time format defined using the provided layout
// (ex.: time.RFC3339).
//
// If a non-nil location value is provided, the time will be localized to that
// location (use time.Local to get localized timestamps).
func NewTimeTransformer(layout string, location *time.Location) Transformer {
	return func(val interface{}) string {
		formatTime := func(t time.Time) string {
			rsp := ""
			if t.Unix() > 0 {
				if location != nil {
					t = t.In(location)
				}
				rsp = t.Format(layout)
			}
			return rsp
		}

		rsp := fmt.Sprint(val)
		if valTime, ok := val.(time.Time); ok {
			rsp = formatTime(valTime)
		} else {
			// cycle through some supported layouts to see if the string form
			// of the object matches any of these layouts
			for _, possibleTimeLayout := range possibleTimeLayouts {
				if valTime, err := time.Parse(possibleTimeLayout, rsp); err == nil {
					rsp = formatTime(valTime)
					break
				}
			}
		}
		return rsp
	}
}

// NewUnixTimeTransformer returns a Transformer that can format a unix-timestamp
// into a well-defined time format as defined by 'layout'. This can handle
// unix-time in Seconds, MilliSeconds, Microseconds and Nanoseconds.
//
// If a non-nil location value is provided, the time will be localized to that
// location (use time.Local to get localized timestamps).
func NewUnixTimeTransformer(layout string, location *time.Location) Transformer {
	timeTransformer := NewTimeTransformer(layout, location)
	formatUnixTime := func(unixTime int64) string {
		if unixTime >= unixTimeMinNanoSeconds {
			unixTime = unixTime / time.Second.Nanoseconds()
		} else if unixTime >= unixTimeMinMicroseconds {
			unixTime = unixTime / (time.Second.Nanoseconds() / 1000)
		} else if unixTime >= unixTimeMinMilliseconds {
			unixTime = unixTime / (time.Second.Nanoseconds() / 1000000)
		}
		return timeTransformer(time.Unix(unixTime, 0))
	}

	return func(val interface{}) string {
		if unixTime, ok := val.(int64); ok {
			return formatUnixTime(unixTime)
		} else if unixTimeStr, ok := val.(string); ok {
			if unixTime, err := strconv.ParseInt(unixTimeStr, 10, 64); err == nil {
				return formatUnixTime(unixTime)
			}
		}
		return fmt.Sprint(val)
	}
}

// NewURLTransformer returns a Transformer that can format and pretty print a string
// that contains an URL (the text is underlined and colored Blue).
func NewURLTransformer() Transformer {
	return func(val interface{}) string {
		return colorsURL.Sprint(val)
	}
}
