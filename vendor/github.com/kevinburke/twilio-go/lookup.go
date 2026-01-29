package twilio

import (
	"context"
	"net/url"
)

const phoneNumbersPath = "PhoneNumbers"

type LookupPhoneNumbersService struct {
	client *Client
}

type CallerLookup struct {
	CallerName string `json:"caller_name"`
	CallerType string `json:"caller_type"`
	ErrorCode  int    `json:"error_code"`
}

type CarrierLookup struct {
	Type              string `json:"type"`
	ErrorCode         int    `json:"error_code"`
	MobileNetworkCode string `json:"mobile_network_code"`
	MobileCountryCode string `json:"mobile_country_code"`
	Name              string `json:"name"`
}

type PhoneLookup struct {
	CountryCode    string        `json:"country_code"`
	PhoneNumber    string        `json:"phone_number"`
	NationalFormat string        `json:"national_format"`
	URL            string        `json:"url"`
	CallerName     CallerLookup  `json:"caller_name"`
	Carrier        CarrierLookup `json:"carrier"`
}

// Get calls the lookups API to retrieve information about a phone number
func (s *LookupPhoneNumbersService) Get(ctx context.Context, phone string, data url.Values) (*PhoneLookup, error) {
	lookup := new(PhoneLookup)
	err := s.client.MakeRequest(ctx, "GET", phoneNumbersPath+"/"+phone, data, lookup)
	return lookup, err
}
