package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Duration represents an RFC 3339 duration (the ISO 8601 duration grammar
// referenced by the OpenAPI "duration" format), e.g. "P3Y6M4DT12H30M5S".
// See https://spec.openapis.org/registry/format/duration
//
// Calendar components (years, months, weeks, days) have no fixed length in
// wall-clock time, so the components are stored as parsed instead of being
// converted to a time.Duration; any spec-valid duration round-trips
// losslessly through ParseDuration and String. Use TimeDuration and
// FromTimeDuration to bridge to time.Duration where the conversion is
// well-defined.
//
// The RFC 3339 grammar makes weeks exclusive ("P2W" cannot be combined with
// other components); ParseDuration enforces that, but the struct cannot. A
// hand-built value mixing Weeks with other components serializes with the
// week between the months and days, which is outside the strict grammar.
type Duration struct {
	Years  int
	Months int
	Weeks  int
	Days   int

	Hours   int
	Minutes int
	// Seconds may carry a fraction (e.g. "PT0.5S"). Strictly, RFC 3339
	// allows only integer components; ISO 8601 permits a fraction on the
	// smallest component, and real-world payloads use it, so fractional
	// seconds are accepted when parsing and emitted when present.
	Seconds float64
}

// designator ranks enforce component order and uniqueness within each part
// of the grammar: Y then M then D in the date part, H then M then S in the
// time part.
var (
	dateDesignators = map[byte]int{'Y': 0, 'M': 1, 'D': 2}
	timeDesignators = map[byte]int{'H': 0, 'M': 1, 'S': 2}
)

// ParseDuration parses an RFC 3339 duration string per the grammar in RFC
// 3339 appendix A: "P" followed by ordered date components ("1Y2M3D"), a
// time part introduced by "T" ("T4H5M6S"), or a standalone week component
// ("2W"). At least one component must be present. As an interoperability
// extension beyond the strict grammar, the seconds component may carry a
// fraction, with either a period or a comma ("PT0.5S", "PT0,5S").
func ParseDuration(s string) (Duration, error) {
	fail := func(msg string) (Duration, error) {
		return Duration{}, fmt.Errorf("invalid RFC 3339 duration %q: %s", s, msg)
	}

	if !strings.HasPrefix(s, "P") {
		return fail("must start with 'P'")
	}
	rest := s[1:]
	if rest == "" {
		return fail("must contain at least one component")
	}

	var d Duration
	inTime := false
	lastRank := -1
	components := 0

	for len(rest) > 0 {
		if rest[0] == 'T' {
			if inTime {
				return fail("repeated 'T'")
			}
			inTime = true
			lastRank = -1
			rest = rest[1:]
			if rest == "" {
				return fail("'T' must be followed by a time component")
			}
			continue
		}

		// Scan the integer part of the number.
		intLen := 0
		for intLen < len(rest) && rest[intLen] >= '0' && rest[intLen] <= '9' {
			intLen++
		}
		if intLen == 0 {
			return fail(fmt.Sprintf("expected a number before %q", rest))
		}
		// Scan an optional fraction; only valid on the seconds component,
		// which is checked once the designator is known.
		numLen := intLen
		hasFraction := false
		if numLen < len(rest) && (rest[numLen] == '.' || rest[numLen] == ',') {
			hasFraction = true
			fracStart := numLen + 1
			numLen = fracStart
			for numLen < len(rest) && rest[numLen] >= '0' && rest[numLen] <= '9' {
				numLen++
			}
			if numLen == fracStart {
				return fail("fraction must contain digits")
			}
		}
		if numLen >= len(rest) {
			return fail("number must be followed by a designator")
		}
		number := rest[:numLen]
		designator := rest[numLen]
		rest = rest[numLen+1:]
		components++

		if designator == 'W' {
			if inTime {
				return fail("'W' is not valid in the time part")
			}
			if components != 1 || rest != "" {
				return fail("a week component cannot be combined with other components")
			}
			if hasFraction {
				return fail("only the seconds component may carry a fraction")
			}
			weeks, err := strconv.Atoi(number)
			if err != nil {
				return fail(fmt.Sprintf("invalid number %q: %v", number, err))
			}
			d.Weeks = weeks
			return d, nil
		}

		designators := dateDesignators
		if inTime {
			designators = timeDesignators
		}
		rank, ok := designators[designator]
		if !ok {
			return fail(fmt.Sprintf("unexpected designator %q", string(designator)))
		}
		if rank <= lastRank {
			return fail(fmt.Sprintf("component %q is out of order or repeated", string(designator)))
		}
		lastRank = rank

		if inTime && designator == 'S' {
			seconds, err := strconv.ParseFloat(strings.Replace(number, ",", ".", 1), 64)
			if err != nil {
				return fail(fmt.Sprintf("invalid number %q: %v", number, err))
			}
			d.Seconds = seconds
			continue
		}
		if hasFraction {
			return fail("only the seconds component may carry a fraction")
		}
		value, err := strconv.Atoi(number)
		if err != nil {
			return fail(fmt.Sprintf("invalid number %q: %v", number, err))
		}
		switch {
		case !inTime && designator == 'Y':
			d.Years = value
		case !inTime && designator == 'M':
			d.Months = value
		case !inTime && designator == 'D':
			d.Days = value
		case inTime && designator == 'H':
			d.Hours = value
		case inTime && designator == 'M':
			d.Minutes = value
		}
	}

	return d, nil
}

// String serializes the duration in RFC 3339 form, omitting zero components.
// The all-zero duration serializes as "PT0S".
func (d Duration) String() string {
	var b strings.Builder
	b.WriteByte('P')
	writeComponent := func(value int, designator byte) {
		if value != 0 {
			b.WriteString(strconv.Itoa(value))
			b.WriteByte(designator)
		}
	}
	writeComponent(d.Years, 'Y')
	writeComponent(d.Months, 'M')
	writeComponent(d.Weeks, 'W')
	writeComponent(d.Days, 'D')
	if d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0 {
		b.WriteByte('T')
		writeComponent(d.Hours, 'H')
		writeComponent(d.Minutes, 'M')
		if d.Seconds != 0 {
			b.WriteString(strconv.FormatFloat(d.Seconds, 'f', -1, 64))
			b.WriteByte('S')
		}
	}
	if b.Len() == 1 {
		return "PT0S"
	}
	return b.String()
}

// TimeDuration converts to a time.Duration. Years and months have no fixed
// length, so their presence is an error; weeks and days are converted with
// the common fixed convention of 7-day weeks and 24-hour days, which ignores
// calendar effects such as DST transitions.
func (d Duration) TimeDuration() (time.Duration, error) {
	if d.Years != 0 || d.Months != 0 {
		return 0, errors.New("duration contains years or months, which have no fixed length")
	}
	return time.Duration(d.Weeks)*7*24*time.Hour +
		time.Duration(d.Days)*24*time.Hour +
		time.Duration(d.Hours)*time.Hour +
		time.Duration(d.Minutes)*time.Minute +
		time.Duration(d.Seconds*float64(time.Second)), nil
}

// FromTimeDuration decomposes a time.Duration into hours, minutes and
// seconds. RFC 3339 durations cannot be negative, so a negative input is an
// error.
func FromTimeDuration(td time.Duration) (Duration, error) {
	if td < 0 {
		return Duration{}, errors.New("RFC 3339 durations cannot be negative")
	}
	var d Duration
	d.Hours = int(td / time.Hour)
	td -= time.Duration(d.Hours) * time.Hour
	d.Minutes = int(td / time.Minute)
	td -= time.Duration(d.Minutes) * time.Minute
	d.Seconds = td.Seconds()
	return d, nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseDuration(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(data []byte) error {
	parsed, err := ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Bind implements the runtime.Binder interface so that Duration is treated
// as a scalar value when binding parameters rather than being decomposed as
// a struct with key-value pairs.
func (d *Duration) Bind(src string) error {
	if src == "" {
		return nil
	}
	parsed, err := ParseDuration(src)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
