package twilio

import (
	"context"
	"net/url"
)

const keyPathPart = "Keys"

type KeyService struct {
	client *Client
}

// A Twilio Key. For more documentation, see
// https://www.twilio.com/docs/api/rest/keys#instance
type Key struct {
	DateCreated  TwilioTime `json:"date_created"`
	DateUpdated  TwilioTime `json:"date_updated"`
	Sid          string     `json:"sid"`
	FriendlyName string     `json:"friendly_name"`
	Secret       string     `json:"secret"`
}

type KeyPage struct {
	Page
	Keys []*Key `json:"keys"`
}

func (c *KeyService) Get(ctx context.Context, sid string) (*Key, error) {
	key := new(Key)
	err := c.client.GetResource(ctx, keyPathPart, sid, key)
	return key, err
}

func (c *KeyService) GetPage(ctx context.Context, data url.Values) (*KeyPage, error) {
	iter := c.GetPageIterator(data)
	return iter.Next(ctx)
}

// Create a new Key. Note the Secret is only returned in response to a Create,
// you can't retrieve it later.
//
// https://www.twilio.com/docs/api/rest/keys#list-post
func (c *KeyService) Create(ctx context.Context, data url.Values) (*Key, error) {
	key := new(Key)
	err := c.client.CreateResource(ctx, keyPathPart, data, key)
	return key, err
}

// Update the key with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/rest/keys#instance-post
func (a *KeyService) Update(ctx context.Context, sid string, data url.Values) (*Key, error) {
	key := new(Key)
	err := a.client.UpdateResource(ctx, keyPathPart, sid, data, key)
	return key, err
}

// Delete the Key with the given sid. If the Key has already been
// deleted, or does not exist, Delete returns nil. If another error or a
// timeout occurs, the error is returned.
func (r *KeyService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, keyPathPart, sid)
}

// KeyPageIterator lets you retrieve consecutive pages of resources.
type KeyPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a KeyPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (c *KeyService) GetPageIterator(data url.Values) *KeyPageIterator {
	iter := NewPageIterator(c.client, data, keyPathPart)
	return &KeyPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *KeyPageIterator) Next(ctx context.Context) (*KeyPage, error) {
	kp := new(KeyPage)
	err := c.p.Next(ctx, kp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(kp.NextPageURI)
	return kp, nil
}
