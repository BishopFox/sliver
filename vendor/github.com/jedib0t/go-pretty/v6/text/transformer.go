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
	// Pre-computed time conversion constants to avoid repeated calculations
	nanosPerSecond  = int64(time.Second)
	microsPerSecond = nanosPerSecond / 1000
	millisPerSecond = nanosPerSecond / 1000000

	// Thresholds for detecting unix timestamp units (10 seconds worth in each unit)
	unixTimeMinMilliseconds = 10 * nanosPerSecond
	unixTimeMinMicroseconds = 10 * nanosPerSecond * 1000
	unixTimeMinNanoSeconds  = 10 * nanosPerSecond * 1000000
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
//   - transforms the number as directed by 'format' (ex.: %.2f)
//   - colors negative values Red
//   - colors positive values Green
//
//gocyclo:ignore
func NewNumberTransformer(format string) Transformer {
	// Pre-compute negative format string to avoid repeated allocations
	negFormat := "-" + format

	transformInt64 := func(val int64) string {
		if val < 0 {
			return colorsNumberNegative.Sprintf(negFormat, -val)
		}
		if val > 0 {
			return colorsNumberPositive.Sprintf(format, val)
		}
		return colorsNumberZero.Sprintf(format, val)
	}

	transformUint64 := func(val uint64) string {
		if val > 0 {
			return colorsNumberPositive.Sprintf(format, val)
		}
		return colorsNumberZero.Sprintf(format, val)
	}

	transformFloat64 := func(val float64) string {
		if val < 0 {
			return colorsNumberNegative.Sprintf(negFormat, -val)
		}
		if val > 0 {
			return colorsNumberPositive.Sprintf(format, val)
		}
		return colorsNumberZero.Sprintf(format, val)
	}

	// Use type switch for O(1) type checking instead of sequential type assertions
	return func(val interface{}) string {
		switch v := val.(type) {
		case int:
			return transformInt64(int64(v))
		case int8:
			return transformInt64(int64(v))
		case int16:
			return transformInt64(int64(v))
		case int32:
			return transformInt64(int64(v))
		case int64:
			return transformInt64(v)
		case uint:
			return transformUint64(uint64(v))
		case uint8:
			return transformUint64(uint64(v))
		case uint16:
			return transformUint64(uint64(v))
		case uint32:
			return transformUint64(uint64(v))
		case uint64:
			return transformUint64(v)
		case float32:
			return transformFloat64(float64(v))
		case float64:
			return transformFloat64(v)
		default:
			return fmt.Sprint(val)
		}
	}
}

// NewJSONTransformer returns a Transformer that can format a JSON string or an
// object into pretty-indented JSON-strings.
func NewJSONTransformer(prefix string, indent string) Transformer {
	return func(val interface{}) string {
		if valStr, ok := val.(string); ok {
			valStr = strings.TrimSpace(valStr)
			// Validate JSON before attempting to indent to avoid unnecessary processing
			if !json.Valid([]byte(valStr)) {
				return fmt.Sprintf("%#v", valStr)
			}
			var b bytes.Buffer
			if err := json.Indent(&b, []byte(valStr), prefix, indent); err == nil {
				return b.String()
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
		// Check for time.Time first to avoid unnecessary fmt.Sprint conversion
		if valTime, ok := val.(time.Time); ok {
			return formatTime(valTime, layout, location)
		}
		// Only convert to string if not already time.Time
		rsp := fmt.Sprint(val)
		// Cycle through some supported layouts to see if the string form
		// of the object matches any of these layouts
		for _, possibleTimeLayout := range possibleTimeLayouts {
			if valTime, err := time.Parse(possibleTimeLayout, rsp); err == nil {
				return formatTime(valTime, layout, location)
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
	transformer := NewTimeTransformer(layout, location)

	return func(val interface{}) string {
		if unixTime, ok := val.(int64); ok {
			return formatTimeUnix(unixTime, transformer)
		} else if unixTimeStr, ok := val.(string); ok {
			if unixTime, err := strconv.ParseInt(unixTimeStr, 10, 64); err == nil {
				return formatTimeUnix(unixTime, transformer)
			}
		}
		return fmt.Sprint(val)
	}
}

// NewURLTransformer returns a Transformer that can format and pretty print a string
// that contains a URL (the text is underlined and colored Blue).
func NewURLTransformer(colors ...Color) Transformer {
	colorsToUse := colorsURL
	if len(colors) > 0 {
		colorsToUse = colors
	}

	return func(val interface{}) string {
		return colorsToUse.Sprint(val)
	}
}

func formatTime(t time.Time, layout string, location *time.Location) string {
	rsp := ""
	if t.Unix() > 0 {
		if location != nil {
			t = t.In(location)
		}
		rsp = t.Format(layout)
	}
	return rsp
}

func formatTimeUnix(unixTime int64, timeTransformer Transformer) string {
	// Use pre-computed constants instead of repeated time.Second.Nanoseconds() calls
	if unixTime >= unixTimeMinNanoSeconds {
		unixTime = unixTime / nanosPerSecond
	} else if unixTime >= unixTimeMinMicroseconds {
		unixTime = unixTime / microsPerSecond
	} else if unixTime >= unixTimeMinMilliseconds {
		unixTime = unixTime / millisPerSecond
	}
	return timeTransformer(time.Unix(unixTime, 0))
}
