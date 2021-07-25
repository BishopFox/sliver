package gojsonq

import "errors"

// option describes type for providing configuration options to JSONQ
type option struct {
	decoder   Decoder
	separator string
}

// OptionFunc represents a contract for option func, it basically set options to jsonq instance options
type OptionFunc func(*JSONQ) error

// SetDecoder take a custom decoder to decode JSON
// Deprecated - use WithDecoder
func SetDecoder(u Decoder) OptionFunc {
	return WithDecoder(u)
}

// SetSeparator set custom separator for traversing child node, default separator is DOT (.)
// Deprecated - use WithSeparator
func SetSeparator(s string) OptionFunc {
	return WithSeparator(s)
}

// WithDecoder take a custom decoder to decode JSON
func WithDecoder(u Decoder) OptionFunc {
	return func(j *JSONQ) error {
		if u == nil {
			return errors.New("decoder can not be nil")
		}
		j.option.decoder = u
		return nil
	}
}

// WithSeparator set custom separator for traversing child node, default separator is DOT (.)
func WithSeparator(s string) OptionFunc {
	return func(j *JSONQ) error {
		if s == "" {
			return errors.New("separator can not be empty")
		}
		j.option.separator = s
		return nil
	}
}
