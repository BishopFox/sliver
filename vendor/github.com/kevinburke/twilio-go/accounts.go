package twilio

import (
	"context"
	"net/url"
)

// This resource is a little special because the endpoint is GET
// /2010-04-01/Accounts.json, there's no extra resource off Accounts. So in
// places we manually insert the `.json` in the right place.

// We need to build this relative to the root, but users can override the
// APIVersion, so give them a chance to before our init runs.
var accountPathPart string

func init() {
	accountPathPart = "/" + APIVersion + "/Accounts"
}

type Account struct {
	Sid             string            `json:"sid"`
	FriendlyName    string            `json:"friendly_name"`
	Type            string            `json:"type"`
	AuthToken       string            `json:"auth_token"`
	OwnerAccountSid string            `json:"owner_account_sid"`
	DateCreated     TwilioTime        `json:"date_created"`
	DateUpdated     TwilioTime        `json:"date_updated"`
	Status          Status            `json:"status"`
	SubresourceURIs map[string]string `json:"subresource_uris"`
	URI             string            `json:"uri"`
}

type AccountPage struct {
	Page
	Accounts []*Account `json:"accounts"`
}

type AccountService struct {
	client *Client
}

func (a *AccountService) Get(ctx context.Context, sid string) (*Account, error) {
	acct := new(Account)
	// hack because this is not a resource off of the account sid
	sidJSON := sid + ".json"
	err := a.client.GetResource(ctx, accountPathPart, sidJSON, acct)
	return acct, err
}

// Create a new Account with the specified values.
//
// https://www.twilio.com/docs/api/rest/subaccounts#creating-subaccounts
func (a *AccountService) Create(ctx context.Context, data url.Values) (*Account, error) {
	acct := new(Account)
	err := a.client.CreateResource(ctx, accountPathPart+".json", data, acct)
	return acct, err
}

// Update the key with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/rest/keys#instance-post
func (a *AccountService) Update(ctx context.Context, sid string, data url.Values) (*Account, error) {
	acct := new(Account)
	// hack because this is not a resource off of the account sid
	sidJSON := sid + ".json"
	err := a.client.UpdateResource(ctx, accountPathPart, sidJSON, data, acct)
	return acct, err
}

func (a *AccountService) GetPage(ctx context.Context, data url.Values) (*AccountPage, error) {
	iter := a.GetPageIterator(data)
	return iter.Next(ctx)
}

// AccountPageIterator lets you retrieve consecutive AccountPages.
type AccountPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a AccountPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (c *AccountService) GetPageIterator(data url.Values) *AccountPageIterator {
	iter := NewPageIterator(c.client, data, accountPathPart+".json")
	return &AccountPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *AccountPageIterator) Next(ctx context.Context) (*AccountPage, error) {
	cp := new(AccountPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
