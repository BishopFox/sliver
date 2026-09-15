package types

import (
	"encoding/json"
	"errors"
	"net/mail"
)

// ErrValidationEmail is the sentinel error returned when an email fails validation
var ErrValidationEmail = errors.New("email: failed to pass regex validation")

// Email represents an email address.
// It is a string type that must pass regex validation before being marshalled
// to JSON or unmarshalled from JSON.
type Email string

func (e Email) MarshalJSON() ([]byte, error) {
	m, err := mail.ParseAddress(string(e))
	if err != nil {
		return nil, ErrValidationEmail
	}

	return json.Marshal(m.Address)
}

func (e *Email) UnmarshalJSON(data []byte) error {
	if e == nil {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	m, err := mail.ParseAddress(s)
	if err != nil {
		return ErrValidationEmail
	}

	*e = Email(m.Address)
	return nil
}
