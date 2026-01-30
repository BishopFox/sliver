package pushover

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var deviceNameRegexp *regexp.Regexp

func init() {
	deviceNameRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]{1,25}$`)
}

// Message represents a pushover message.
type Message struct {
	// Required
	Message string

	// Optional
	Title       string
	Priority    int
	URL         string
	URLTitle    string
	Timestamp   int64
	Retry       time.Duration
	Expire      time.Duration
	CallbackURL string
	DeviceName  string
	Sound       string
	HTML        bool
	Monospace   bool
	TTL         time.Duration

	// attachment
	attachment io.Reader
}

// NewMessage returns a simple new message.
func NewMessage(message string) *Message {
	return &Message{Message: message}
}

// NewMessageWithTitle returns a simple new message with a title.
func NewMessageWithTitle(message, title string) *Message {
	return &Message{Message: message, Title: title}
}

// AddAttachment adds an attachment to the message it's programmer's
// responsibility to close the reader.
func (m *Message) AddAttachment(attachment io.Reader) error {
	m.attachment = attachment
	return nil
}

// Validate the message values.
func (m *Message) validate() error {
	// Message should no be empty
	if m.Message == "" {
		return ErrMessageEmpty
	}

	// Validate message length
	if utf8.RuneCountInString(m.Message) > MessageMaxLength {
		return ErrMessageTooLong
	}

	// Validate Title field length
	if utf8.RuneCountInString(m.Title) > MessageTitleMaxLength {
		return ErrMessageTitleTooLong
	}

	// Validate URL field
	if utf8.RuneCountInString(m.URL) > MessageURLMaxLength {
		return ErrMessageURLTooLong
	}

	// Validate URL title field
	if utf8.RuneCountInString(m.URLTitle) > MessageURLTitleMaxLength {
		return ErrMessageURLTitleTooLong
	}

	// URLTitle should not be set with an empty URL
	if m.URL == "" && m.URLTitle != "" {
		return ErrEmptyURL
	}

	// Validate priorities
	if m.Priority > PriorityEmergency || m.Priority < PriorityLowest {
		return ErrInvalidPriority
	}

	// Validate emergency priority
	if m.Priority == PriorityEmergency {
		if m.Retry == 0 || m.Expire == 0 {
			return ErrMissingEmergencyParameter
		}
	}

	// Test device name
	if m.DeviceName != "" {
		// Accept comma separated device names
		devices := strings.Split(m.DeviceName, ",")
		for _, d := range devices {
			if !deviceNameRegexp.MatchString(d) {
				return ErrInvalidDeviceName
			}
		}
	}

	return nil
}

// Return a map filled with the relevant data.
func (m *Message) toMap(pToken, rToken string) map[string]string {
	ret := map[string]string{
		"token":    pToken,
		"user":     rToken,
		"message":  m.Message,
		"priority": strconv.Itoa(m.Priority),
	}

	if m.Title != "" {
		ret["title"] = m.Title
	}

	if m.URL != "" {
		ret["url"] = m.URL
	}

	if m.URLTitle != "" {
		ret["url_title"] = m.URLTitle
	}

	if m.Sound != "" {
		ret["sound"] = m.Sound
	}

	if m.DeviceName != "" {
		ret["device"] = m.DeviceName
	}

	if m.Timestamp != 0 {
		ret["timestamp"] = strconv.FormatInt(m.Timestamp, 10)
	}

	if m.HTML {
		ret["html"] = "1"
	}

	if m.Monospace {
		ret["monospace"] = "1"
	}

	if m.Priority == PriorityEmergency {
		ret["retry"] = strconv.FormatFloat(m.Retry.Seconds(), 'f', -1, 64)
		ret["expire"] = strconv.FormatFloat(m.Expire.Seconds(), 'f', -1, 64)
		if m.CallbackURL != "" {
			ret["callback"] = m.CallbackURL
		}
	}

	if m.TTL != 0 {
		ret["ttl"] = strconv.FormatFloat(m.TTL.Seconds(), 'f', -1, 64)
	}

	return ret
}

// Send sends the message using the pushover and the recipient tokens.
func (m *Message) send(pToken, rToken string) (*Response, error) {
	url := fmt.Sprintf("%s/messages.json", APIEndpoint)

	var f func(string, string, string) (*http.Request, error)
	if m.attachment == nil {
		// Use a URL-encoded request if there's no need to attach files
		f = m.urlEncodedRequest
	} else {
		// Use a multipart request if a file should be sent
		f = m.multipartRequest
	}

	// Post the from and check the headers of the response
	req, err := f(pToken, rToken, url)
	if err != nil {
		return nil, err
	}

	resp := &Response{}
	if err := do(req, resp, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// multipartRequest returns a new multipart POST request with a file attached.
func (m *Message) multipartRequest(pToken, rToken, url string) (*http.Request, error) {
	body := &bytes.Buffer{}

	if m.attachment == nil {
		return nil, ErrMissingAttachment
	}

	// Write the body as multipart form data
	w := multipart.NewWriter(body)

	// Write the file in the body
	fw, err := w.CreateFormFile("attachment", "attachment")
	if err != nil {
		return nil, err
	}

	written, err := io.Copy(fw, m.attachment)
	if err != nil {
		return nil, err
	}

	if written > MessageMaxAttachmentByte {
		return nil, ErrMessageAttachmentTooLarge
	}

	// Handle params
	for k, v := range m.toMap(pToken, rToken) {
		if err := w.WriteField(k, v); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

// urlEncodedRequest returns a new url encoded request.
func (m *Message) urlEncodedRequest(pToken, rToken, endpoint string) (*http.Request, error) {
	return newURLEncodedRequest("POST", endpoint, m.toMap(pToken, rToken))
}
