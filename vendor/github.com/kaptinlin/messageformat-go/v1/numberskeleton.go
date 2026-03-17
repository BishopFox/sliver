// Package v1 provides MessageFormat v1 (ICU MessageFormat) implementation for Go
package v1

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled regular expressions for performance
var (
	expDigitsRegex = regexp.MustCompile(`^[+*]e+$`)
)

// Skeleton represents an object representation of a parsed string skeleton with token values grouped by type
// TypeScript original code:
//
//	export interface Skeleton {
//	  affix?: { pos: [string, string]; neg?: [string, string] };
//	  decimal?: 'decimal-auto' | 'decimal-always';
//	  group?: 'group-off' | 'group-min2' | 'group-auto' | 'group-on-aligned' | 'group-thousands';
//	  integerWidth?: { min: number; max?: number; source?: string };
//	  notation?: { style: 'compact-short' | 'compact-long' | 'notation-simple' } | { ... };
//	  numberingSystem?: string;
//	  precision?: { ... };
//	  roundingMode?: '...' | '...';
//	  scale?: number;
//	  sign?: '...' | '...';
//	  unit?: { style: 'percent' | 'permille' | 'base-unit' } | { ... };
//	  unitPer?: string;
//	  unitWidth?: '...' | '...';
//	}
type Skeleton struct {
	Affix           *AffixConfig     `json:"affix,omitempty"`
	Decimal         DecimalDisplay   `json:"decimal,omitempty"` // Type-safe decimal constants
	Group           GroupDisplay     `json:"group,omitempty"`   // Type-safe group constants
	IntegerWidth    *IntegerWidth    `json:"integerWidth,omitempty"`
	Notation        *NotationConfig  `json:"notation,omitempty"`
	NumberingSystem *string          `json:"numberingSystem,omitempty"`
	Precision       *PrecisionConfig `json:"precision,omitempty"`
	RoundingMode    RoundingMode     `json:"roundingMode,omitempty"` // Type-safe rounding constants
	Scale           *int             `json:"scale,omitempty"`
	Sign            SignDisplay      `json:"sign,omitempty"` // Type-safe sign constants
	Unit            *UnitConfig      `json:"unit,omitempty"`
	UnitPer         *string          `json:"unitPer,omitempty"`
	UnitWidth       UnitWidth        `json:"unitWidth,omitempty"` // Type-safe unit width constants
}

// AffixConfig represents prefix and suffix configuration
// TypeScript original code:
// affix?: { pos: [string, string]; neg?: [string, string] };
type AffixConfig struct {
	Pos []string  `json:"pos"`           // [prefix, suffix]
	Neg *[]string `json:"neg,omitempty"` // [prefix, suffix] for negative numbers
}

// IntegerWidth represents integer width configuration
// TypeScript original code:
// integerWidth?: { min: number; max?: number; source?: string };
type IntegerWidth struct {
	Min    int     `json:"min"`
	Max    *int    `json:"max,omitempty"`
	Source *string `json:"source,omitempty"`
}

// NotationConfig represents notation configuration with type-safe constants
// TypeScript original code:
// notation?: { style: 'compact-short' | 'compact-long' | 'notation-simple' } |
//
//	{ style: 'scientific' | 'engineering'; expDigits?: number; expSign?: Skeleton['sign']; source?: string; }
type NotationConfig struct {
	Style     NotationStyle `json:"style"` // Type-safe notation style constants
	ExpDigits *int          `json:"expDigits,omitempty"`
	ExpSign   SignDisplay   `json:"expSign,omitempty"` // Type-safe sign constants
	Source    *string       `json:"source,omitempty"`
}

// PrecisionConfig represents precision configuration with type-safe constants
// TypeScript original code:
// precision?: { style: '...' | '...' | '...' | '...'; trailingZero?: 'auto' | 'stripIfInteger'; } |
//
//	{ style: 'precision-increment'; increment: number; trailingZero?: 'auto' | 'stripIfInteger'; } |
//	{ style: 'precision-fraction'; minFraction?: number; maxFraction?: number; ... }
type PrecisionConfig struct {
	Style            PrecisionStyle      `json:"style"`
	Increment        *int                `json:"increment,omitempty"`
	MinFraction      *int                `json:"minFraction,omitempty"`
	MaxFraction      *int                `json:"maxFraction,omitempty"`
	MinSignificant   *int                `json:"minSignificant,omitempty"`
	MaxSignificant   *int                `json:"maxSignificant,omitempty"`
	RoundingPriority RoundingPriority    `json:"roundingPriority,omitempty"` // Type-safe rounding priority constants
	TrailingZero     TrailingZeroDisplay `json:"trailingZero,omitempty"`     // Type-safe trailing zero constants
	Source           *string             `json:"source,omitempty"`
}

// UnitConfig represents unit configuration with type-safe constants
// TypeScript original code:
// unit?: { style: 'percent' | 'permille' | 'base-unit' } |
//
//	{ style: 'currency'; currency: string } |
//	{ style: 'measure-unit'; unit: string } |
//	{ style: 'concise-unit'; unit: string }
type UnitConfig struct {
	Style    UnitStyle `json:"style"`              // Type-safe unit style constants
	Currency *string   `json:"currency,omitempty"` // For currency style
	Unit     *string   `json:"unit,omitempty"`     // For measure-unit or concise-unit style
}

// NumberSkeletonError represents errors during skeleton parsing
// TypeScript original code:
// Various error types: BadOptionError, BadStemError, MaskedValueError, etc.
type NumberSkeletonError struct {
	Type    string
	Message string
	Stem    string
	Option  string
}

func (e *NumberSkeletonError) Error() string {
	return fmt.Sprintf("NumberSkeleton %s: %s", e.Type, e.Message)
}

// TokenParser represents a number skeleton token parser
// TypeScript original code:
//
//	export class TokenParser {
//	  #error: (err: NumberFormatError) => void;
//	  skeleton: Skeleton = {};
//	  constructor(onError: (err: NumberFormatError) => void)
//	  parseToken(stem: string, options: string[])
//	}
type TokenParser struct {
	skeleton Skeleton
	onError  func(error)
}

// NewTokenParser creates a new token parser
// TypeScript original code:
//
//	constructor(onError: (err: NumberFormatError) => void) {
//	  this.#error = onError;
//	}
func NewTokenParser(onError func(error)) *TokenParser {
	return &TokenParser{
		skeleton: Skeleton{},
		onError:  onError,
	}
}

// ParseNumberSkeleton parses an ICU NumberFormatter skeleton string into a Skeleton structure
// TypeScript original code:
// export function parseNumberSkeleton(
//
//	src: string,
//	onError: (err: NumberFormatError) => void = error => { throw error; }
//
//	): Skeleton {
//	  const parser = new TokenParser(onError);
//	  for (const part of src.split(' ')) {
//	    if (part) {
//	      const [stem, ...options] = part.split('/');
//	      parser.parseToken(stem, options);
//	    }
//	  }
//	  return parser.skeleton;
//	}
func ParseNumberSkeleton(src string, onError ...func(error)) (Skeleton, error) {
	var errorHandler func(error)
	if len(onError) > 0 && onError[0] != nil {
		errorHandler = onError[0]
	} else {
		// Default error handler - collect errors and return at the end
		var errors []error
		errorHandler = func(err error) {
			errors = append(errors, err)
		}
		defer func() {
			if len(errors) > 0 {
				// Return first error if any
				panic(errors[0])
			}
		}()
	}

	parser := NewTokenParser(errorHandler)
	parts := strings.Fields(src) // Split by whitespace

	for _, part := range parts {
		if part != "" {
			components := strings.Split(part, "/")
			stem := components[0]
			options := components[1:]
			parser.ParseToken(stem, options)
		}
	}

	return parser.skeleton, nil
}

// ParseToken parses a single skeleton token
// TypeScript original code:
// parseToken(stem: string, options: string[])
func (tp *TokenParser) ParseToken(stem string, options []string) {
	// Helper function for validation
	ok := func(fieldName string, min, max int) bool {
		_ = fieldName // Currently unused, but kept for future masked value validation
		if len(options) > max {
			if max == 0 {
				for _, opt := range options {
					tp.onError(&NumberSkeletonError{
						Type:    "BadOption",
						Message: fmt.Sprintf("Unexpected option %s for token %s", opt, stem),
						Stem:    stem,
						Option:  opt,
					})
				}
			} else {
				tp.onError(&NumberSkeletonError{
					Type:    "TooManyOptions",
					Message: fmt.Sprintf("Too many options for token %s (expected max %d, got %d)", stem, max, len(options)),
					Stem:    stem,
				})
			}
			return false
		} else if len(options) < min {
			tp.onError(&NumberSkeletonError{
				Type:    "MissingOption",
				Message: fmt.Sprintf("Missing required option for token %s", stem),
				Stem:    stem,
			})
			return false
		}
		return true
	}

	switch stem {
	// Notation tokens with type-safe constants
	case "compact-short":
		if ok("notation", 0, 0) {
			tp.skeleton.Notation = &NotationConfig{Style: NotationCompactShort}
		}
	case "compact-long":
		if ok("notation", 0, 0) {
			tp.skeleton.Notation = &NotationConfig{Style: NotationCompactLong}
		}
	case "notation-simple":
		if ok("notation", 0, 0) {
			tp.skeleton.Notation = &NotationConfig{Style: NotationSimple}
		}
	case "K":
		if ok("notation", 0, 0) {
			tp.skeleton.Notation = &NotationConfig{Style: NotationCompactShort}
		}
	case "KK":
		if ok("notation", 0, 0) {
			tp.skeleton.Notation = &NotationConfig{Style: NotationCompactLong}
		}
	case "scientific":
		if !ok("notation", 0, 2) {
			return
		}
		notation := &NotationConfig{Style: NotationScientific}

		for _, opt := range options {
			switch opt {
			case "sign-auto":
				notation.ExpSign = SignAuto
			case "sign-always":
				notation.ExpSign = SignAlways
			case "sign-never":
				notation.ExpSign = SignNever
			case "sign-accounting":
				notation.ExpSign = SignAccounting
			case "sign-accounting-always":
				notation.ExpSign = SignAccountingAlways
			case "sign-except-zero":
				notation.ExpSign = SignExceptZero
			case "sign-accounting-except-zero":
				notation.ExpSign = SignAccountingExceptZero
			default:
				// Check for exponent digits pattern like "+e" or "*ee"
				if expDigitsRegex.MatchString(opt) {
					expDigits := len(opt) - 1
					notation.ExpDigits = &expDigits
				} else {
					tp.onError(&NumberSkeletonError{
						Type:    "BadOption",
						Message: fmt.Sprintf("Invalid option %s for token %s", opt, stem),
						Stem:    stem,
						Option:  opt,
					})
				}
			}
		}
		tp.skeleton.Notation = notation
	case "engineering":
		if !ok("notation", 0, 2) {
			return
		}
		notation := &NotationConfig{Style: NotationEngineering}

		for _, opt := range options {
			switch opt {
			case "sign-auto":
				notation.ExpSign = SignAuto
			case "sign-always":
				notation.ExpSign = SignAlways
			case "sign-never":
				notation.ExpSign = SignNever
			case "sign-accounting":
				notation.ExpSign = SignAccounting
			case "sign-accounting-always":
				notation.ExpSign = SignAccountingAlways
			case "sign-except-zero":
				notation.ExpSign = SignExceptZero
			case "sign-accounting-except-zero":
				notation.ExpSign = SignAccountingExceptZero
			default:
				// Check for exponent digits pattern like "+e" or "*ee"
				if expDigitsRegex.MatchString(opt) {
					expDigits := len(opt) - 1
					notation.ExpDigits = &expDigits
				} else {
					tp.onError(&NumberSkeletonError{
						Type:    "BadOption",
						Message: fmt.Sprintf("Invalid option %s for token %s", opt, stem),
						Stem:    stem,
						Option:  opt,
					})
				}
			}
		}
		tp.skeleton.Notation = notation

	// Unit tokens with type-safe constants
	case "percent":
		if ok("unit", 0, 0) {
			tp.skeleton.Unit = &UnitConfig{Style: UnitPercent}
		}
	case "permille":
		if ok("unit", 0, 0) {
			tp.skeleton.Unit = &UnitConfig{Style: UnitPermille}
		}
	case "base-unit":
		if ok("unit", 0, 0) {
			tp.skeleton.Unit = &UnitConfig{Style: UnitBaseUnit}
		}
	case "currency":
		if ok("unit", 1, 1) {
			tp.skeleton.Unit = &UnitConfig{
				Style:    UnitCurrency,
				Currency: &options[0],
			}
		}
	case "measure-unit":
		if ok("unit", 1, 1) {
			tp.skeleton.Unit = &UnitConfig{
				Style: UnitMeasureUnit,
				Unit:  &options[0],
			}
		}
	case "concise-unit":
		if ok("unit", 1, 1) {
			tp.skeleton.Unit = &UnitConfig{
				Style: UnitConciseUnit,
				Unit:  &options[0],
			}
		}

	// Precision tokens with type-safe constants
	case "precision-integer":
		if ok("precision", 0, 1) {
			precision := &PrecisionConfig{Style: PrecisionInteger}
			if len(options) > 0 {
				switch options[0] {
				case "auto":
					precision.TrailingZero = TrailingZeroAuto
				case "stripIfInteger":
					precision.TrailingZero = TrailingZeroStripIfInteger
				}
			}
			tp.skeleton.Precision = precision
		}
	case "precision-unlimited":
		if ok("precision", 0, 1) {
			precision := &PrecisionConfig{Style: PrecisionUnlimited}
			if len(options) > 0 {
				switch options[0] {
				case "auto":
					precision.TrailingZero = TrailingZeroAuto
				case "stripIfInteger":
					precision.TrailingZero = TrailingZeroStripIfInteger
				}
			}
			tp.skeleton.Precision = precision
		}
	case "precision-currency-standard":
		if ok("precision", 0, 1) {
			precision := &PrecisionConfig{Style: PrecisionCurrencyStandard}
			if len(options) > 0 {
				switch options[0] {
				case "auto":
					precision.TrailingZero = TrailingZeroAuto
				case "stripIfInteger":
					precision.TrailingZero = TrailingZeroStripIfInteger
				}
			}
			tp.skeleton.Precision = precision
		}
	case "precision-currency-cash":
		if ok("precision", 0, 1) {
			precision := &PrecisionConfig{Style: PrecisionCurrencyCash}
			if len(options) > 0 {
				switch options[0] {
				case "auto":
					precision.TrailingZero = TrailingZeroAuto
				case "stripIfInteger":
					precision.TrailingZero = TrailingZeroStripIfInteger
				}
			}
			tp.skeleton.Precision = precision
		}
	case "precision-increment":
		if ok("precision", 1, 2) {
			increment, err := strconv.Atoi(options[0])
			if err != nil {
				tp.onError(&NumberSkeletonError{
					Type:    "BadOption",
					Message: fmt.Sprintf("Invalid increment value %s for precision-increment", options[0]),
					Stem:    stem,
					Option:  options[0],
				})
				return
			}
			precision := &PrecisionConfig{
				Style:     PrecisionIncrement,
				Increment: &increment,
			}
			if len(options) > 1 {
				switch options[1] {
				case "auto":
					precision.TrailingZero = TrailingZeroAuto
				case "stripIfInteger":
					precision.TrailingZero = TrailingZeroStripIfInteger
				}
			}
			tp.skeleton.Precision = precision
		}

	// Sign tokens with type-safe constants
	case "sign-auto":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAuto
		}
	case "sign-always":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAlways
		}
	case "sign-never":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignNever
		}
	case "sign-accounting":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAccounting
		}
	case "sign-accounting-always":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAccountingAlways
		}
	case "sign-except-zero":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignExceptZero
		}
	case "sign-accounting-except-zero":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAccountingExceptZero
		}
	case "sign-negative":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignNegative
		}
	case "sign-accounting-negative":
		if ok("sign", 0, 0) {
			tp.skeleton.Sign = SignAccountingNegative
		}

	// Group tokens with type-safe constants
	case "group-off":
		if ok("group", 0, 0) {
			tp.skeleton.Group = GroupOff
		}
	case "group-min2":
		if ok("group", 0, 0) {
			tp.skeleton.Group = GroupMin2
		}
	case "group-auto":
		if ok("group", 0, 0) {
			tp.skeleton.Group = GroupAuto
		}
	case "group-on-aligned":
		if ok("group", 0, 0) {
			tp.skeleton.Group = GroupOnAligned
		}
	case "group-thousands":
		if ok("group", 0, 0) {
			tp.skeleton.Group = GroupThousands
		}

	// Decimal tokens with type-safe constants
	case "decimal-auto":
		if ok("decimal", 0, 0) {
			tp.skeleton.Decimal = DecimalAuto
		}
	case "decimal-always":
		if ok("decimal", 0, 0) {
			tp.skeleton.Decimal = DecimalAlways
		}

	// Unit width tokens with type-safe constants
	case "unit-width-narrow":
		if ok("unitWidth", 0, 0) {
			tp.skeleton.UnitWidth = UnitWidthNarrow
		}
	case "unit-width-short":
		if ok("unitWidth", 0, 0) {
			tp.skeleton.UnitWidth = UnitWidthShort
		}
	case "unit-width-full-name":
		if ok("unitWidth", 0, 0) {
			tp.skeleton.UnitWidth = UnitWidthFullName
		}
	case "unit-width-iso-code":
		if ok("unitWidth", 0, 0) {
			tp.skeleton.UnitWidth = UnitWidthIsoCode
		}
	case "unit-width-hidden":
		if ok("unitWidth", 0, 0) {
			tp.skeleton.UnitWidth = UnitWidthHidden
		}

	// Rounding mode tokens with type-safe constants
	case "rounding-mode-ceiling":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingCeiling
		}
	case "rounding-mode-floor":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingFloor
		}
	case "rounding-mode-down":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingDown
		}
	case "rounding-mode-up":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingUp
		}
	case "rounding-mode-half-even":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfEven
		}
	case "rounding-mode-half-odd":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfOdd
		}
	case "rounding-mode-half-ceiling":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfCeiling
		}
	case "rounding-mode-half-floor":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfFloor
		}
	case "rounding-mode-half-down":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfDown
		}
	case "rounding-mode-half-up":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingHalfUp
		}
	case "rounding-mode-unnecessary":
		if ok("roundingMode", 0, 0) {
			tp.skeleton.RoundingMode = RoundingUnnecessary
		}

	// Scale token
	case "scale":
		if ok("scale", 1, 1) {
			scale, err := strconv.Atoi(options[0])
			if err != nil {
				tp.onError(&NumberSkeletonError{
					Type:    "BadOption",
					Message: fmt.Sprintf("Invalid scale value %s", options[0]),
					Stem:    stem,
					Option:  options[0],
				})
				return
			}
			tp.skeleton.Scale = &scale
		}

	// Integer width token
	case "integer-width":
		if ok("integerWidth", 1, 2) {
			// Parse min value
			min, err := strconv.Atoi(options[0])
			if err != nil {
				tp.onError(&NumberSkeletonError{
					Type:    "BadOption",
					Message: fmt.Sprintf("Invalid integer width min value %s", options[0]),
					Stem:    stem,
					Option:  options[0],
				})
				return
			}

			intWidth := &IntegerWidth{Min: min}
			if len(options) > 1 {
				max, err := strconv.Atoi(options[1])
				if err != nil {
					tp.onError(&NumberSkeletonError{
						Type:    "BadOption",
						Message: fmt.Sprintf("Invalid integer width max value %s", options[1]),
						Stem:    stem,
						Option:  options[1],
					})
					return
				}
				intWidth.Max = &max
			}
			tp.skeleton.IntegerWidth = intWidth
		}

	// Numbering system token
	case "numbering-system":
		if ok("numberingSystem", 1, 1) {
			tp.skeleton.NumberingSystem = &options[0]
		}

	default:
		tp.onError(&NumberSkeletonError{
			Type:    "BadStem",
			Message: fmt.Sprintf("Unknown skeleton token: %s", stem),
			Stem:    stem,
		})
	}
}

// Skeleton returns the parsed skeleton
func (tp *TokenParser) Skeleton() Skeleton {
	return tp.skeleton
}
