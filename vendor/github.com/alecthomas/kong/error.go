package kong

// ParseError is the error type returned by Kong.Parse().
//
// It contains the parse Context that triggered the error.
type ParseError struct {
	error
	Context *Context
}

// Unwrap returns the original cause of the error.
func (p *ParseError) Unwrap() error { return p.error }
