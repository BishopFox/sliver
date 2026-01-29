package twilio

import (
	"context"
	"net/url"
)

const phoneNumbersPathPart = "PhoneNumbers"

type PhoneNumberPriceService struct {
	Countries *CountryPhoneNumberPriceService
}

type CountryPhoneNumberPriceService struct {
	client *Client
}

type PhoneNumberPrice struct {
	BasePrice    string `json:"base_price"`
	CurrentPrice string `json:"current_price"`
	NumberType   string `json:"number_type"`
}

type NumberPrices struct {
	Country           string             `json:"country"`
	IsoCountry        string             `json:"iso_country"`
	PhoneNumberPrices []PhoneNumberPrice `json:"phone_number_prices"`
	PriceUnit         string             `json:"price_unit"`
	URL               string             `json:"url"`
}

type PriceCountry struct {
	Country    string `json:"country"`
	IsoCountry string `json:"iso_country"`
	URL        string `json:"url"`
}

type CountriesPricePage struct {
	Meta      Meta            `json:"meta"`
	Countries []*PriceCountry `json:"countries"`
}

// returns the phone number price by country
func (cpnps *CountryPhoneNumberPriceService) Get(ctx context.Context, isoCountry string, data url.Values) (*NumberPrices, error) {
	numberPrice := new(NumberPrices)
	err := cpnps.client.ListResource(ctx, phoneNumbersPathPart+"/Countries/"+isoCountry, data, numberPrice)
	return numberPrice, err
}

// returns a list of countries where Twilio phone numbers are supported
func (cpnps *CountryPhoneNumberPriceService) GetPage(ctx context.Context, data url.Values) (*CountriesPricePage, error) {
	return cpnps.GetPageIterator(data).Next(ctx)
}

type CountryPricePageIterator struct {
	p *PageIterator
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (cpnps *CountryPhoneNumberPriceService) GetPageIterator(data url.Values) *CountryPricePageIterator {
	iter := NewPageIterator(cpnps.client, data, phoneNumbersPathPart+"/Countries")
	return &CountryPricePageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *CountryPricePageIterator) Next(ctx context.Context) (*CountriesPricePage, error) {
	cp := new(CountriesPricePage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.Meta.NextPageURL)
	return cp, nil
}
