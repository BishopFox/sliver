package twilio

import (
	"context"
	"net/url"
)

const callerIDPathPart = "OutgoingCallerIds"

type OutgoingCallerIDService struct {
	client *Client
}

type OutgoingCallerID struct {
	Sid          string      `json:"sid"`
	FriendlyName string      `json:"friendly_name"`
	PhoneNumber  PhoneNumber `json:"phone_number"`
	AccountSid   string      `json:"account_sid"`
	DateCreated  TwilioTime  `json:"date_created"`
	DateUpdated  TwilioTime  `json:"date_updated"`
	URI          string      `json:"uri"`
}

type CallerIDRequest struct {
	AccountSid   string      `json:"account_sid"`
	PhoneNumber  PhoneNumber `json:"phone_number"`
	FriendlyName string      `json:"friendly_name"`
	// Usually six digits, but a string to avoid stripping leading 0's
	ValidationCode string `json:"validation_code"`
	CallSid        string `json:"call_sid"`
}

type OutgoingCallerIDPage struct {
	Page
	OutgoingCallerIDs []*OutgoingCallerID `json:"outgoing_caller_ids"`
}

// Create a new OutgoingCallerID. Note the ValidationCode is only returned in
// response to a Create, you can't retrieve it later.
//
// https://www.twilio.com/docs/api/rest/outgoing-caller-ids#list-post
func (c *OutgoingCallerIDService) Create(ctx context.Context, data url.Values) (*CallerIDRequest, error) {
	id := new(CallerIDRequest)
	err := c.client.CreateResource(ctx, callerIDPathPart, data, id)
	return id, err
}

func (o *OutgoingCallerIDService) Get(ctx context.Context, sid string) (*OutgoingCallerID, error) {
	id := new(OutgoingCallerID)
	err := o.client.GetResource(ctx, callerIDPathPart, sid, id)
	return id, err
}

func (o *OutgoingCallerIDService) GetPage(ctx context.Context, data url.Values) (*OutgoingCallerIDPage, error) {
	op := new(OutgoingCallerIDPage)
	err := o.client.ListResource(ctx, callerIDPathPart, data, op)
	return op, err
}

// Update the caller ID with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/rest/outgoing-caller-ids#list
func (o *OutgoingCallerIDService) Update(ctx context.Context, sid string, data url.Values) (*OutgoingCallerID, error) {
	id := new(OutgoingCallerID)
	err := o.client.UpdateResource(ctx, callerIDPathPart, sid, data, id)
	return id, err
}

// Delete the Caller ID with the given sid. If the ID has already been deleted,
// or does not exist, Delete returns nil. If another error or a timeout occurs,
// the error is returned.
func (o *OutgoingCallerIDService) Delete(ctx context.Context, sid string) error {
	return o.client.DeleteResource(ctx, callerIDPathPart, sid)
}

// OutgoingCallerIDPageIterator lets you retrieve consecutive pages of resources.
type OutgoingCallerIDPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a OutgoingCallerIDPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (o *OutgoingCallerIDService) GetPageIterator(data url.Values) *OutgoingCallerIDPageIterator {
	iter := NewPageIterator(o.client, data, callerIDPathPart)
	return &OutgoingCallerIDPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (o *OutgoingCallerIDPageIterator) Next(ctx context.Context) (*OutgoingCallerIDPage, error) {
	op := new(OutgoingCallerIDPage)
	err := o.p.Next(ctx, op)
	if err != nil {
		return nil, err
	}
	o.p.SetNextPageURI(op.NextPageURI)
	return op, nil
}
