package twilio

import (
	"context"
	"net/url"
)

const availableNumbersPath = "AvailablePhoneNumbers"

type AvailableNumberBase struct {
	client   *Client
	pathPart string
}

type AvailableNumberService struct {
	Local              *AvailableNumberBase
	Mobile             *AvailableNumberBase
	TollFree           *AvailableNumberBase
	SupportedCountries *SupportedCountriesService
}

// The subresources of the AvailableNumbers resource let you search for local, toll-free and
// mobile phone numbers that are available for you to purchase.
// See https://www.twilio.com/docs/api/rest/available-phone-numbers for details
type AvailableNumber struct {
	FriendlyName        string            `json:"friendly_name"`
	PhoneNumber         PhoneNumber       `json:"phone_number"`
	Lata                string            `json:"lata"`
	RateCenter          string            `json:"rate_center"`
	Latitude            string            `json:"latitude"`
	Longitude           string            `json:"longitude"`
	Region              string            `json:"region"`
	PostalCode          string            `json:"postal_code"`
	ISOCountry          string            `json:"iso_country"`
	Capabilities        *NumberCapability `json:"capabilities"`
	AddressRequirements string            `json:"address_requirements"`
	Beta                bool              `json:"beta"`
}

type AvailableNumberPage struct {
	URI     string             `json:"uri"`
	Numbers []*AvailableNumber `json:"available_phone_numbers"`
}

// GetPage returns a page of available phone numbers.
//
// For more information, see the Twilio documentation:
// https://www.twilio.com/docs/api/rest/available-phone-numbers#local
// https://www.twilio.com/docs/api/rest/available-phone-numbers#toll-free
// https://www.twilio.com/docs/api/rest/available-phone-numbers#mobile
func (s *AvailableNumberBase) GetPage(ctx context.Context, isoCountry string, filters url.Values) (*AvailableNumberPage, error) {
	sr := new(AvailableNumberPage)
	path := availableNumbersPath + "/" + isoCountry + "/" + s.pathPart
	err := s.client.ListResource(ctx, path, filters, sr)
	if err != nil {
		return nil, err
	}

	return sr, nil
}

type SupportedCountriesService struct {
	client *Client
}

type SupportedCountry struct {
	// The ISO Country code to lookup phone numbers for.
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	URI         string `json:"uri"`

	// If true, all phone numbers available in this country are new to the Twilio platform.
	// If false, all numbers are not in the Twilio Phone Number Beta program.
	Beta            bool              `json:"beta"`
	SubresourceURIs map[string]string `json:"subresource_uris"`
}

type SupportedCountries struct {
	URI       string              `json:"uri"`
	Countries []*SupportedCountry `json:"countries"`
}

// Get returns supported countries.
// If beta is true, only include countries where phone numbers new to the Twilio platform are available.
// If false, do not include new inventory.
//
// See https://www.twilio.com/docs/phone-numbers/api/available-phone-numbers#countries
func (s *SupportedCountriesService) Get(ctx context.Context, beta bool) (*SupportedCountries, error) {
	sc := new(SupportedCountries)
	path := availableNumbersPath
	data := url.Values{}
	if beta {
		data.Set("Beta", "true")
	}
	err := s.client.ListResource(ctx, path, data, sc)
	if err != nil {
		return nil, err
	}

	return sc, nil
}
