// Package pushover provides a wrapper around the Pushover API
package pushover

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

// Regexp validation.
var tokenRegexp *regexp.Regexp

func init() {
	tokenRegexp = regexp.MustCompile(`^[A-Za-z0-9]{30}$`)
}

// APIEndpoint is the API base URL for any request.
var APIEndpoint = "https://api.pushover.net/1"

// Pushover custom errors.
var (
	ErrHTTPPushover              = errors.New("pushover: http error")
	ErrEmptyToken                = errors.New("pushover: empty API token")
	ErrEmptyURL                  = errors.New("pushover: empty URL, URLTitle needs an URL")
	ErrEmptyRecipientToken       = errors.New("pushover: empty recipient token")
	ErrInvalidRecipientToken     = errors.New("pushover: invalid recipient token")
	ErrInvalidHeaders            = errors.New("pushover: invalid headers in server response")
	ErrInvalidPriority           = errors.New("pushover: invalid priority")
	ErrInvalidToken              = errors.New("pushover: invalid API token")
	ErrMessageEmpty              = errors.New("pushover: message empty")
	ErrMessageTitleTooLong       = errors.New("pushover: message title too long")
	ErrMessageTooLong            = errors.New("pushover: message too long")
	ErrMessageAttachmentTooLarge = errors.New("pushover: message attachment is too large")
	ErrMessageURLTitleTooLong    = errors.New("pushover: message URL title too long")
	ErrMessageURLTooLong         = errors.New("pushover: message URL too long")
	ErrMissingAttachment         = errors.New("pushover: missing attachment")
	ErrMissingEmergencyParameter = errors.New("pushover: missing emergency parameter")
	ErrInvalidDeviceName         = errors.New("pushover: invalid device name")
	ErrEmptyReceipt              = errors.New("pushover: empty receipt")
	ErrGlancesMissingData        = errors.New("pushover: glance update data missing")
	ErrGlancesTitleTooLong       = errors.New("pushover: glance title too long")
	ErrGlancesTextTooLong        = errors.New("pushover: glance text too long")
	ErrGlancesSubtextTooLong     = errors.New("pushover: glance subtext too long")
	ErrGlancesInvalidPercent     = errors.New("pushover: glance percent must be in range of 0-100")
)

// API limitations.
const (
	// MessageMaxLength is the max message number of characters.
	MessageMaxLength = 1024
	// MessageTitleMaxLength is the max title number of characters.
	MessageTitleMaxLength = 250
	// MessageURLMaxLength is the max URL number of characters.
	MessageURLMaxLength = 512
	// MessageURLTitleMaxLength is the max URL title number of characters.
	MessageURLTitleMaxLength = 100
	// MessageMaxAttachmentByte is the max attachment size in byte.
	MessageMaxAttachmentByte = 2621440
)

// Message priorities
const (
	PriorityLowest    = -2
	PriorityLow       = -1
	PriorityNormal    = 0
	PriorityHigh      = 1
	PriorityEmergency = 2
)

// Sounds
const (
	SoundPushover     = "pushover"
	SoundBike         = "bike"
	SoundBugle        = "bugle"
	SoundCashRegister = "cashregister"
	SoundClassical    = "classical"
	SoundCosmic       = "cosmic"
	SoundFalling      = "falling"
	SoundGamelan      = "gamelan"
	SoundIncoming     = "incoming"
	SoundIntermission = "intermission"
	SoundMagic        = "magic"
	SoundMechanical   = "mechanical"
	SoundPianobar     = "pianobar"
	SoundSiren        = "siren"
	SoundSpaceAlarm   = "spacealarm"
	SoundTugBoat      = "tugboat"
	SoundAlien        = "alien"
	SoundClimb        = "climb"
	SoundPersistent   = "persistent"
	SoundEcho         = "echo"
	SoundUpDown       = "updown"
	SoundVibrate      = "vibrate"
	SoundNone         = "none"
)

// Pushover is the representation of an app using the pushover API.
type Pushover struct {
	token string
}

// New returns a new app to talk to the pushover API.
func New(token string) *Pushover {
	return &Pushover{token}
}

// Validate Pushover token.
func (p *Pushover) validate() error {
	// Check empty token
	if p.token == "" {
		return ErrEmptyToken
	}

	// Check invalid token
	if !tokenRegexp.MatchString(p.token) {
		return ErrInvalidToken
	}
	return nil
}

// SendMessage is used to send message to a recipient.
func (p *Pushover) SendMessage(message *Message, recipient *Recipient) (*Response, error) {
	// Validate pushover
	if err := p.validate(); err != nil {
		return nil, err
	}

	// Validate recipient
	if err := recipient.validate(); err != nil {
		return nil, err
	}

	// Validate message
	if err := message.validate(); err != nil {
		return nil, err
	}

	return message.send(p.token, recipient.token)
}

// SendGlanceUpdate is used to send glance updates to a recipient.
// It can be used to display widgets on a smart watch
func (p *Pushover) SendGlanceUpdate(msg *Glance, rec *Recipient) (*Response, error) {
	// Validate pushover
	if err := p.validate(); err != nil {
		return nil, err
	}

	// Validate rec
	if err := rec.validate(); err != nil {
		return nil, err
	}

	// Validate msg
	if err := msg.validate(); err != nil {
		return nil, err
	}

	return msg.send(p.token, rec.token)
}

// GetReceiptDetails return detailed information about a receipt. This is used
// used to check the acknowledged status of an Emergency notification.
func (p *Pushover) GetReceiptDetails(receipt string) (*ReceiptDetails, error) {
	url := fmt.Sprintf("%s/receipts/%s.json?token=%s", APIEndpoint, receipt, p.token)

	if receipt == "" {
		return nil, ErrEmptyReceipt
	}

	// Send request
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// Decode the JSON response
	var details *ReceiptDetails
	if err = json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return details, nil
}

// GetRecipientDetails allows to check if a recipient exists, if it's a group
// and the devices associated to this recipient. The Errors field of the
// RecipientDetails object will contain an error if the recipient is not valid
// in the Pushover API.
func (p *Pushover) GetRecipientDetails(recipient *Recipient) (*RecipientDetails, error) {
	endpoint := fmt.Sprintf("%s/users/validate.json", APIEndpoint)

	// Validate pushover
	if err := p.validate(); err != nil {
		return nil, err
	}

	// Validate recipient
	if err := recipient.validate(); err != nil {
		return nil, err
	}

	req, err := newURLEncodedRequest("POST", endpoint,
		map[string]string{"token": p.token, "user": recipient.token})
	if err != nil {
		return nil, err
	}

	var response RecipientDetails
	if err := do(req, &response, false); err != nil {
		return nil, err
	}

	return &response, nil
}

// CancelEmergencyNotification helps stop a notification retry in case of a
// notification with an Emergency priority before reaching the expiration time.
// It requires the response receipt in order to stop the right notification.
func (p *Pushover) CancelEmergencyNotification(receipt string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/receipts/%s/cancel.json", APIEndpoint, receipt)

	req, err := newURLEncodedRequest("POST", endpoint, map[string]string{"token": p.token})
	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// CreateGroup creates a new pushover group with the given name. The Group ID is returned as Response.Group.
// ref: https://pushover.net/api/groups#create
func (p *Pushover) CreateGroup(name string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups.json", APIEndpoint)

	req, err := newURLEncodedRequest("POST", endpoint, map[string]string{
		"token": p.token,
		"name":  name,
	})

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// ListGroups retrieves a list of groups.
// ref: https://pushover.net/api/groups#list
func (p *Pushover) ListGroups() (*GroupsListResponse, error) {
	url := fmt.Sprintf("%s/groups.json?token=%s", APIEndpoint, p.token)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var groups *GroupsListResponse

	if err = json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroup retrieves information about a given groupKey.
// ref: https://pushover.net/api/groups#show
func (p *Pushover) GetGroup(groupKey string) (*GroupDetailsResponse, error) {
	url := fmt.Sprintf("%s/groups/%s.json?token=%s", APIEndpoint, groupKey, p.token)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var groups *GroupDetailsResponse

	if err = json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// AddUserToGroup adds a user with userKey to the group with groupKey.
// If device or memo are provided, they are associated with the subscription.
// ref:  https://pushover.net/api/groups#add_user
func (p *Pushover) AddUserToGroup(groupKey, userKey string, device string, memo string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups/%s/add_user.json", APIEndpoint, groupKey)

	params := map[string]string{
		"token": p.token,
		"user":  userKey,
	}

	if device != "" {
		params["device"] = device
	}

	if memo != "" {
		params["memo"] = memo
	}

	req, err := newURLEncodedRequest("POST", endpoint, params)

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// RemoveUserFromGroup removes a user with userKey from the group with groupKey.
// If device is provided, only the given device is removed from the group.
// ref: https://pushover.net/api/groups#remove_user
func (p *Pushover) RemoveUserFromGroup(groupKey, userKey string, device string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups/%s/remove_user.json", APIEndpoint, groupKey)

	params := map[string]string{
		"token": p.token,
		"user":  userKey,
	}

	if device != "" {
		params["device"] = device
	}

	req, err := newURLEncodedRequest("POST", endpoint, params)

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// DisableUserInGroup disables a user with userKey in the group with groupKey.
// If device is provided, only the given device is disabled in the group.
// ref: https://pushover.net/api/groups#disable_user
func (p *Pushover) DisableUserInGroup(groupKey, userKey string, device string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups/%s/disable_user.json", APIEndpoint, groupKey)

	params := map[string]string{
		"token": p.token,
		"user":  userKey,
	}

	if device != "" {
		params["device"] = device
	}

	req, err := newURLEncodedRequest("POST", endpoint, params)

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// EnableUserInGroup enables a user with userKey in the group with groupKey.
// If device is provided, only the given device is enabled in the group.
// ref: https://pushover.net/api/groups#enable_user
func (p *Pushover) EnableUserInGroup(groupKey, userKey string, device string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups/%s/enable_user.json", APIEndpoint, groupKey)

	params := map[string]string{
		"token": p.token,
		"user":  userKey,
	}

	if device != "" {
		params["device"] = device
	}

	req, err := newURLEncodedRequest("POST", endpoint, params)

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}

// RenameGroup renames the group with groupKey to the provided name.
// ref: https://pushover.net/api/groups#rename
func (p *Pushover) RenameGroup(groupKey string, name string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/groups/%s/rename.json", APIEndpoint, groupKey)

	params := map[string]string{
		"token": p.token,
		"name":  name,
	}

	req, err := newURLEncodedRequest("POST", endpoint, params)

	if err != nil {
		return nil, err
	}

	response := &Response{}
	if err := do(req, response, false); err != nil {
		return nil, err
	}

	return response, nil
}
