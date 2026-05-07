package mtypes

// StoredMessage structures contain the (parsed) message content for an email
// sent to a Mailgun account.
//
// The MessageHeaders field is special, in that it's formatted as a slice of pairs.
// Each pair consists of a name [0] and value [1].  Array notation is used instead of a map
// because that's how it's sent over the wire, and it's how encoding/json expects this field
// to be.
type StoredMessage struct {
	Recipients        string             `json:"recipients"`
	Sender            string             `json:"sender"`
	From              string             `json:"from"`
	Subject           string             `json:"subject"`
	BodyPlain         string             `json:"body-plain"`
	StrippedText      string             `json:"stripped-text"`
	StrippedSignature string             `json:"stripped-signature"`
	BodyHtml          string             `json:"body-html"`
	StrippedHtml      string             `json:"stripped-html"`
	Attachments       []StoredAttachment `json:"attachments"`
	MessageUrl        string             `json:"message-url"`
	ContentIDMap      map[string]struct {
		URL         string `json:"url"`
		ContentType string `json:"content-type"`
		Name        string `json:"name"`
		Size        int64  `json:"size"`
	} `json:"content-id-map"`
	MessageHeaders [][]string `json:"message-headers"`
}

// StoredAttachment structures contain information on an attachment associated with a stored message.
type StoredAttachment struct {
	Size        int    `json:"size"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	ContentType string `json:"content-type"`
}

type StoredMessageRaw struct {
	Recipients string `json:"recipients"`
	Sender     string `json:"sender"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	BodyMime   string `json:"body-mime"`
}
