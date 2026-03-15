// Package v1 provides MessageFormat v1 (ICU MessageFormat) implementation for Go
package v1

import (
	"fmt"
	"strings"
)

// DateToken represents an object representation of a parsed date skeleton token
// TypeScript original code:
// export type DateToken =
//
//	| string
//	| { char: string; width: number }
//	| { error: string };
type DateToken interface {
	isDateToken()
}

// DateTokenString represents a string token (literal text)
// TypeScript original code: string (in DateToken union)
type DateTokenString struct {
	Value string
}

func (d *DateTokenString) isDateToken()   {}
func (d *DateTokenString) String() string { return d.Value }

// DateTokenField represents a date field token with character and width
// TypeScript original code: { char: string; width: number }
type DateTokenField struct {
	Char  string
	Width int
}

func (d *DateTokenField) isDateToken() {}
func (d *DateTokenField) String() string {
	return fmt.Sprintf("{char: %s, width: %d}", d.Char, d.Width)
}

// DateTokenError represents an error token
// TypeScript original code: { error: string }
type DateTokenError struct {
	Error string
}

func (d *DateTokenError) isDateToken() {}
func (d *DateTokenError) String() string {
	return fmt.Sprintf("{error: %s}", d.Error)
}

// DateFormatError represents errors during date formatting
// TypeScript original code:
//
//	export class DateFormatError extends Error {
//	  type: 'invalid' | 'duplicate' | 'unsupported' | 'literal';
//	  token: DateToken;
//	  constructor(type: DateFormatError['type'], msg: string, token: DateToken)
//	}
type DateFormatError struct {
	Type    string // "invalid" | "duplicate" | "unsupported" | "literal"
	Message string
	Token   DateToken
}

func (e *DateFormatError) Error() string {
	return fmt.Sprintf("DateFormat %s: %s", e.Type, e.Message)
}

// NewDateFormatError creates a new DateFormatError
// TypeScript original code:
//
//	constructor(type: DateFormatError['type'], msg: string, token: DateToken) {
//	  super(msg);
//	  this.type = type || 'error';
//	  this.token = token;
//	}
func NewDateFormatError(errorType, message string, token DateToken) *DateFormatError {
	return &DateFormatError{
		Type:    errorType,
		Message: message,
		Token:   token,
	}
}

// isLetter checks if a character is a letter (A-Z or a-z)
// TypeScript original code:
// const isLetter = (char: string) =>
//
//	(char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z');
func isLetter(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

// ParseDateTokens parses an ICU DateFormat skeleton string into a DateToken array
// TypeScript original code:
//
//	export function parseDateTokens(src: string) {
//	  const tokens: DateToken[] = [];
//	  let pos = 0;
//	  while (true) {
//	    const char = src[pos];
//	    if (!char) break;
//	    let token: DateToken;
//	    if (isLetter(char)) {
//	      let width = 1;
//	      while (src[++pos] === char) ++width;
//	      token = { char, width };
//	    } else if (char === "'") {
//	      // handle quoted literals...
//	    } else {
//	      // handle other characters...
//	    }
//	    tokens.push(token);
//	  }
//	  return tokens;
//	}
func ParseDateTokens(src string) []DateToken {
	var tokens []DateToken
	runes := []rune(src)
	pos := 0

	for pos < len(runes) {
		ch := runes[pos]

		var token DateToken
		switch {
		case isLetter(ch):
			char := string(ch)
			width := 1
			pos++

			for pos < len(runes) && runes[pos] == ch {
				width++
				pos++
			}

			token = &DateTokenField{
				Char:  char,
				Width: width,
			}
		case ch == '\'':
			pos++

			if pos < len(runes) && runes[pos] == '\'' {
				token = &DateTokenString{Value: "'"}
				pos++
			} else {
				var str strings.Builder

				for pos < len(runes) {
					next := runes[pos]
					if next == '\'' {
						pos++
						if pos < len(runes) && runes[pos] == '\'' {
							str.WriteRune('\'')
							pos++
						} else {
							break
						}
					} else {
						str.WriteRune(next)
						pos++
					}
				}

				if pos > len(runes) || (pos == len(runes) && src[len(src)-1] != '\'') {
					token = &DateTokenError{
						Error: fmt.Sprintf("Unterminated quoted literal in pattern: %s", src),
					}
				} else {
					token = &DateTokenString{Value: str.String()}
				}
			}
		default:
			var str strings.Builder
			str.WriteRune(ch)
			pos++

			for pos < len(runes) && !isLetter(runes[pos]) && runes[pos] != '\'' {
				str.WriteRune(runes[pos])
				pos++
			}

			token = &DateTokenString{Value: str.String()}
		}

		tokens = append(tokens, token)
	}

	return tokens
}

// DateTimeFormatOptions represents Intl.DateTimeFormat options
// This is a simplified version of the JavaScript Intl.DateTimeFormat options
type DateTimeFormatOptions struct {
	Era              string `json:"era,omitempty"`              // "narrow" | "short" | "long"
	Year             string `json:"year,omitempty"`             // "numeric" | "2-digit"
	Month            string `json:"month,omitempty"`            // "numeric" | "2-digit" | "narrow" | "short" | "long"
	Day              string `json:"day,omitempty"`              // "numeric" | "2-digit"
	Weekday          string `json:"weekday,omitempty"`          // "narrow" | "short" | "long"
	Hour             string `json:"hour,omitempty"`             // "numeric" | "2-digit"
	Minute           string `json:"minute,omitempty"`           // "numeric" | "2-digit"
	Second           string `json:"second,omitempty"`           // "numeric" | "2-digit"
	TimeZoneName     string `json:"timeZoneName,omitempty"`     // "short" | "long"
	HourCycle        string `json:"hourCycle,omitempty"`        // "h11" | "h12" | "h23" | "h24"
	Calendar         string `json:"calendar,omitempty"`         // Calendar type
	NumberingSystem  string `json:"numberingSystem,omitempty"`  // Numbering system
	TimeZone         string `json:"timeZone,omitempty"`         // Time zone
	FractionalSecond string `json:"fractionalSecond,omitempty"` // "1" | "2" | "3"
}

// GetDateTimeFormatOptions converts DateTokens to Intl.DateTimeFormat options
// TypeScript original code from options.ts (simplified version)
// This function maps ICU date skeleton patterns to Intl.DateTimeFormat options
func GetDateTimeFormatOptions(tokens []DateToken, onError func(errorType, message string, token DateToken)) *DateTimeFormatOptions {
	options := &DateTimeFormatOptions{}

	for _, token := range tokens {
		switch t := token.(type) {
		case *DateTokenField:
			switch t.Char {
			// Era
			case "G":
				switch t.Width {
				case 1, 2, 3:
					options.Era = "short"
				case 4:
					options.Era = "long"
				case 5:
					options.Era = "narrow"
				default:
					if onError != nil {
						onError("invalid", fmt.Sprintf("Invalid era width: %d", t.Width), token)
					}
				}

			// Year
			case "y":
				if t.Width == 2 {
					options.Year = "2-digit"
				} else {
					options.Year = "numeric"
				}

			// Month
			case "M", "L":
				switch t.Width {
				case 1:
					options.Month = "numeric"
				case 2:
					options.Month = "2-digit"
				case 3:
					options.Month = "short"
				case 4:
					options.Month = "long"
				case 5:
					options.Month = "narrow"
				default:
					if onError != nil {
						onError("invalid", fmt.Sprintf("Invalid month width: %d", t.Width), token)
					}
				}

			// Day
			case "d":
				if t.Width == 1 {
					options.Day = "numeric"
				} else {
					options.Day = "2-digit"
				}

			// Weekday
			case "E", "e", "c":
				switch t.Width {
				case 1, 2, 3:
					options.Weekday = "short"
				case 4:
					options.Weekday = "long"
				case 5, 6:
					options.Weekday = "narrow"
				default:
					if onError != nil {
						onError("invalid", fmt.Sprintf("Invalid weekday width: %d", t.Width), token)
					}
				}

			// Hour
			case "h", "H", "k", "K":
				if t.Width == 1 {
					options.Hour = "numeric"
				} else {
					options.Hour = "2-digit"
				}

				// Set hour cycle based on pattern
				switch t.Char {
				case "h":
					options.HourCycle = "h12" // 1-12
				case "H":
					options.HourCycle = "h23" // 0-23
				case "k":
					options.HourCycle = "h24" // 1-24
				case "K":
					options.HourCycle = "h11" // 0-11
				}

			// Minute
			case "m":
				if t.Width == 1 {
					options.Minute = "numeric"
				} else {
					options.Minute = "2-digit"
				}

			// Second
			case "s":
				if t.Width == 1 {
					options.Second = "numeric"
				} else {
					options.Second = "2-digit"
				}

			// Fractional seconds
			case "S":
				if t.Width >= 1 && t.Width <= 3 {
					options.FractionalSecond = fmt.Sprintf("%d", t.Width)
				} else if onError != nil {
					onError("invalid", fmt.Sprintf("Invalid fractional second width: %d", t.Width), token)
				}

			// Time zone
			case "z":
				switch t.Width {
				case 1, 2, 3:
					options.TimeZoneName = "short"
				case 4:
					options.TimeZoneName = "long"
				default:
					if onError != nil {
						onError("invalid", fmt.Sprintf("Invalid timezone width: %d", t.Width), token)
					}
				}

			// Calendar
			case "u":
				options.Calendar = "gregory" // Default calendar
				if onError != nil {
					onError("unsupported", fmt.Sprintf("Calendar field %s not fully supported", t.Char), token)
				}

			default:
				if onError != nil {
					onError("unsupported", fmt.Sprintf("Unsupported date field: %s", t.Char), token)
				}
			}

		case *DateTokenString:
			// Literal strings are ignored in options conversion
			// They would be handled differently in actual formatting

		case *DateTokenError:
			if onError != nil {
				onError("invalid", t.Error, token)
			}
		}
	}

	return options
}

// GetDateFormatter returns a date formatter function for the given locales and date skeleton
// TypeScript original code:
// export function getDateFormatter(
//
//	locales: string | string[],
//	tokens: string | DateToken[],
//	timeZone?: string | ((error: DateFormatError) => void),
//	onError?: (error: DateFormatError) => void
//
// ): (date: Date | number) => string
func GetDateFormatter(locales any, skeleton string, timeZone string, onError func(*DateFormatError)) (func(any) (string, error), error) {
	tokens := ParseDateTokens(skeleton)

	// Convert DateFormatError callback to internal format
	var internalOnError func(string, string, DateToken)
	if onError != nil {
		internalOnError = func(errorType, message string, token DateToken) {
			err := NewDateFormatError(errorType, message, token)
			onError(err)
		}
	}

	options := GetDateTimeFormatOptions(tokens, internalOnError)
	if timeZone != "" {
		options.TimeZone = timeZone
	}

	// For Go implementation, we'll create a simple formatter function
	// In a real implementation, this would use Go's time formatting with the converted options
	formatter := func(date any) (string, error) {
		// This is a simplified implementation
		// In practice, you'd use Go's time.Format with the appropriate layout
		// converted from the DateTimeFormatOptions

		// For now, return a placeholder that shows the skeleton was parsed
		return fmt.Sprintf("DateFormatter[%s](%v)", skeleton, date), nil
	}

	return formatter, nil
}

// GetDateFormatterSource returns JavaScript source for a date formatter
// TypeScript original code:
// export function getDateFormatterSource(
//
//	locales: string | string[],
//	tokens: string | DateToken[],
//	timeZone?: string | ((err: DateFormatError) => void),
//	onError?: (err: DateFormatError) => void
//
// ): string
func GetDateFormatterSource(locales any, skeleton string, timeZone string, onError func(*DateFormatError)) (string, error) {
	tokens := ParseDateTokens(skeleton)

	// Convert DateFormatError callback to internal format
	var internalOnError func(string, string, DateToken)
	if onError != nil {
		internalOnError = func(errorType, message string, token DateToken) {
			err := NewDateFormatError(errorType, message, token)
			onError(err)
		}
	}

	options := GetDateTimeFormatOptions(tokens, internalOnError)
	if timeZone != "" {
		options.TimeZone = timeZone
	}

	// Convert locales to proper format
	var localesStr string
	switch v := locales.(type) {
	case string:
		localesStr = fmt.Sprintf(`"%s"`, v)
	case []string:
		localesStr = fmt.Sprintf(`[%s]`, strings.Join(v, ","))
	default:
		return "", ErrInvalidType
	}

	// Generate JavaScript source (simplified)
	source := fmt.Sprintf(`(function() {
  var opt = %s;
  var dtf = new Intl.DateTimeFormat(%s, opt);
  return function(value) { return dtf.format(value); }
})()`, toJSON(options), localesStr)

	return source, nil
}

// Helper function to convert options to JSON (simplified)
func toJSON(options *DateTimeFormatOptions) string {
	// This is a simplified JSON conversion
	// In practice, you'd use proper JSON marshaling
	var parts []string

	if options.Era != "" {
		parts = append(parts, fmt.Sprintf(`"era":"%s"`, options.Era))
	}
	if options.Year != "" {
		parts = append(parts, fmt.Sprintf(`"year":"%s"`, options.Year))
	}
	if options.Month != "" {
		parts = append(parts, fmt.Sprintf(`"month":"%s"`, options.Month))
	}
	if options.Day != "" {
		parts = append(parts, fmt.Sprintf(`"day":"%s"`, options.Day))
	}
	if options.Weekday != "" {
		parts = append(parts, fmt.Sprintf(`"weekday":"%s"`, options.Weekday))
	}
	if options.Hour != "" {
		parts = append(parts, fmt.Sprintf(`"hour":"%s"`, options.Hour))
	}
	if options.Minute != "" {
		parts = append(parts, fmt.Sprintf(`"minute":"%s"`, options.Minute))
	}
	if options.Second != "" {
		parts = append(parts, fmt.Sprintf(`"second":"%s"`, options.Second))
	}
	if options.HourCycle != "" {
		parts = append(parts, fmt.Sprintf(`"hourCycle":"%s"`, options.HourCycle))
	}
	if options.TimeZoneName != "" {
		parts = append(parts, fmt.Sprintf(`"timeZoneName":"%s"`, options.TimeZoneName))
	}
	if options.Calendar != "" {
		parts = append(parts, fmt.Sprintf(`"calendar":"%s"`, options.Calendar))
	}
	if options.TimeZone != "" {
		parts = append(parts, fmt.Sprintf(`"timeZone":"%s"`, options.TimeZone))
	}

	return "{" + strings.Join(parts, ",") + "}"
}
