package mtypes

// Bounce aggregates data relating to undeliverable messages to a specific intended recipient,
// identified by Address.
type Bounce struct {
	// The time at which Mailgun detected the bounce.
	CreatedAt RFC2822Time `json:"created_at"`
	// Code provides the SMTP error code that caused the bounce
	Code string `json:"code"`
	// Address the bounce is for
	Address string `json:"address"`
	// human readable reason why
	Error string `json:"error"`
}

type BouncesListResponse struct {
	Items  []Bounce `json:"items"`
	Paging Paging   `json:"paging"`
}
