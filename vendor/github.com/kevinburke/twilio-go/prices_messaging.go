package twilio

import (
	"context"
	"net/url"
)

const messagingPathPart = "Messaging"

type MessagingPriceService struct {
	Countries *CountryMessagingPriceService
}

type CountryMessagingPriceService struct {
	client *Client
}

type OutboundSMSPrice struct {
	Carrier string         `json:"carrier"`
	MCC     string         `json:"mcc"`
	MNC     string         `json:"mnc"`
	Prices  []InboundPrice `json:"prices"`
}

type MessagePrices struct {
	Country           string             `json:"country"`
	IsoCountry        string             `json:"iso_country"`
	OutboundSMSPrices []OutboundSMSPrice `json:"outbound_sms_prices"`
	InboundSmsPrices  []InboundPrice     `json:"inbound_sms_prices"`
	PriceUnit         string             `json:"price_unit"`
	URL               string             `json:"url"`
}

// returns the message price by country
func (cmps *CountryMessagingPriceService) Get(ctx context.Context, isoCountry string, data url.Values) (*MessagePrices, error) {
	messagePrice := new(MessagePrices)
	err := cmps.client.ListResource(ctx, messagingPathPart+"/Countries/"+isoCountry, data, messagePrice)
	return messagePrice, err
}

// returns a list of countries where Twilio messaging services are available and the corresponding URL
// for retrieving the country specific messaging prices.
func (cmps *CountryMessagingPriceService) GetPage(ctx context.Context, data url.Values) (*CountriesPricePage, error) {
	return cmps.GetPageIterator(data).Next(ctx)
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (cmps *CountryMessagingPriceService) GetPageIterator(data url.Values) *CountryPricePageIterator {
	iter := NewPageIterator(cmps.client, data, messagingPathPart+"/Countries")
	return &CountryPricePageIterator{
		p: iter,
	}
}
