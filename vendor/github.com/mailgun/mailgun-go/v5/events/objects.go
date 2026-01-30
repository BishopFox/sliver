package events

type ClientInfo struct {
	AcceptLanguage string `json:"accept-language"`
	ClientName     string `json:"client-name"`
	ClientOS       string `json:"client-os"`
	ClientType     string `json:"client-type"`
	DeviceType     string `json:"device-type"`
	IP             string `json:"ip"`
	UserAgent      string `json:"user-agent"`
	Bot            string `json:"bot"`
}

type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Region  string `json:"region"`
}

type MailingList struct {
	Address string `json:"address"`
	ListID  string `json:"list-id"`
	SID     string `json:"sid"`
}

type Message struct {
	Headers     MessageHeaders `json:"headers"`
	Attachments []Attachment   `json:"attachments"`
	Recipients  []string       `json:"recipients"`
	Size        int            `json:"size"`
}

type Envelope struct {
	MailFrom    string `json:"mail-from"`
	Sender      string `json:"sender"`
	Transport   string `json:"transport"`
	Targets     string `json:"targets"`
	SendingHost string `json:"sending-host"`
	SendingIP   string `json:"sending-ip"`
}

type Storage struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type Flags struct {
	IsAuthenticated bool `json:"is-authenticated"`
	IsBig           bool `json:"is-big"`
	IsSystemTest    bool `json:"is-system-test"`
	IsTestMode      bool `json:"is-test-mode"`
	IsDelayedBounce bool `json:"is-delayed-bounce"`
}

type Attachment struct {
	FileName    string `json:"filename"`
	ContentType string `json:"content-type"`
	Size        int    `json:"size"`
}

type MessageHeaders struct {
	To        string `json:"to"`
	MessageID string `json:"message-id"`
	From      string `json:"from"`
	Subject   string `json:"subject"`
}

type Campaign struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeliveryStatus struct {
	Code                int     `json:"code"`
	AttemptNo           int     `json:"attempt-no"`
	Description         string  `json:"description,omitempty"`
	Message             string  `json:"message"`
	SessionSeconds      float64 `json:"session-seconds"`
	EnhancedCode        string  `json:"enhanced-code,omitempty"`
	MxHost              string  `json:"mx-host,omitempty"`
	RetrySeconds        int     `json:"retry-seconds,omitempty"`
	CertificateVerified *bool   `json:"certificate-verified,omitempty"`
	TLS                 *bool   `json:"tls,omitempty"`
	Utf8                *bool   `json:"utf8,omitempty"`
}
