package mtypes

// Use these to specify a spam action when creating a new domain.
const (
	// SpamActionTag tags the received message with headers providing a measure of its spamness.
	SpamActionTag = SpamAction("tag")
	// SpamActionDisabled prevents Mailgun from taking any action on what it perceives to be spam.
	SpamActionDisabled = SpamAction("disabled")
	// SpamActionDelete instructs Mailgun to just block or delete the message all-together.
	SpamActionDelete = SpamAction("delete")
)

type SpamAction string

type ListDomainsResponse struct {
	// is -1 if Next() or First() have not been called
	TotalCount int      `json:"total_count"`
	Items      []Domain `json:"items"`
}

// A Domain structure holds information about a domain used when sending mail.
type Domain struct {
	CreatedAt                  RFC2822Time `json:"created_at"`
	ID                         string      `json:"id"`
	IsDisabled                 bool        `json:"is_disabled"`
	Name                       string      `json:"name"`
	RequireTLS                 bool        `json:"require_tls"`
	SkipVerification           bool        `json:"skip_verification"`
	SMTPLogin                  string      `json:"smtp_login"`
	SMTPPassword               string      `json:"smtp_password,omitempty"`
	SpamAction                 SpamAction  `json:"spam_action"`
	State                      string      `json:"state"`
	Type                       string      `json:"type"`
	TrackingHost               string      `json:"tracking_host,omitempty"`
	UseAutomaticSenderSecurity bool        `json:"use_automatic_sender_security"`
	WebPrefix                  string      `json:"web_prefix"`
	WebScheme                  string      `json:"web_scheme"`
	Wildcard                   bool        `json:"wildcard"`
}

// DNSRecord structures describe intended records to properly configure your domain for use with Mailgun.
// Note that Mailgun does not host DNS records.
type DNSRecord struct {
	Active     bool     `json:"is_active"`
	Cached     []string `json:"cached"`
	Name       string   `json:"name,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	RecordType string   `json:"record_type"`
	Valid      string   `json:"valid"`
	Value      string   `json:"value"`
}

type GetDomainResponse struct {
	Domain                     Domain      `json:"domain"`
	ReceivingDNSRecords        []DNSRecord `json:"receiving_dns_records"`
	SendingDNSRecords          []DNSRecord `json:"sending_dns_records"`
	UseAutomaticSenderSecurity bool        `json:"use_automatic_sender_security"`
}
