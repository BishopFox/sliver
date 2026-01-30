package mailgun

import (
	"fmt"
	"strings"
)

type Recipient struct {
	Name  string `json:"-"`
	Email string `json:"-"`
}

func (r Recipient) String() string {
	if r.Name != "" {
		return fmt.Sprintf("%s <%s>", r.Name, r.Email)
	}
	return r.Email
}

// MarshalText satisfies TextMarshaler
func (r Recipient) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (r *Recipient) UnmarshalText(text []byte) error {
	s := string(text)
	if s[len(s)-1:] != ">" {
		*r = Recipient{Email: s}
		return nil
	}

	i := strings.Index(s, "<")
	// at least 1 char followed by a space
	if i < 2 {
		return fmt.Errorf("malformed recipient string '%s'", s)
	}
	*r = Recipient{
		Name:  strings.TrimSpace(s[:i]),
		Email: s[i+1 : len(s)-1],
	}

	return nil
}
