// Package v1 provides MessageFormat v1 (ICU MessageFormat) implementation for Go
// TypeScript original code:
// export type ReturnType = 'string' | 'values';
// export type MessageFunction<ReturnType extends 'string' | 'values'> = (
//
//	param?: Record<string, unknown> | unknown[]
//
// ) => ReturnType extends 'string' ? string : unknown[];
package v1

// Type-safe constants for API options (replacing string literals)
// This provides better IDE support, compile-time checking, and prevents typos

// ReturnType represents the return type of compiled message functions
type ReturnType string

const (
	// ReturnTypeString indicates compiled functions return concatenated strings
	ReturnTypeString ReturnType = "string"
	// ReturnTypeValues indicates compiled functions return arrays of values
	ReturnTypeValues ReturnType = "values"
)

// SignDisplay represents number sign display options
type SignDisplay string

const (
	// SignAuto shows sign automatically (negative only)
	SignAuto SignDisplay = "sign-auto"
	// SignAlways shows sign always (+ for positive, - for negative)
	SignAlways SignDisplay = "sign-always"
	// SignNever never shows sign
	SignNever SignDisplay = "sign-never"
	// SignAccounting uses accounting notation for negative numbers
	SignAccounting SignDisplay = "sign-accounting"
	// SignAccountingAlways uses accounting notation always
	SignAccountingAlways SignDisplay = "sign-accounting-always"
	// SignExceptZero shows sign except for zero
	SignExceptZero SignDisplay = "sign-except-zero"
	// SignAccountingExceptZero uses accounting notation except for zero
	SignAccountingExceptZero SignDisplay = "sign-accounting-except-zero"
	// SignNegative shows sign only for negative numbers
	SignNegative SignDisplay = "sign-negative"
	// SignAccountingNegative uses accounting notation only for negative numbers
	SignAccountingNegative SignDisplay = "sign-accounting-negative"
)

// GroupDisplay represents number grouping display options
type GroupDisplay string

const (
	// GroupOff disables grouping separators
	GroupOff GroupDisplay = "group-off"
	// GroupMin2 enables grouping only for numbers with 4+ digits
	GroupMin2 GroupDisplay = "group-min2"
	// GroupAuto enables grouping automatically based on locale
	GroupAuto GroupDisplay = "group-auto"
	// GroupOnAligned enables grouping with aligned separators
	GroupOnAligned GroupDisplay = "group-on-aligned"
	// GroupThousands enables thousands separators
	GroupThousands GroupDisplay = "group-thousands"
)

// DecimalDisplay represents decimal point display options
type DecimalDisplay string

const (
	// DecimalAuto shows decimal point automatically when needed
	DecimalAuto DecimalDisplay = "decimal-auto"
	// DecimalAlways always shows decimal point
	DecimalAlways DecimalDisplay = "decimal-always"
)

// UnitStyle represents unit formatting style options
type UnitStyle string

const (
	// UnitPercent formats as percentage (%)
	UnitPercent UnitStyle = "percent"
	// UnitPermille formats as permille (‰)
	UnitPermille UnitStyle = "permille"
	// UnitBaseUnit formats as base unit (no unit symbol)
	UnitBaseUnit UnitStyle = "base-unit"
	// UnitCurrency formats as currency
	UnitCurrency UnitStyle = "currency"
	// UnitMeasureUnit formats with measurement units (km, kg, etc.)
	UnitMeasureUnit UnitStyle = "measure-unit"
	// UnitConciseUnit formats with concise unit notation
	UnitConciseUnit UnitStyle = "concise-unit"
)

// NotationStyle represents number notation style options
type NotationStyle string

const (
	// NotationSimple uses simple number notation
	NotationSimple NotationStyle = "notation-simple"
	// NotationCompactShort uses short compact notation (1K, 1M)
	NotationCompactShort NotationStyle = "compact-short"
	// NotationCompactLong uses long compact notation (1 thousand, 1 million)
	NotationCompactLong NotationStyle = "compact-long"
	// NotationScientific uses scientific notation (1.23E+4)
	NotationScientific NotationStyle = "scientific"
	// NotationEngineering uses engineering notation (12.3E+3)
	NotationEngineering NotationStyle = "engineering"
)

// RoundingMode represents number rounding mode options
type RoundingMode string

const (
	// RoundingCeiling rounds towards positive infinity
	RoundingCeiling RoundingMode = "rounding-mode-ceiling"
	// RoundingFloor rounds towards negative infinity
	RoundingFloor RoundingMode = "rounding-mode-floor"
	// RoundingDown rounds towards zero
	RoundingDown RoundingMode = "rounding-mode-down"
	// RoundingUp rounds away from zero
	RoundingUp RoundingMode = "rounding-mode-up"
	// RoundingHalfEven rounds to nearest, ties to even (banker's rounding)
	RoundingHalfEven RoundingMode = "rounding-mode-half-even"
	// RoundingHalfOdd rounds to nearest, ties to odd
	RoundingHalfOdd RoundingMode = "rounding-mode-half-odd"
	// RoundingHalfCeiling rounds to nearest, ties towards positive infinity
	RoundingHalfCeiling RoundingMode = "rounding-mode-half-ceiling"
	// RoundingHalfFloor rounds to nearest, ties towards negative infinity
	RoundingHalfFloor RoundingMode = "rounding-mode-half-floor"
	// RoundingHalfDown rounds to nearest, ties towards zero
	RoundingHalfDown RoundingMode = "rounding-mode-half-down"
	// RoundingHalfUp rounds to nearest, ties away from zero
	RoundingHalfUp RoundingMode = "rounding-mode-half-up"
	// RoundingUnnecessary requires exact representation (throws error if rounding needed)
	RoundingUnnecessary RoundingMode = "rounding-mode-unnecessary"
)

// UnitWidth represents unit width display options
type UnitWidth string

const (
	// UnitWidthNarrow uses narrow unit symbols (e.g., $, €, ¥)
	UnitWidthNarrow UnitWidth = "unit-width-narrow"
	// UnitWidthShort uses short unit symbols (e.g., USD, EUR, JPY)
	UnitWidthShort UnitWidth = "unit-width-short"
	// UnitWidthFullName uses full unit names (e.g., US Dollar, Euro)
	UnitWidthFullName UnitWidth = "unit-width-full-name"
	// UnitWidthIsoCode uses ISO unit codes
	UnitWidthIsoCode UnitWidth = "unit-width-iso-code"
	// UnitWidthHidden hides the unit symbol
	UnitWidthHidden UnitWidth = "unit-width-hidden"
)

// PrecisionStyle represents precision style options
type PrecisionStyle string

const (
	// PrecisionInteger shows only integer part
	PrecisionInteger PrecisionStyle = "precision-integer"
	// PrecisionUnlimited allows unlimited decimal places
	PrecisionUnlimited PrecisionStyle = "precision-unlimited"
	// PrecisionCurrencyStandard uses standard currency precision
	PrecisionCurrencyStandard PrecisionStyle = "precision-currency-standard"
	// PrecisionCurrencyCash uses cash currency precision (rounding to physical denominations)
	PrecisionCurrencyCash PrecisionStyle = "precision-currency-cash"
	// PrecisionIncrement uses increment-based precision
	PrecisionIncrement PrecisionStyle = "precision-increment"
	// PrecisionFraction uses fractional precision
	PrecisionFraction PrecisionStyle = "precision-fraction"
	// PrecisionSignificant uses significant digits precision
	PrecisionSignificant PrecisionStyle = "precision-significant"
)

// TrailingZeroDisplay represents trailing zero display options
type TrailingZeroDisplay string

const (
	// TrailingZeroAuto shows trailing zeros automatically based on precision
	TrailingZeroAuto TrailingZeroDisplay = "auto"
	// TrailingZeroStripIfInteger strips trailing zeros if the result is an integer
	TrailingZeroStripIfInteger TrailingZeroDisplay = "stripIfInteger"
)

// RoundingPriority represents rounding priority options
type RoundingPriority string

const (
	// RoundingPriorityRelaxed allows relaxed rounding rules
	RoundingPriorityRelaxed RoundingPriority = "relaxed"
	// RoundingPriorityStrict requires strict adherence to rounding rules
	RoundingPriorityStrict RoundingPriority = "strict"
)

// PluralKeyMode represents strict plural key checking mode
// This handles the special case where StrictPluralKeys defaults to true
type PluralKeyMode int

const (
	// PluralKeyModeDefault uses the default behavior (strict=true)
	PluralKeyModeDefault PluralKeyMode = iota
	// PluralKeyModeStrict explicitly enables strict plural key checking
	PluralKeyModeStrict
	// PluralKeyModeRelaxed explicitly disables strict plural key checking
	PluralKeyModeRelaxed
)

// Ptr returns a pointer to the given value.
//
// This is a generic helper function for creating pointers to any type.
// It's particularly useful when you need to pass pointer values to optional
// struct fields or function parameters.
//
// Examples:
//
//	// With type constants
//	signDisplay := Ptr(SignAuto)
//	groupDisplay := Ptr(GroupOff)
//
//	// With literal values
//	name := Ptr("John")
//	age := Ptr(25)
//	enabled := Ptr(true)
//
//	// In struct initialization
//	config := Config{
//	    ReturnType: Ptr(ReturnTypeValues),
//	    Strict:     Ptr(true),
//	}
//
//go:fix inline
func Ptr[T any](v T) *T {
	return &v
}
