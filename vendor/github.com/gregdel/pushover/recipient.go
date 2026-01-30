package pushover

import "regexp"

var recipientRegexp *regexp.Regexp

func init() {
	recipientRegexp = regexp.MustCompile(`^[A-Za-z0-9]{30}$`)
}

// Recipient represents a recipient to notify.
type Recipient struct {
	token string
}

// NewRecipient is the representation of the recipient to notify.
// A recipient may be a Group ID.
func NewRecipient(token string) *Recipient {
	return &Recipient{token}
}

// Validates recipient token.
func (r *Recipient) validate() error {
	// Check empty token
	if r.token == "" {
		return ErrEmptyRecipientToken
	}

	// Check invalid token
	if !recipientRegexp.MatchString(r.token) {
		return ErrInvalidRecipientToken
	}
	return nil
}

// RecipientDetails represents the receipt informations in case of emergency
// priority.
type RecipientDetails struct {
	Status    int      `json:"status"`
	Group     int      `json:"group"`
	Devices   []string `json:"devices"`
	RequestID string   `json:"request"`
	Errors    Errors   `json:"errors"`
}
