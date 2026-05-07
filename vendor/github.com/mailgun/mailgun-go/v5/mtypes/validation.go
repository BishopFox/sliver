package mtypes

// ValidateEmailResponse records basic facts about a validated e-mail address.
// See the ValidateEmail method and example for more details.
type ValidateEmailResponse struct {
	// Echoes the address provided.
	Address string `json:"address"`

	// Indicates whether Mailgun thinks the address is from a known
	// disposable mailbox provider.
	IsDisposableAddress bool `json:"is_disposable_address"`

	// Indicates whether Mailgun thinks the address is an email distribution list.
	IsRoleAddress bool `json:"is_role_address"`

	// A list of potential reasons why a specific validation may be unsuccessful.
	Reason []string `json:"reason"`

	// Result
	Result string `json:"result"`

	// Risk assessment for the provided email: low/medium/high/unknown.
	Risk string `json:"risk"`

	LastSeen int64 `json:"last_seen,omitempty"`

	// Provides a simple recommendation in case the address is invalid or
	// Mailgun thinks you might have a typo. May be empty, in which case
	// Mailgun has no recommendation to give.
	DidYouMean string `json:"did_you_mean,omitempty"`

	// Engagement results are a macro-level view that explain an email recipient’s propensity to engage.
	// https://documentation.mailgun.com/docs/inboxready/mailgun-validate/validate_engagement/
	Engagement *EngagementData `json:"engagement,omitempty"`

	RootAddress string `json:"root_address,omitempty"`
}

type EngagementData struct {
	Engaging bool   `json:"engaging"`
	IsBot    bool   `json:"is_bot"`
	Behavior string `json:"behavior,omitempty"`
}
