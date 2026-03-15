// formatters.go - Built-in MessageFormat formatters
// TypeScript original code: /packages/runtime/src/fmt/ module
package v1

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// NumberFmt formats numbers with specified parameters
// TypeScript original code:
// export function numberFmt(
//
//	value: number,
//	lc: string | string[],
//	arg: string,
//	defaultCurrency: string
//
//	) {
//	  const [type, currency] = (arg && arg.split(':')) || [];
//	  const opt: Record<string, Intl.NumberFormatOptions | undefined> = {
//	    integer: { maximumFractionDigits: 0 },
//	    percent: { style: 'percent' },
//	    currency: {
//	      style: 'currency',
//	      currency: (currency && currency.trim()) || defaultCurrency,
//	      minimumFractionDigits: 2,
//	      maximumFractionDigits: 2
//	    }
//	  };
//	  return nf(lc, opt[type] || {}).format(value);
//	}
func NumberFmt(value any, lc string, arg string, defaultCurrency string) (string, error) {
	numValue, err := toFloat64(value)
	if err != nil {
		return "", WrapInvalidNumberValue(value)
	}

	var formatType, currency string
	if arg != "" {
		parts := strings.Split(arg, ":")
		formatType = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			currency = strings.TrimSpace(parts[1])
		}
	}

	if currency == "" {
		currency = defaultCurrency
	}

	switch formatType {
	case "integer":
		return NumberInteger(numValue, lc), nil
	case "percent":
		return NumberPercent(numValue, lc), nil
	case "currency":
		return NumberCurrency(numValue, lc, currency), nil
	default:
		tag, _ := language.Parse(lc)
		printer := message.NewPrinter(tag)
		return printer.Sprintf("%.10g", numValue), nil
	}
}

// NumberCurrency formats a number as currency
// TypeScript original code:
// export const numberCurrency = (
//
//	value: number,
//	lc: string | string[],
//	arg: string
//
// ) =>
//
//	nf(lc, {
//	  style: 'currency',
//	  currency: arg,
//	  minimumFractionDigits: 2,
//	  maximumFractionDigits: 2
//	}).format(value);
func NumberCurrency(value any, lc string, currencyCode string) string {
	numValue, err := toFloat64(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	tag, err := language.Parse(lc)
	if err != nil {
		tag = language.English
	}

	printer := message.NewPrinter(tag)

	// Simple currency symbol mapping - this should be enhanced for full locale support
	symbol := "$" // default USD
	switch strings.ToUpper(currencyCode) {
	case "EUR":
		symbol = "€"
	case "GBP":
		symbol = "£"
	case "JPY":
		symbol = "¥"
	case "USD":
		symbol = "$"
	}

	return printer.Sprintf("%s%.2f", symbol, numValue)
}

// NumberInteger formats a number as integer
// TypeScript original code:
// export const numberInteger = (value: number, lc: string | string[]) =>
//
//	nf(lc, { maximumFractionDigits: 0 }).format(value);
func NumberInteger(value any, lc string) string {
	numValue, err := toFloat64(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	tag, err := language.Parse(lc)
	if err != nil {
		tag = language.English
	}

	// Use locale-specific integer formatting
	printer := message.NewPrinter(tag)
	return printer.Sprintf("%.0f", numValue)
}

// NumberPercent formats a number as percentage
// TypeScript original code:
// export const numberPercent = (value: number, lc: string | string[]) =>
//
//	nf(lc, { style: 'percent' }).format(value);
func NumberPercent(value any, lc string) string {
	numValue, err := toFloat64(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	tag, err := language.Parse(lc)
	if err != nil {
		tag = language.English
	}

	// Convert to percentage (multiply by 100)
	percentage := numValue * 100
	printer := message.NewPrinter(tag)
	return printer.Sprintf("%.0f%%", percentage)
}

// DateFormatter formats dates with locale-specific formatting
// TypeScript original code:
// export function date(
//
//	value: number | string,
//	lc: string | string[],
//	size?: 'short' | 'default' | 'long' | 'full'
//
//	) {
//	  const o: Intl.DateTimeFormatOptions = {
//	    day: 'numeric',
//	    month: 'short',
//	    year: 'numeric'
//	  };
//	  switch (size) {
//	    case 'full':
//	      o.weekday = 'long';
//	    case 'long':
//	      o.month = 'long';
//	      break;
//	    case 'short':
//	      o.month = 'numeric';
//	  }
//	  return new Date(value).toLocaleDateString(lc, o);
//	}
func DateFormatter(value any, lc string, size string) (string, error) {
	var t time.Time

	switch v := value.(type) {
	case int64:
		// Unix timestamp in milliseconds
		t = time.Unix(v/1000, (v%1000)*1000000)
	case int:
		t = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
	case float64:
		t = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
	case string:
		// Parse string as date - try multiple common formats
		var err error
		formats := []string{
			time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
			"2006-01-02",          // "2023-12-25"
			"01/02/2006",          // "12/25/2023"
			"2006-01-02 15:04:05", // "2023-12-25 00:00:00"
		}

		for _, format := range formats {
			if t, err = time.Parse(format, v); err == nil {
				break
			}
		}

		if err != nil {
			// Try parsing as Unix timestamp
			if timestamp, parseErr := strconv.ParseInt(v, 10, 64); parseErr == nil {
				t = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
			} else {
				return "", WrapInvalidDateValue(v)
			}
		}
	case time.Time:
		t = v
	default:
		return "", WrapInvalidType(fmt.Sprintf("%T", value))
	}

	// Format based on size parameter - matching TypeScript behavior
	switch size {
	case "short":
		return t.Format("1/2/2006"), nil
	case "long":
		return t.Format("January 2, 2006"), nil
	case "full":
		return t.Format("Monday, January 2, 2006"), nil
	default: // "default"
		return t.Format("Jan 2, 2006"), nil
	}
}

// TimeFormatter formats time values
// TypeScript original code: Similar to date formatter but for time
func TimeFormatter(value any, lc string, size string) (string, error) {
	var t time.Time

	switch v := value.(type) {
	case int64:
		t = time.Unix(v/1000, (v%1000)*1000000)
	case int:
		t = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
	case float64:
		t = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000)
	case string:
		// Parse string as time - try multiple common formats
		var err error
		formats := []string{
			time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
			"2006-01-02",          // "2023-12-25"
			"01/02/2006",          // "12/25/2023"
			"2006-01-02 15:04:05", // "2023-12-25 00:00:00"
		}

		for _, format := range formats {
			if t, err = time.Parse(format, v); err == nil {
				break
			}
		}

		if err != nil {
			if timestamp, parseErr := strconv.ParseInt(v, 10, 64); parseErr == nil {
				t = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
			} else {
				return "", WrapInvalidTimeValue(v)
			}
		}
	case time.Time:
		t = v
	default:
		return "", WrapInvalidType(fmt.Sprintf("%T", value))
	}

	// Format based on size parameter
	switch size {
	case "short":
		return t.Format("3:04 PM"), nil
	case "long":
		return t.Format("3:04:05 PM MST"), nil
	case "full":
		return t.Format("3:04:05 PM MST"), nil
	default: // "default"
		return t.Format("3:04:05 PM"), nil
	}
}

// GetFormatter returns a formatter function by name
func GetFormatter(name string) func(any, string, string) (string, error) {
	switch name {
	case "number":
		return func(value any, lc string, arg string) (string, error) {
			return NumberFmt(value, lc, arg, "USD")
		}
	case "date":
		return DateFormatter
	case "time":
		return TimeFormatter
	default:
		return nil
	}
}
