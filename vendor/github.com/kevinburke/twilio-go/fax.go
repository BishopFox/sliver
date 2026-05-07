package twilio

import (
	"context"
	"net/url"
)

const faxPathPart = "Faxes"

type FaxService struct {
	client *Client
}

type Fax struct {
	Sid         string      `json:"sid"`
	AccountSid  string      `json:"account_sid"`
	From        PhoneNumber `json:"from"`
	To          PhoneNumber `json:"to"`
	Direction   Direction   `json:"direction"`
	NumPages    uint        `json:"num_pages"`
	Duration    uint        `json:"duration"`
	MediaURL    string      `json:"media_url"`
	Status      Status      `json:"status"`
	DateCreated TwilioTime  `json:"date_created"`
	DateUpdated TwilioTime  `json:"date_updated"`
	Price       string      `json:"price"`
	PriceUnit   string      `json:"price_unit"`
	Quality     string      `json:"quality"`
	URL         string      `json:"url"`
	APIVersion  string      `json:"api_version"`
}

// FaxPage represents a page of Faxes.
type FaxPage struct {
	Meta  Meta   `json:"meta"`
	Faxes []*Fax `json:"faxes"`
}

// FriendlyPrice flips the sign of the Price (which is usually reported from
// the API as a negative number) and adds an appropriate currency symbol in
// front of it. For example, a PriceUnit of "USD" and a Price of "-1.25" is
// reported as "$1.25".
func (f *Fax) FriendlyPrice() string {
	if f == nil {
		return ""
	}
	if len(f.Price) >= 1 && f.Price[0] != '-' {
		// this is a hack because prices are returned as positive numbers in this
		// API
		return price(f.PriceUnit, "-"+f.Price)
	}
	return price(f.PriceUnit, "-"+f.Price)
}

// Get finds a single Fax resource by its sid, or returns an error.
func (f *FaxService) Get(ctx context.Context, sid string) (*Fax, error) {
	fax := new(Fax)
	err := f.client.GetResource(ctx, faxPathPart, sid, fax)
	return fax, err
}

// Update the fax with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/fax/rest/faxes#fax-instance-post
func (c *FaxService) Update(ctx context.Context, sid string, data url.Values) (*Fax, error) {
	fax := new(Fax)
	err := c.client.UpdateResource(ctx, faxPathPart, sid, data, fax)
	return fax, err
}

// Cancel an in-progress Fax with the given sid. Cancel will not affect
// in-progress Faxes, only those in queued or in-progress.
func (f *FaxService) Cancel(sid string) (*Fax, error) {
	data := url.Values{}
	data.Set("Status", string(StatusCanceled))
	return f.Update(context.Background(), sid, data)
}

// GetPage returns a single Page of resources, filtered by data.
//
// See https://www.twilio.com/docs/api/fax/rest/faxes#fax-list-get.
func (f *FaxService) GetPage(ctx context.Context, data url.Values) (*FaxPage, error) {
	return f.GetPageIterator(data).Next(ctx)
}

// Create a fax with the given url.Values. For more information on valid values,
// see https://www.twilio.com/docs/api/fax/rest/faxes#fax-list-post or use the
// SendFax helper.
func (f *FaxService) Create(ctx context.Context, data url.Values) (*Fax, error) {
	fax := new(Fax)
	err := f.client.CreateResource(ctx, faxPathPart, data, fax)
	return fax, err
}

// SendFax sends an outbound Fax with the given mediaURL. For more control over
// the parameters, use FaxService.Create.
func (f *FaxService) SendFax(from string, to string, mediaURL *url.URL) (*Fax, error) {
	v := url.Values{
		"MediaUrl": []string{mediaURL.String()},
		"From":     []string{from},
		"To":       []string{to},
	}
	return f.Create(context.Background(), v)
}

// FaxPageIterator lets you retrieve consecutive pages of resources.
type FaxPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*FaxPage, error)
}

type faxPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a FaxPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (f *FaxService) GetPageIterator(data url.Values) FaxPageIterator {
	iter := NewPageIterator(f.client, data, faxPathPart)
	return &faxPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (f *faxPageIterator) Next(ctx context.Context) (*FaxPage, error) {
	ap := new(FaxPage)
	err := f.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	f.p.SetNextPageURI(ap.Meta.NextPageURL)
	return ap, nil
}
