package twilio

import (
	"context"
	"net/url"

	types "github.com/kevinburke/go-types"
)

const numbersPathPart = "IncomingPhoneNumbers"

type NumberPurchasingService struct {
	client   *Client
	pathPart string
}

type IncomingNumberService struct {
	*NumberPurchasingService
	client   *Client
	Local    *NumberPurchasingService
	TollFree *NumberPurchasingService
}

type NumberCapability struct {
	MMS   bool `json:"mms"`
	SMS   bool `json:"sms"`
	Voice bool `json:"voice"`
}

type IncomingPhoneNumber struct {
	Sid                  string            `json:"sid"`
	PhoneNumber          PhoneNumber       `json:"phone_number"`
	FriendlyName         string            `json:"friendly_name"`
	DateCreated          TwilioTime        `json:"date_created"`
	AccountSid           string            `json:"account_sid"`
	AddressRequirements  string            `json:"address_requirements"`
	APIVersion           string            `json:"api_version"`
	Beta                 bool              `json:"beta"`
	Capabilities         *NumberCapability `json:"capabilities"`
	DateUpdated          TwilioTime        `json:"date_updated"`
	EmergencyAddressSid  types.NullString  `json:"emergency_address_sid"`
	EmergencyStatus      string            `json:"emergency_status"`
	SMSApplicationSid    string            `json:"sms_application_sid"`
	SMSFallbackMethod    string            `json:"sms_fallback_method"`
	SMSFallbackURL       string            `json:"sms_fallback_url"`
	SMSMethod            string            `json:"sms_method"`
	SMSURL               string            `json:"sms_url"`
	StatusCallback       string            `json:"status_callback"`
	StatusCallbackMethod string            `json:"status_callback_method"`
	TrunkSid             types.NullString  `json:"trunk_sid"`
	URI                  string            `json:"uri"`
	VoiceApplicationSid  string            `json:"voice_application_sid"`
	VoiceCallerIDLookup  bool              `json:"voice_caller_id_lookup"`
	VoiceFallbackMethod  string            `json:"voice_fallback_method"`
	VoiceFallbackURL     string            `json:"voice_fallback_url"`
	VoiceMethod          string            `json:"voice_method"`
	VoiceURL             string            `json:"voice_url"`
}

type IncomingPhoneNumberPage struct {
	Page
	IncomingPhoneNumbers []*IncomingPhoneNumber `json:"incoming_phone_numbers"`
}

// Create a phone number (buy a number) with the given values.
//
// https://www.twilio.com/docs/api/rest/incoming-phone-numbers#toll-free-incomingphonenumber-factory-resource
func (n *NumberPurchasingService) Create(ctx context.Context, data url.Values) (*IncomingPhoneNumber, error) {
	number := new(IncomingPhoneNumber)
	pathPart := numbersPathPart
	if n.pathPart != "" {
		pathPart += "/" + n.pathPart
	}
	err := n.client.CreateResource(ctx, pathPart, data, number)
	return number, err
}

// BuyNumber attempts to buy the provided phoneNumber and returns it if
// successful.
func (ipn *IncomingNumberService) BuyNumber(phoneNumber string) (*IncomingPhoneNumber, error) {
	data := url.Values{"PhoneNumber": []string{phoneNumber}}
	return ipn.NumberPurchasingService.Create(context.Background(), data)
}

// Get retrieves a single IncomingPhoneNumber.
func (ipn *IncomingNumberService) Get(ctx context.Context, sid string) (*IncomingPhoneNumber, error) {
	number := new(IncomingPhoneNumber)
	err := ipn.client.GetResource(ctx, numbersPathPart, sid, number)
	return number, err
}

// Release removes an IncomingPhoneNumber from your account.
func (ipn *IncomingNumberService) Release(ctx context.Context, numberSid string) error {
	return ipn.client.DeleteResource(ctx, numbersPathPart, numberSid)
}

// Tries to update the incoming phone number's properties, and returns the updated resource representation if successful.
// https://www.twilio.com/docs/api/rest/incoming-phone-numbers#instance-post
func (ipn *IncomingNumberService) Update(ctx context.Context, sid string, data url.Values) (*IncomingPhoneNumber, error) {
	number := new(IncomingPhoneNumber)
	err := ipn.client.UpdateResource(ctx, numbersPathPart, sid, data, number)
	return number, err
}

// GetPage retrieves an IncomingPhoneNumberPage, filtered by the given data.
func (ins *IncomingNumberService) GetPage(ctx context.Context, data url.Values) (*IncomingPhoneNumberPage, error) {
	iter := ins.GetPageIterator(data)
	return iter.Next(ctx)
}

type IncomingPhoneNumberPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (c *IncomingNumberService) GetPageIterator(data url.Values) *IncomingPhoneNumberPageIterator {
	iter := NewPageIterator(c.client, data, numbersPathPart)
	return &IncomingPhoneNumberPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *IncomingPhoneNumberPageIterator) Next(ctx context.Context) (*IncomingPhoneNumberPage, error) {
	cp := new(IncomingPhoneNumberPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
