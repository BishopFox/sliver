package twilio

import (
	"context"
	"net/url"
)

const transcriptionPathPart = "Transcriptions"

type TranscriptionService struct {
	client *Client
}

type Transcription struct {
	Sid               string         `json:"sid"`
	TranscriptionText string         `json:"transcription_text"`
	DateCreated       TwilioTime     `json:"date_created"`
	DateUpdated       TwilioTime     `json:"date_updated"`
	Duration          TwilioDuration `json:"duration"`
	Price             string         `json:"price"`
	PriceUnit         string         `json:"price_unit"`
	RecordingSid      string         `json:"recording_sid"`
	Status            Status         `json:"status"`
	Type              string         `json:"type"`
	AccountSid        string         `json:"account_sid"`
	APIVersion        string         `json:"api_version"`
	URI               string         `json:"uri"`
}

type TranscriptionPage struct {
	Page
	Transcriptions []*Transcription
}

// FriendlyPrice flips the sign of the Price (which is usually reported from
// the API as a negative number) and adds an appropriate currency symbol in
// front of it. For example, a PriceUnit of "USD" and a Price of "-1.25" is
// reported as "$1.25".
func (t *Transcription) FriendlyPrice() string {
	if t == nil {
		return ""
	}
	return price(t.PriceUnit, t.Price)
}

// Get returns a single Transcription or an error.
func (c *TranscriptionService) Get(ctx context.Context, sid string) (*Transcription, error) {
	transcription := new(Transcription)
	err := c.client.GetResource(ctx, transcriptionPathPart, sid, transcription)
	return transcription, err
}

// Delete the Transcription with the given sid. If the Transcription has
// already been deleted, or does not exist, Delete returns nil. If another
// error or a timeout occurs, the error is returned.
func (c *TranscriptionService) Delete(ctx context.Context, sid string) error {
	return c.client.DeleteResource(ctx, transcriptionPathPart, sid)
}

func (c *TranscriptionService) GetPage(ctx context.Context, data url.Values) (*TranscriptionPage, error) {
	iter := c.GetPageIterator(data)
	return iter.Next(ctx)
}

type TranscriptionPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a TranscriptionPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again to
// retrieve subsequent pages).
func (c *TranscriptionService) GetPageIterator(data url.Values) *TranscriptionPageIterator {
	iter := NewPageIterator(c.client, data, transcriptionPathPart)
	return &TranscriptionPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *TranscriptionPageIterator) Next(ctx context.Context) (*TranscriptionPage, error) {
	cp := new(TranscriptionPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
