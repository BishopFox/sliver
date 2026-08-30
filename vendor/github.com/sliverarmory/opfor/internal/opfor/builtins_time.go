package opfor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (r *Runtime) sleepTimeFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"formatDate": r.formatDate,
		"parseDate":  r.parseDate,
		"ticks":      r.ticks,
	}
}

func (r *Runtime) aggressorTimeFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"dstamp": r.dstamp,
		"tstamp": r.tstamp,
	}
}

const (
	// The public Aggressor reference defines only that dstamp includes seconds
	// and tstamp omits them. It does not publish the concrete layout. Keep the
	// independently specified OPFOR policy numeric, locale-independent, and
	// shared with the existing SimpleDateFormat bridge so a later licensed
	// differential can replace these constants without changing the clock or
	// coercion boundary.
	aggressorDStampPattern = "yyyy-MM-dd HH:mm:ss"
	aggressorTStampPattern = "yyyy-MM-dd HH:mm"
)

func (r *Runtime) ticks(_ context.Context, _ Invocation) (Value, error) {
	return Long(r.clock.Now().UnixMilli()), nil
}

// dstamp formats Unix milliseconds as a date/time value that includes
// seconds. The input instant is displayed in the configured clock's location,
// matching formatDate's explicit-timestamp form.
func (r *Runtime) dstamp(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorTimestamp(invocation, aggressorDStampPattern)
}

// tstamp formats Unix milliseconds as a date/time value without seconds.
func (r *Runtime) tstamp(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorTimestamp(invocation, aggressorTStampPattern)
}

func (r *Runtime) aggressorTimestamp(invocation Invocation, pattern string) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	instant := time.UnixMilli(invocation.Arg(0).Int64()).In(r.clock.Now().Location())
	formatted, err := formatJavaDate(instant, pattern)
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return String(formatted), nil
}

func (r *Runtime) formatDate(_ context.Context, invocation Invocation) (Value, error) {
	instant := r.clock.Now()
	patternIndex := 0
	if len(invocation.Arguments) == 2 {
		// TimeDateBridge reads the explicit timestamp through
		// BridgeUtilities.getLong. In particular, a Sleep StringValue uses
		// Long.parseLong semantics rather than OPFOR's broader host-facing
		// Value.Int64 conversion.
		instant = time.UnixMilli(sleepInt64(invocation.Arg(0))).In(instant.Location())
		patternIndex = 1
	}
	pattern, err := sleepBridgeArgument(invocation, patternIndex)
	if err != nil {
		return Null(), err
	}
	formatted, err := formatJavaDate(instant, pattern.String())
	if err != nil {
		var patternErr *javaDatePatternError
		if errors.As(err, &patternErr) {
			return Null(), sleepBridgeIllegalArgument(patternErr.Error())
		}
		return Null(), err
	}
	return String(formatted), nil
}

func (r *Runtime) parseDate(_ context.Context, invocation Invocation) (Value, error) {
	pattern, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	value, err := sleepBridgeArgument(invocation, 1)
	if err != nil {
		return Null(), err
	}
	instant, err := parseJavaDate(value.String(), pattern.String(), r.clock.Now())
	if err != nil {
		var patternErr *javaDatePatternError
		if errors.As(err, &patternErr) {
			return Null(), sleepBridgeIllegalArgument(patternErr.Error())
		}
		// SimpleDateFormat.parse(String, ParsePosition) returns null for a
		// nonmatching input. TimeDateBridge immediately dereferences that value,
		// and Sleep's Block translates the resulting NullPointerException.
		return Null(), sleepBridgeNullValue()
	}
	return Long(instant.UnixMilli()), nil
}

// javaDateLayout converts the SimpleDateFormat fields commonly used by Sleep
// scripts into Go's reference-time layout. Literal text and single-quote
// escaping follow SimpleDateFormat's quoting rules.
func javaDateLayout(pattern string) (string, error) {
	var layout strings.Builder
	quoted := false
	for index := 0; index < len(pattern); {
		if pattern[index] == '\'' {
			if index+1 < len(pattern) && pattern[index+1] == '\'' {
				layout.WriteByte('\'')
				index += 2
				continue
			}
			quoted = !quoted
			index++
			continue
		}
		if quoted || !asciiLetter(pattern[index]) {
			layout.WriteByte(pattern[index])
			index++
			continue
		}
		start := index
		for index < len(pattern) && pattern[index] == pattern[start] {
			index++
		}
		field, ok := javaDateField(pattern[start], index-start)
		if !ok {
			return "", fmt.Errorf("unsupported SimpleDateFormat field %q", pattern[start:index])
		}
		layout.WriteString(field)
	}
	if quoted {
		return "", fmt.Errorf("unterminated quoted literal")
	}
	return layout.String(), nil
}

func asciiLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func javaDateField(field byte, count int) (string, bool) {
	switch field {
	case 'y':
		if count == 2 {
			return "06", true
		}
		return "2006", true
	case 'M':
		switch {
		case count >= 4:
			return "January", true
		case count == 3:
			return "Jan", true
		case count == 2:
			return "01", true
		default:
			return "1", true
		}
	case 'd':
		if count >= 2 {
			return "02", true
		}
		return "2", true
	case 'H':
		return "15", true
	case 'h':
		if count >= 2 {
			return "03", true
		}
		return "3", true
	case 'm':
		if count >= 2 {
			return "04", true
		}
		return "4", true
	case 's':
		if count >= 2 {
			return "05", true
		}
		return "5", true
	case 'S':
		return strings.Repeat("0", min(count, 9)), true
	case 'E':
		if count >= 4 {
			return "Monday", true
		}
		return "Mon", true
	case 'a':
		return "PM", true
	case 'z':
		return "MST", true
	case 'Z':
		return "-0700", true
	case 'X':
		switch count {
		case 1:
			return "Z07", true
		case 2:
			return "Z0700", true
		default:
			return "Z07:00", true
		}
	default:
		return "", false
	}
}
