package sqlite3

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ncruces/julianday"
)

// TimeFormat specifies how to encode/decode time values.
//
// See the documentation for the [TimeFormatDefault] constant
// for formats recognized by SQLite.
//
// https://www.sqlite.org/lang_datefunc.html
type TimeFormat string

// TimeFormats recognized by SQLite to encode/decode time values.
//
// https://www.sqlite.org/lang_datefunc.html
const (
	TimeFormatDefault TimeFormat = "" // time.RFC3339Nano

	// Text formats
	TimeFormat1  TimeFormat = "2006-01-02"
	TimeFormat2  TimeFormat = "2006-01-02 15:04"
	TimeFormat3  TimeFormat = "2006-01-02 15:04:05"
	TimeFormat4  TimeFormat = "2006-01-02 15:04:05.000"
	TimeFormat5  TimeFormat = "2006-01-02T15:04"
	TimeFormat6  TimeFormat = "2006-01-02T15:04:05"
	TimeFormat7  TimeFormat = "2006-01-02T15:04:05.000"
	TimeFormat8  TimeFormat = "15:04"
	TimeFormat9  TimeFormat = "15:04:05"
	TimeFormat10 TimeFormat = "15:04:05.000"

	TimeFormat2TZ  = TimeFormat2 + "Z07:00"
	TimeFormat3TZ  = TimeFormat3 + "Z07:00"
	TimeFormat4TZ  = TimeFormat4 + "Z07:00"
	TimeFormat5TZ  = TimeFormat5 + "Z07:00"
	TimeFormat6TZ  = TimeFormat6 + "Z07:00"
	TimeFormat7TZ  = TimeFormat7 + "Z07:00"
	TimeFormat8TZ  = TimeFormat8 + "Z07:00"
	TimeFormat9TZ  = TimeFormat9 + "Z07:00"
	TimeFormat10TZ = TimeFormat10 + "Z07:00"

	// Numeric formats
	TimeFormatJulianDay TimeFormat = "julianday"
	TimeFormatUnix      TimeFormat = "unixepoch"
	TimeFormatUnixFrac  TimeFormat = "unixepoch_frac"
	TimeFormatUnixMilli TimeFormat = "unixepoch_milli" // not an SQLite format
	TimeFormatUnixMicro TimeFormat = "unixepoch_micro" // not an SQLite format
	TimeFormatUnixNano  TimeFormat = "unixepoch_nano"  // not an SQLite format

	// Auto
	TimeFormatAuto TimeFormat = "auto"
)

// Encode encodes a time value using this format.
//
// [TimeFormatDefault] and [TimeFormatAuto] encode using [time.RFC3339Nano],
// with nanosecond accuracy, and preserving any timezone offset.
//
// This is the format used by the [database/sql] driver:
// [database/sql.Row.Scan] will decode as [time.Time]
// values encoded with [time.RFC3339Nano].
//
// Time values encoded with [time.RFC3339Nano] cannot be sorted as strings
// to produce a time-ordered sequence.
//
// Assuming that the time zones of the time values are the same (e.g., all in UTC),
// and expressed using the same string (e.g., all "Z" or all "+00:00"),
// use the TIME [collating sequence] to produce a time-ordered sequence.
//
// Otherwise, use [TimeFormat7] for time-ordered encoding.
//
// Formats [TimeFormat1] through [TimeFormat10]
// convert time values to UTC before encoding.
//
// Returns a string for the text formats,
// a float64 for [TimeFormatJulianDay] and [TimeFormatUnixFrac],
// or an int64 for the other numeric formats.
//
// https://www.sqlite.org/lang_datefunc.html
//
// [collating sequence]: https://www.sqlite.org/datatype3.html#collating_sequences
func (f TimeFormat) Encode(t time.Time) any {
	switch f {
	// Numeric formats
	case TimeFormatJulianDay:
		return julianday.Float(t)
	case TimeFormatUnix:
		return t.Unix()
	case TimeFormatUnixFrac:
		return float64(t.Unix()) + float64(t.Nanosecond())*1e-9
	case TimeFormatUnixMilli:
		return t.UnixMilli()
	case TimeFormatUnixMicro:
		return t.UnixMicro()
	case TimeFormatUnixNano:
		return t.UnixNano()
	// Special formats
	case TimeFormatDefault, TimeFormatAuto:
		f = time.RFC3339Nano
	// SQLite assumes UTC if unspecified.
	case
		TimeFormat1, TimeFormat2,
		TimeFormat3, TimeFormat4,
		TimeFormat5, TimeFormat6,
		TimeFormat7, TimeFormat8,
		TimeFormat9, TimeFormat10:
		t = t.UTC()
	}
	return t.Format(string(f))
}

// Decode decodes a time value using this format.
//
// The time value can be a string, an int64, or a float64.
//
// Formats [TimeFormat8] through [TimeFormat10]
// (and [TimeFormat8TZ] through [TimeFormat10TZ])
// assume a date of 2000-01-01.
//
// The timezone indicator and fractional seconds are always optional
// for formats [TimeFormat2] through [TimeFormat10]
// (and [TimeFormat2TZ] through [TimeFormat10TZ]).
//
// [TimeFormatAuto] implements (and extends) the SQLite auto modifier.
// Julian day numbers are safe to use for historical dates,
// from 4712BC through 9999AD.
// Unix timestamps (expressed in seconds, milliseconds, microseconds, or nanoseconds)
// are safe to use for current events, from at least 1980 through at least 2260.
// Unix timestamps before 1980 and after 9999 may be misinterpreted as julian day numbers,
// or have the wrong time unit.
//
// https://www.sqlite.org/lang_datefunc.html
func (f TimeFormat) Decode(v any) (time.Time, error) {
	switch f {
	// Numeric formats
	case TimeFormatJulianDay:
		switch v := v.(type) {
		case string:
			return julianday.Parse(v)
		case float64:
			return julianday.FloatTime(v), nil
		case int64:
			return julianday.Time(v, 0), nil
		default:
			return time.Time{}, timeErr
		}

	case TimeFormatUnix, TimeFormatUnixFrac:
		if s, ok := v.(string); ok {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return time.Time{}, err
			}
			v = f
		}
		switch v := v.(type) {
		case float64:
			sec, frac := math.Modf(v)
			nsec := math.Floor(frac * 1e9)
			return time.Unix(int64(sec), int64(nsec)), nil
		case int64:
			return time.Unix(v, 0), nil
		default:
			return time.Time{}, timeErr
		}

	case TimeFormatUnixMilli:
		if s, ok := v.(string); ok {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			v = i
		}
		switch v := v.(type) {
		case float64:
			return time.UnixMilli(int64(math.Floor(v))), nil
		case int64:
			return time.UnixMilli(int64(v)), nil
		default:
			return time.Time{}, timeErr
		}

	case TimeFormatUnixMicro:
		if s, ok := v.(string); ok {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			v = i
		}
		switch v := v.(type) {
		case float64:
			return time.UnixMicro(int64(math.Floor(v))), nil
		case int64:
			return time.UnixMicro(int64(v)), nil
		default:
			return time.Time{}, timeErr
		}

	case TimeFormatUnixNano:
		if s, ok := v.(string); ok {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, timeErr
			}
			v = i
		}
		switch v := v.(type) {
		case float64:
			return time.Unix(0, int64(math.Floor(v))), nil
		case int64:
			return time.Unix(0, int64(v)), nil
		default:
			return time.Time{}, timeErr
		}

	// Special formats
	case TimeFormatAuto:
		switch s := v.(type) {
		case string:
			i, err := strconv.ParseInt(s, 10, 64)
			if err == nil {
				v = i
				break
			}
			f, err := strconv.ParseFloat(s, 64)
			if err == nil {
				v = f
				break
			}

			dates := []TimeFormat{
				TimeFormat6TZ, TimeFormat6, TimeFormat3TZ, TimeFormat3,
				TimeFormat5TZ, TimeFormat5, TimeFormat2TZ, TimeFormat2,
				TimeFormat1,
			}
			for _, f := range dates {
				t, err := time.Parse(string(f), s)
				if err == nil {
					return t, nil
				}
			}

			times := []TimeFormat{
				TimeFormat9TZ, TimeFormat9, TimeFormat8TZ, TimeFormat8,
			}
			for _, f := range times {
				t, err := time.Parse(string(f), s)
				if err == nil {
					return t.AddDate(2000, 0, 0), nil
				}
			}
		}
		switch v := v.(type) {
		case float64:
			if 0 <= v && v < 5373484.5 {
				return TimeFormatJulianDay.Decode(v)
			}
			if v < 253402300800 {
				return TimeFormatUnixFrac.Decode(v)
			}
			if v < 253402300800_000 {
				return TimeFormatUnixMilli.Decode(v)
			}
			if v < 253402300800_000000 {
				return TimeFormatUnixMicro.Decode(v)
			}
			return TimeFormatUnixNano.Decode(v)
		case int64:
			if 0 <= v && v < 5373485 {
				return TimeFormatJulianDay.Decode(v)
			}
			if v < 253402300800 {
				return TimeFormatUnixFrac.Decode(v)
			}
			if v < 253402300800_000 {
				return TimeFormatUnixMilli.Decode(v)
			}
			if v < 253402300800_000000 {
				return TimeFormatUnixMicro.Decode(v)
			}
			return TimeFormatUnixNano.Decode(v)
		default:
			return time.Time{}, timeErr
		}

	case
		TimeFormat2, TimeFormat2TZ,
		TimeFormat3, TimeFormat3TZ,
		TimeFormat4, TimeFormat4TZ,
		TimeFormat5, TimeFormat5TZ,
		TimeFormat6, TimeFormat6TZ,
		TimeFormat7, TimeFormat7TZ:
		s, ok := v.(string)
		if !ok {
			return time.Time{}, timeErr
		}
		return f.parseRelaxed(s)

	case
		TimeFormat8, TimeFormat8TZ,
		TimeFormat9, TimeFormat9TZ,
		TimeFormat10, TimeFormat10TZ:
		s, ok := v.(string)
		if !ok {
			return time.Time{}, timeErr
		}
		t, err := f.parseRelaxed(s)
		return t.AddDate(2000, 0, 0), err

	default:
		s, ok := v.(string)
		if !ok {
			return time.Time{}, timeErr
		}
		if f == "" {
			f = time.RFC3339Nano
		}
		return time.Parse(string(f), s)
	}
}

func (f TimeFormat) parseRelaxed(s string) (time.Time, error) {
	fs := string(f)
	fs = strings.TrimSuffix(fs, "Z07:00")
	fs = strings.TrimSuffix(fs, ".000")
	t, err := time.Parse(fs+"Z07:00", s)
	if err != nil {
		return time.Parse(fs, s)
	}
	return t, nil
}
