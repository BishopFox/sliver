package jsonschema

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/go-json-experiment/json"
)

// Rat wraps a big.Rat to enable custom JSON marshaling and unmarshaling.
type Rat struct {
	*big.Rat
}

// UnmarshalJSON implements the json.Unmarshaler interface for Rat.
func (r *Rat) UnmarshalJSON(data []byte) error {
	var tmp any
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	converted, err := convertToBigRat(tmp)
	if err != nil {
		return err
	}

	r.Rat = converted
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Rat.
func (r *Rat) MarshalJSON() ([]byte, error) {
	formattedValue := FormatRat(r)
	if strings.Contains(formattedValue, "/") {
		// Output as a JSON string if it still contains a fraction
		return json.Marshal(formattedValue)
	}
	// Output as a JSON number
	return []byte(formattedValue), nil
}

// convertToBigRat converts various types to big.Rat.
func convertToBigRat(data any) (*big.Rat, error) {
	var str string
	switch v := data.(type) {
	case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		str = fmt.Sprint(v)
	case string:
		str = v
	default:
		return nil, ErrUnsupportedTypeForRat
	}

	numRat := new(big.Rat)
	if _, ok := numRat.SetString(str); !ok {
		return nil, ErrRatConversion
	}
	return numRat, nil
}

// NewRat creates a new Rat instance from a given value.
func NewRat(value any) *Rat {
	converted, err := convertToBigRat(value)
	if err != nil {
		return nil
	}
	return &Rat{converted}
}

// FormatRat formats a Rat as a string.
func FormatRat(r *Rat) string {
	if r == nil {
		return "null"
	}

	// Check if the Rat is an integer
	if r.IsInt() {
		return r.Num().String() // Output as a plain integer string
	}

	// Format as a decimal maintaining precision
	dec := r.FloatString(10) // You might adjust precision as needed

	// Trim unnecessary trailing zeros and decimal point if no fractional part
	trimmedDec := strings.TrimRight(dec, "0")
	trimmedDec = strings.TrimRight(trimmedDec, ".")

	if trimmedDec == "" {
		return "0" // correct trimming edge case of "0.0000"
	}

	return trimmedDec
}
