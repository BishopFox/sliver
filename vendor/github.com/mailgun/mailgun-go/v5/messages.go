package mailgun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// MaxNumberOfRecipients represents the largest batch of recipients that Mailgun can support in a single API call.
// This figure includes To:, Cc:, Bcc:, etc. recipients.
const MaxNumberOfRecipients = 1000

// MaxNumberOfTags represents the maximum number of tags that can be added for a message
const MaxNumberOfTags = 10

// CommonMessage contains both the message text and the envelope for an e-mail message.
type CommonMessage struct {
	domain                   string
	to                       []string
	tags                     []string
	dkim                     *bool
	secondaryDKIM            string
	secondaryDKIMPublic      string
	deliveryTime             time.Time
	stoPeriod                string
	attachments              []string
	readerAttachments        []ReaderAttachment
	inlines                  []string
	readerInlines            []ReaderAttachment
	bufferAttachments        []BufferAttachment
	nativeSend               bool
	testMode                 bool
	tracking                 *bool
	trackingClicks           *string
	trackingOpens            *bool
	trackingPixelLocationTop *string
	headers                  map[string]string
	variables                map[string]string
	options                  map[string]string
	templateVariables        map[string]any
	recipientVariables       map[string]map[string]any
	templateVersionTag       string
	templateRenderText       bool
	requireTLS               bool
	skipVerification         bool
}

type ReaderAttachment struct {
	Filename   string
	ReadCloser io.ReadCloser
}

type BufferAttachment struct {
	Filename string
	Buffer   []byte
}

// PlainMessage contains fields relevant to plain API-synthesized messages.
// You're expected to use various setters to set most of these attributes,
// although from, subject, and text are set when the message is created with
// NewMessage.
type PlainMessage struct {
	CommonMessage

	from     string
	cc       []string
	bcc      []string
	subject  string
	text     string
	html     string
	ampHtml  string
	template string
}

func (m *PlainMessage) From() string {
	return m.from
}

func (m *PlainMessage) CC() []string {
	return m.cc
}

func (m *PlainMessage) BCC() []string {
	return m.bcc
}

func (m *PlainMessage) Subject() string {
	return m.subject
}

func (m *PlainMessage) Text() string {
	return m.text
}

func (m *PlainMessage) HTML() string {
	return m.html
}

func (m *PlainMessage) AmpHTML() string {
	return m.ampHtml
}

func (m *PlainMessage) Template() string {
	return m.template
}

// MimeMessage contains fields relevant to pre-packaged MIME messages.
type MimeMessage struct {
	CommonMessage

	body io.ReadCloser
}

// TrackingOptions contains fields relevant to tracking.
type TrackingOptions struct {
	Tracking                 bool
	TrackingClicks           string
	TrackingOpens            bool
	TrackingPixelLocationTop string
}

// The Specific abstracts the common characteristics between plain text and MIME messages.
//
// TODO(v6): remove setters, as they are not used by Send(),
// and merge into Message interface.
type Specific interface {
	// AddCC appends a receiver to the carbon-copy header of a message.
	AddCC(string)

	// AddBCC appends a receiver to the blind-carbon-copy header of a message.
	AddBCC(string)

	// SetHTML If you're sending a message that isn't already MIME encoded, it will arrange to bundle
	// an HTML representation of your message in addition to your plain-text body.
	SetHTML(string)

	// SetAmpHTML If you're sending a message that isn't already MIME encoded, it will arrange to bundle
	// an AMP-For-Email representation of your message in addition to your HTML & plain-text content.
	SetAmpHTML(string)

	// AddValues invoked by Send() to add message-type-specific MIME headers for the API call
	// to Mailgun.
	//
	// TODO(v6): Make a `Params() map[string][]string` getter instead:
	AddValues(*FormDataPayload)

	// IsValid yields true if and only if the message is valid enough for sending
	// through the API.
	IsValid() bool

	// Endpoint tells Send() which endpoint to use to submit the API call.
	Endpoint() string

	// RecipientCount returns the total number of recipients for the message.
	// This includes To:, Cc:, and Bcc: fields.
	//
	// NOTE: At present, this method is reliable only for non-MIME messages, as the
	// Bcc: and Cc: fields are easily accessible.
	// For MIME messages, only the To: field is considered.
	// A fix for this issue is planned for a future release.
	// For now, MIME messages are always assumed to have 10 recipients between Cc: and Bcc: fields.
	// If your MIME messages have more than 10 non-To: field recipients,
	// you may find that some recipients will not receive your e-mail.
	// It's perfectly OK, of course, for a MIME message to not have any Cc: or Bcc: recipients.
	RecipientCount() int

	// SetTemplate sets the name of a template stored via the template API.
	// See https://documentation.mailgun.com/docs/mailgun/user-manual/sending-messages/#templates
	SetTemplate(string)
}

// NewMessage returns a new e-mail message with the simplest envelop needed to send.
//
// Supports arbitrary-sized recipient lists by
// automatically sending mail in batches of up to MaxNumberOfRecipients.
//
// To support batch sending, do not provide `to` at this point.
// You can do this explicitly, or implicitly, as follows:
//
//	// Note absence of `to` parameter(s)!
//	m := NewMessage("example.com", "me@example.com", "Help save our planet", "Hello world!")
//
// Note that you'll need to invoke the AddRecipientAndVariables or AddRecipient method
// before sending, though.
func NewMessage(domain, from, subject, text string, to ...string) *PlainMessage {
	return &PlainMessage{
		CommonMessage: CommonMessage{
			domain: domain,
			to:     to,
		},

		from:    from,
		subject: subject,
		text:    text,
	}
}

// NewMIMEMessage creates a new MIME message. These messages are largely canned;
// you do not need to invoke setters to set message-related headers.
// However, you do still need to call setters for Mailgun-specific settings.
//
// Supports arbitrary-sized recipient lists by
// automatically sending mail in batches of up to MaxNumberOfRecipients.
//
// To support batch sending, do not provide `to` at this point.
// You can do this explicitly, or implicitly, as follows:
//
//	// Note absence of `to` parameter(s)!
//	m := NewMIMEMessage(domain, body)
//
// Note that you'll need to invoke the AddRecipientAndVariables or AddRecipient method
// before sending, though.
func NewMIMEMessage(domain string, body io.ReadCloser, to ...string) *MimeMessage {
	return &MimeMessage{
		CommonMessage: CommonMessage{
			domain: domain,
			to:     to,
		},
		body: body,
	}
}

func (m *CommonMessage) Domain() string {
	return m.domain
}

func (m *CommonMessage) To() []string {
	return m.to
}

func (m *CommonMessage) Tags() []string {
	return m.tags
}

func (m *CommonMessage) DKIM() *bool {
	return m.dkim
}

func (m *CommonMessage) SecondaryDKIM() string {
	return m.secondaryDKIM
}

func (m *CommonMessage) SecondaryDKIMPublic() string {
	return m.secondaryDKIMPublic
}

func (m *CommonMessage) DeliveryTime() time.Time {
	return m.deliveryTime
}

func (m *CommonMessage) STOPeriod() string {
	return m.stoPeriod
}

func (m *CommonMessage) Attachments() []string {
	return m.attachments
}

func (m *CommonMessage) ReaderAttachments() []ReaderAttachment {
	return m.readerAttachments
}

func (m *CommonMessage) Inlines() []string {
	return m.inlines
}

func (m *CommonMessage) ReaderInlines() []ReaderAttachment {
	return m.readerInlines
}

func (m *CommonMessage) BufferAttachments() []BufferAttachment {
	return m.bufferAttachments
}

func (m *CommonMessage) NativeSend() bool {
	return m.nativeSend
}

func (m *CommonMessage) TestMode() bool {
	return m.testMode
}

func (m *CommonMessage) Tracking() *bool {
	return m.tracking
}

func (m *CommonMessage) TrackingClicks() *string {
	return m.trackingClicks
}

func (m *CommonMessage) TrackingOpens() *bool {
	return m.trackingOpens
}

func (m *CommonMessage) TrackingPixelLocationTop() *string {
	return m.trackingPixelLocationTop
}

func (m *CommonMessage) Variables() map[string]string {
	return m.variables
}

func (m *CommonMessage) TemplateVariables() map[string]any {
	return m.templateVariables
}

func (m *CommonMessage) RecipientVariables() map[string]map[string]any {
	return m.recipientVariables
}

func (m *CommonMessage) TemplateVersionTag() string {
	return m.templateVersionTag
}

func (m *CommonMessage) TemplateRenderText() bool {
	return m.templateRenderText
}

func (m *CommonMessage) RequireTLS() bool {
	return m.requireTLS
}

func (m *CommonMessage) SkipVerification() bool {
	return m.skipVerification
}

// AddReaderAttachment arranges to send a file along with the e-mail message.
// File contents are read from an io.ReadCloser.
// The filename parameter is the resulting filename of the attachment.
// The readCloser parameter is the io.ReadCloser that reads the actual bytes to be used
// as the contents of the attached file.
func (m *CommonMessage) AddReaderAttachment(filename string, readCloser io.ReadCloser) {
	ra := ReaderAttachment{Filename: filename, ReadCloser: readCloser}
	m.readerAttachments = append(m.readerAttachments, ra)
}

// AddBufferAttachment arranges to send a file along with the e-mail message.
// File contents are read from the []byte array provided
// The filename parameter is the resulting filename of the attachment.
// The buffer parameter is the []byte array which contains the actual bytes to be used
// as the contents of the attached file.
func (m *CommonMessage) AddBufferAttachment(filename string, buffer []byte) {
	ba := BufferAttachment{Filename: filename, Buffer: buffer}
	m.bufferAttachments = append(m.bufferAttachments, ba)
}

// AddAttachment arranges to send a file from the filesystem along with the e-mail message.
// The attachment parameter is a filename, which must refer to a file which actually resides
// in the local filesystem.
func (m *CommonMessage) AddAttachment(attachment string) {
	m.attachments = append(m.attachments, attachment)
}

// AddReaderInline arranges to send a file along with the e-mail message.
// File contents are read from an io.ReadCloser.
// The filename parameter is the resulting filename of the attachment.
// The readCloser parameter is the io.ReadCloser that reads the actual bytes to be used
// as the contents of the attached file.
func (m *CommonMessage) AddReaderInline(filename string, readCloser io.ReadCloser) {
	ra := ReaderAttachment{Filename: filename, ReadCloser: readCloser}
	m.readerInlines = append(m.readerInlines, ra)
}

// AddInline arranges to send a file along with the e-mail message, but does so
// in a way that its data remains "inline" with the rest of the message.  This
// can be used to send image or font data along with an HTML-encoded message body.
// The attachment parameter is a filename, which must refer to a file which actually resides
// in the local filesystem.
func (m *CommonMessage) AddInline(inline string) {
	m.inlines = append(m.inlines, inline)
}

// AddRecipient appends a receiver to the To: header of a message.
// It will return an error if the limit of recipients has been exceeded for this message
func (m *PlainMessage) AddRecipient(recipient string) error {
	return m.AddRecipientAndVariables(recipient, nil)
}

// AddRecipientAndVariables appends a receiver to the To: header of a message,
// and as well attaches a set of variables relevant for this recipient.
// It will return an error if the limit of recipients has been exceeded for this message
func (m *PlainMessage) AddRecipientAndVariables(r string, vars map[string]any) error {
	if m.RecipientCount() >= MaxNumberOfRecipients {
		return fmt.Errorf("recipient limit exceeded (max %d)", MaxNumberOfRecipients)
	}
	m.to = append(m.to, r)
	if vars != nil {
		if m.recipientVariables == nil {
			m.recipientVariables = make(map[string]map[string]any)
		}
		m.recipientVariables[r] = vars
	}
	return nil
}

func (m *MimeMessage) AddRecipient(recipient string) error {
	if m.RecipientCount() >= MaxNumberOfRecipients {
		return fmt.Errorf("recipient limit exceeded (max %d)", MaxNumberOfRecipients)
	}
	m.to = append(m.to, recipient)

	return nil
}

func (m *PlainMessage) RecipientCount() int {
	return len(m.To()) + len(m.BCC()) + len(m.CC())
}

func (m *MimeMessage) RecipientCount() int {
	return 10 + len(m.To())
}

// SetReplyTo sets the receiver who should receive replies
func (m *CommonMessage) SetReplyTo(recipient string) {
	m.AddHeader("Reply-To", recipient)
}

func (m *PlainMessage) AddCC(r string) {
	m.cc = append(m.CC(), r)
}

func (*MimeMessage) AddCC(_ string) {}

func (m *PlainMessage) AddBCC(r string) {
	m.bcc = append(m.BCC(), r)
}

func (*MimeMessage) AddBCC(_ string) {}

func (m *PlainMessage) SetHTML(h string) {
	m.html = h
}

func (*MimeMessage) SetHTML(_ string) {}

func (m *PlainMessage) SetAmpHTML(h string) {
	m.ampHtml = h
}

func (*MimeMessage) SetAmpHTML(_ string) {}

// AddTag attaches tags to the message.  Tags are useful for metrics gathering and event tracking purposes.
// Refer to the Mailgun documentation for further details.
func (m *CommonMessage) AddTag(tag ...string) error {
	if len(m.Tags()) >= MaxNumberOfTags {
		return fmt.Errorf("cannot add any new tags. Message tag limit (%d) reached", MaxNumberOfTags)
	}

	m.tags = append(m.Tags(), tag...)
	return nil
}

func (m *PlainMessage) SetTemplate(t string) {
	m.template = t
}

func (*MimeMessage) SetTemplate(_ string) {}

// SetDKIM arranges to send the o:dkim header with the message, and sets its value accordingly.
// Refer to the Mailgun documentation for more information.
func (m *CommonMessage) SetDKIM(dkim bool) {
	m.dkim = &dkim
}

// SetSecondaryDKIM specifies a second domain key to sign the email with.
// The value is formatted as signing_domain/selector, e.g. example.com/s1.
func (m *CommonMessage) SetSecondaryDKIM(s string) {
	m.secondaryDKIM = s
}

// SetSecondaryDKIMPublic specifies an alias of the domain key specified in o:secondary-dkim.
// Also formatted as public_signing_domain/selector.
// o:secondary-dkim option must also be provided.
func (m *CommonMessage) SetSecondaryDKIMPublic(s string) {
	m.secondaryDKIMPublic = s
}

// EnableNativeSend allows the return path to match the address in the CommonMessage.Headers.From:
// field when sending from Mailgun rather than the usual bounce+ address in the return path.
func (m *CommonMessage) EnableNativeSend() {
	m.nativeSend = true
}

// EnableTestMode allows submittal of a message, such that it will be discarded by Mailgun.
// This facilitates testing client-side software without actually consuming e-mail resources.
func (m *CommonMessage) EnableTestMode() {
	m.testMode = true
}

// SetDeliveryTime schedules the message for transmission at the indicated time.
// Pass nil to remove any installed schedule.
// Refer to the Mailgun documentation for more information.
func (m *CommonMessage) SetDeliveryTime(dt time.Time) {
	m.deliveryTime = dt
}

// SetSTOPeriod toggles Send Time Optimization (STO) on a per-message basis.
// String should be set to the number of hours in [0-9]+h format,
// with the minimum being 24h and the maximum being 72h.
// Refer to the Mailgun documentation for more information.
func (m *CommonMessage) SetSTOPeriod(stoPeriod string) error {
	validPattern := `^([2-6][4-9]|[3-6][0-9]|7[0-2])h$`
	// TODO(vtopc): regexp.Compile, which is called by regexp.MatchString, is a heave operation, move into global variable
	// or just parse using time.ParseDuration().
	match, err := regexp.MatchString(validPattern, stoPeriod)
	if err != nil {
		return err
	}

	if !match {
		return errors.New("STO period is invalid. Valid range is 24h to 72h")
	}

	m.stoPeriod = stoPeriod
	return nil
}

// SetTracking sets the o:tracking message parameter to adjust, on a message-by-message basis,
// whether or not Mailgun will rewrite URLs to facilitate event tracking.
// Events tracked includes opens, clicks, unsubscribes, etc.
// Note: simply calling this method ensures that the o:tracking header is passed in with the message.
// Its yes/no setting is determined by the call's parameter.
// Note that this header is not passed on to the final recipient(s).
// Refer to the Mailgun documentation for more information.
func (m *CommonMessage) SetTracking(tracking bool) {
	m.tracking = &tracking
}

// SetTrackingClicks information is found in the Mailgun documentation.
func (m *CommonMessage) SetTrackingClicks(trackingClicks bool) {
	m.trackingClicks = ptr(yesNo(trackingClicks))
}

// SetTrackingOptions sets o:tracking, o:tracking-clicks, o:tracking-pixel-location-top, and o:tracking-opens at once.
func (m *CommonMessage) SetTrackingOptions(options *TrackingOptions) {
	m.tracking = &options.Tracking
	m.trackingClicks = &options.TrackingClicks
	m.trackingOpens = &options.TrackingOpens

	if options.TrackingPixelLocationTop != "" {
		m.trackingPixelLocationTop = &options.TrackingPixelLocationTop
	}
}

// SetRequireTLS information is found in the Mailgun documentation.
func (m *CommonMessage) SetRequireTLS(b bool) {
	m.requireTLS = b
}

// SetSkipVerification information is found in the Mailgun documentation.
func (m *CommonMessage) SetSkipVerification(b bool) {
	m.skipVerification = b
}

// SetTrackingOpens information is found in the Mailgun documentation.
func (m *CommonMessage) SetTrackingOpens(trackingOpens bool) {
	m.trackingOpens = &trackingOpens
}

// SetTemplateVersion information is found in the Mailgun documentation.
func (m *CommonMessage) SetTemplateVersion(tag string) {
	m.templateVersionTag = tag
}

// SetTemplateRenderText information is found in the Mailgun documentation.
func (m *CommonMessage) SetTemplateRenderText(render bool) {
	m.templateRenderText = render
}

// AddHeader allows you to send custom MIME headers with the message.
func (m *CommonMessage) AddHeader(header, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[header] = value
}

// AddVariable lets you associate a set of variables with messages you send,
// which Mailgun can use to, in essence, complete form-mail.
// Refer to the Mailgun documentation for more information.
func (m *CommonMessage) AddVariable(variable string, value any) error {
	if m.variables == nil {
		m.variables = make(map[string]string)
	}

	j, err := json.Marshal(value)
	if err != nil {
		return err
	}

	encoded := string(j)
	v, err := strconv.Unquote(encoded)
	if err != nil {
		v = encoded
	}

	m.variables[variable] = v
	return nil
}

// AddTemplateVariable adds a template variable to the map of template variables, replacing the variable if it is already there.
// This is used for server-side message templates and can nest arbitrary values. At send time, the resulting map will be converted into
// a JSON string and sent as a header in the X-Mailgun-Variables header.
func (m *CommonMessage) AddTemplateVariable(variable string, value any) error {
	if m.templateVariables == nil {
		m.templateVariables = make(map[string]any)
	}
	m.templateVariables[variable] = value
	return nil
}

// AddDomain allows you to use a separate domain for the type of messages you are sending.
func (m *CommonMessage) AddDomain(domain string) {
	m.domain = domain
}

// Headers retrieve the HTTP headers associated with this message.
func (m *CommonMessage) Headers() map[string]string {
	return m.headers
}

// AddOption add additional "o:" option.
// The key parameter should NOT include the "o:" prefix.
func (m *CommonMessage) AddOption(key, value string) {
	if m.options == nil {
		m.options = make(map[string]string)
	}

	m.options[key] = value
}

// Options retrieve additional options("o:") associated with this message.
func (m *CommonMessage) Options() map[string]string {
	return m.options
}

// ErrInvalidMessage is returned by `Send()` when the `mailgun.CommonMessage` struct is incomplete
var ErrInvalidMessage = errors.New("message not valid")

type Message interface {
	Specific

	Domain() string
	To() []string
	Attachments() []string
	ReaderAttachments() []ReaderAttachment
	Inlines() []string
	ReaderInlines() []ReaderAttachment
	BufferAttachments() []BufferAttachment
	Headers() map[string]string
	Options() map[string]string
	Variables() map[string]string
	TemplateVariables() map[string]any
	RecipientVariables() map[string]map[string]any
	TemplateVersionTag() string
	TemplateRenderText() bool

	// TODO(v6): remove next methods and use Options() getter to get these values

	Tags() []string
	DKIM() *bool
	SecondaryDKIM() string
	SecondaryDKIMPublic() string
	DeliveryTime() time.Time
	STOPeriod() string
	NativeSend() bool
	TestMode() bool
	Tracking() *bool
	TrackingClicks() *string
	TrackingOpens() *bool
	TrackingPixelLocationTop() *string
	RequireTLS() bool
	SkipVerification() bool
}

// Send attempts to queue a message (see PlainMessage, MimeMessage and its methods) for delivery.
// It returns the Mailgun server response, which consists of two components:
//   - A human-readable status message, typically "Queued. Thank you."
//   - A Message ID, which is the id used to track the queued message. The message id is useful
//     when contacting support to report an issue with a specific message or to relate a
//     delivered, accepted or failed event back to a specific message.
//
// The status and message ID are only returned if no error occurred.
//
// Returned error can be wrapped internal and standard
// Go errors like `url.Error`. The error can also be of type
// mailgun.UnexpectedResponseError which contains the error returned by the mailgun API.
//
// See the public mailgun documentation for all possible return codes and error messages
func (mg *Client) Send(ctx context.Context, m Message) (mtypes.SendMessageResponse, error) {
	var response mtypes.SendMessageResponse

	if m.Domain() == "" {
		err := errors.New("you must provide a valid domain before calling Send()")
		return response, err
	}

	invalidChars := ":&'@(),!?#;%+=<>"
	if i := strings.ContainsAny(m.Domain(), invalidChars); i {
		err := fmt.Errorf("you called Send() with a domain that contains invalid characters")
		return response, err
	}

	if mg.apiKey == "" {
		err := errors.New("you must provide a valid api-key before calling Send()")
		return response, err
	}

	if !isValid(m) {
		err := ErrInvalidMessage
		return response, err
	}

	if m.STOPeriod() != "" && m.RecipientCount() > 1 {
		err := errors.New("STO can only be used on a per-message basis")
		return response, err
	}
	payload := NewFormDataPayload()

	m.AddValues(payload)

	// TODO: make (CommonMessage).AddValues()?
	err := addMessageValues(payload, m)
	if err != nil {
		return response, err
	}

	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, m.Endpoint(), m.Domain()))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	// Override any HTTP headers if provided
	for k, v := range mg.overrideHeaders {
		r.addHeader(k, v)
	}

	err = postResponseFromJSON(ctx, r, payload, &response)

	return response, err
}

func addMessageValues(dst *FormDataPayload, src Message) error {
	addMessageOptions(dst, src)
	addMessageHeaders(dst, src)

	err := addMessageVariables(dst, src)
	if err != nil {
		return err
	}

	addMessageAttachment(dst, src)

	return nil
}

func addMessageOptions(dst *FormDataPayload, src Message) {
	for _, to := range src.To() {
		dst.addValue("to", to)
	}

	for _, tag := range src.Tags() {
		dst.addValue("o:tag", tag)
	}
	if src.DKIM() != nil {
		dst.addValue("o:dkim", yesNo(*src.DKIM()))
	}
	if src.SecondaryDKIM() != "" {
		dst.addValue("o:secondary-dkim", src.SecondaryDKIM())
	}
	if src.SecondaryDKIMPublic() != "" {
		dst.addValue("o:secondary-dkim-public", src.SecondaryDKIMPublic())
	}
	if !src.DeliveryTime().IsZero() {
		dst.addValue("o:deliverytime", formatMailgunTime(src.DeliveryTime()))
	}
	if src.STOPeriod() != "" {
		dst.addValue("o:deliverytime-optimize-period", src.STOPeriod())
	}
	if src.NativeSend() {
		dst.addValue("o:native-send", "yes")
	}
	if src.TestMode() {
		dst.addValue("o:testmode", "yes")
	}
	if src.Tracking() != nil {
		dst.addValue("o:tracking", yesNo(*src.Tracking()))
	}
	if src.TrackingClicks() != nil {
		dst.addValue("o:tracking-clicks", *src.TrackingClicks())
	}
	if src.TrackingOpens() != nil {
		dst.addValue("o:tracking-opens", yesNo(*src.TrackingOpens()))
	}
	if src.TrackingPixelLocationTop() != nil {
		dst.addValue("o:tracking-pixel-location-top", *src.TrackingPixelLocationTop())
	}
	if src.RequireTLS() {
		dst.addValue("o:require-tls", trueFalse(src.RequireTLS()))
	}
	if src.SkipVerification() {
		dst.addValue("o:skip-verification", trueFalse(src.SkipVerification()))
	}

	// Add any additional options
	for key, value := range src.Options() {
		dst.addValue("o:"+key, value)
	}

	if src.TemplateVersionTag() != "" {
		dst.addValue("t:version", src.TemplateVersionTag())
	}
	if src.TemplateRenderText() {
		dst.addValue("t:text", yesNo(src.TemplateRenderText()))
	}
}

func addMessageHeaders(dst *FormDataPayload, src Message) {
	if src.Headers() != nil {
		for header, value := range src.Headers() {
			dst.addValue("h:"+header, value)
		}
	}
}

func addMessageVariables(dst *FormDataPayload, src Message) error {
	if src.Variables() != nil {
		for variable, value := range src.Variables() {
			dst.addValue("v:"+variable, value)
		}
	}
	if src.TemplateVariables() != nil {
		variableString, err := json.Marshal(src.TemplateVariables())
		if err == nil {
			// the map was marshaled as JSON so add it
			dst.addValue("h:X-Mailgun-Variables", string(variableString))
		}
	}
	if src.RecipientVariables() != nil {
		j, err := json.Marshal(src.RecipientVariables())
		if err != nil {
			return err
		}
		dst.addValue("recipient-variables", string(j))
	}

	return nil
}

func addMessageAttachment(dst *FormDataPayload, src Message) {
	if src.Attachments() != nil {
		for _, attachment := range src.Attachments() {
			dst.addFile("attachment", attachment)
		}
	}
	if src.ReaderAttachments() != nil {
		for _, readerAttachment := range src.ReaderAttachments() {
			dst.addReadCloser("attachment", readerAttachment.Filename, readerAttachment.ReadCloser)
		}
	}
	if src.BufferAttachments() != nil {
		for _, bufferAttachment := range src.BufferAttachments() {
			dst.addBuffer("attachment", bufferAttachment.Filename, bufferAttachment.Buffer)
		}
	}
	if src.Inlines() != nil {
		for _, inline := range src.Inlines() {
			dst.addFile("inline", inline)
		}
	}
	if src.ReaderInlines() != nil {
		for _, readerAttachment := range src.ReaderInlines() {
			dst.addReadCloser("inline", readerAttachment.Filename, readerAttachment.ReadCloser)
		}
	}
}

func (m *PlainMessage) AddValues(p *FormDataPayload) {
	p.addValue("from", m.From())
	p.addValue("subject", m.Subject())
	p.addValue("text", m.Text())
	for _, cc := range m.CC() {
		p.addValue("cc", cc)
	}
	for _, bcc := range m.BCC() {
		p.addValue("bcc", bcc)
	}
	if m.HTML() != "" {
		p.addValue("html", m.HTML())
	}
	if m.Template() != "" {
		p.addValue("template", m.Template())
	}
	if m.AmpHTML() != "" {
		p.addValue("amp-html", m.AmpHTML())
	}
}

func (m *MimeMessage) AddValues(p *FormDataPayload) {
	p.addReadCloser("message", "message.mime", m.body)
}

func (*PlainMessage) Endpoint() string {
	return messagesEndpoint
}

func (*MimeMessage) Endpoint() string {
	return mimeMessagesEndpoint
}

// yesNo translates a true/false boolean value into a yes/no setting suitable for the Mailgun API.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func trueFalse(b bool) string {
	return strconv.FormatBool(b)
}

// isValid returns true if, and only if,
// a CommonMessage instance is sufficiently initialized to send via the Mailgun interface.
func isValid(m Message) bool {
	if m == nil {
		return false
	}

	if !m.IsValid() {
		return false
	}

	if m.RecipientCount() == 0 {
		return false
	}

	if !validateStringList(m.Tags(), false) {
		return false
	}

	return true
}

func (m *PlainMessage) IsValid() bool {
	if !validateStringList(m.CC(), false) {
		return false
	}

	if !validateStringList(m.BCC(), false) {
		return false
	}

	if m.Template() != "" {
		// m.text or m.html not needed if template is supplied.
		//
		// From is not required if sending with a template that has a pre-set From header,
		// but it will override it if provided.
		return true
	}

	if m.From() == "" {
		return false
	}

	if m.Text() == "" && m.HTML() == "" {
		return false
	}

	return true
}

func (m *MimeMessage) IsValid() bool {
	return m.body != nil
}

// validateStringList returns true if, and only if,
// a slice of strings exists AND all of its elements exist,
// OR if the slice doesn't exist AND it's not required to exist.
// The requireOne parameter indicates whether the list is required to exist.
func validateStringList(list []string, requireOne bool) bool {
	hasOne := false

	if list == nil {
		return !requireOne
	}

	for _, a := range list {
		if a == "" {
			return false
		}

		hasOne = true
	}

	return hasOne
}
