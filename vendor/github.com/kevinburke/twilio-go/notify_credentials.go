package twilio

import (
	"context"
	"net/url"
)

type CredentialType string

const credentialsPathPart = "Credentials"

// Credential type. Currently APNS, FCM and GCM types are supported.
// https://www.twilio.com/docs/api/notify/rest/credentials
const (
	TypeGCM = CredentialType("gcm")
	TypeFCM = CredentialType("fcm")
	TypeAPN = CredentialType("apn")
)

type NotifyCredentialsService struct {
	client *Client
}

type NotifyCredential struct {
	Sid          string         `json:"sid"`
	FriendlyName string         `json:"friendly_name"`
	AccountSid   string         `json:"account_sid"`
	Type         CredentialType `json:"type"`
	DateCreated  TwilioTime     `json:"date_created"`
	DateUpdated  TwilioTime     `json:"date_updated"`
	URL          string         `json:"url"`
}

type NotifyCredentialPage struct {
	Page
	Credentials []*NotifyCredential `json:"credentials"`
}

type NotifyCredentialPageIterator struct {
	p *PageIterator
}

func (n *NotifyCredentialsService) Create(ctx context.Context, data url.Values) (*NotifyCredential, error) {
	credential := new(NotifyCredential)
	err := n.client.CreateResource(ctx, credentialsPathPart, data, credential)
	return credential, err
}

// To create an FCM credential, use Secret parameter, which can be found in your Firebase console as Server key
// See https://www.twilio.com/docs/api/notify/rest/credentials#create-a-credential for details
func (n *NotifyCredentialsService) CreateFCM(ctx context.Context, friendlyName string, secret string) (*NotifyCredential, error) {
	data := url.Values{}
	data.Set("FriendlyName", friendlyName)
	data.Set("Type", "fcm")
	data.Set("Secret", secret)

	return n.Create(ctx, data)
}

func (n *NotifyCredentialsService) CreateGCM(ctx context.Context, friendlyName string, apiKey string) (*NotifyCredential, error) {
	data := url.Values{}
	data.Set("FriendlyName", friendlyName)
	data.Set("Type", "gcm")
	data.Set("ApiKey", apiKey)

	return n.Create(ctx, data)
}

func (n *NotifyCredentialsService) CreateAPN(ctx context.Context, friendlyName string, cert string, privateKey string, sandbox bool) (*NotifyCredential, error) {
	data := url.Values{}
	data.Set("FriendlyName", friendlyName)
	data.Set("Type", "apn")
	data.Set("Certificate", cert)
	data.Set("PrivateKey", privateKey)

	if sandbox {
		data.Set("Sandbox", "true")
	} else {
		data.Set("Sandbox", "false")
	}

	return n.Create(ctx, data)
}

func (n *NotifyCredentialsService) GetPage(ctx context.Context, data url.Values) (*NotifyCredentialPage, error) {
	iter := n.GetPageIterator(data)
	return iter.Next(ctx)
}

func (n *NotifyCredentialsService) Get(ctx context.Context, sid string) (*NotifyCredential, error) {
	credential := new(NotifyCredential)
	err := n.client.GetResource(ctx, credentialsPathPart, sid, credential)
	return credential, err
}

func (n *NotifyCredentialsService) Update(ctx context.Context, sid string, data url.Values) (*NotifyCredential, error) {
	credential := new(NotifyCredential)
	err := n.client.UpdateResource(ctx, credentialsPathPart, sid, data, credential)
	return credential, err
}

func (n *NotifyCredentialsService) Delete(ctx context.Context, sid string) error {
	return n.client.DeleteResource(ctx, credentialsPathPart, sid)
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (n *NotifyCredentialsService) GetPageIterator(data url.Values) *NotifyCredentialPageIterator {
	iter := NewPageIterator(n.client, data, credentialsPathPart)
	return &NotifyCredentialPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (r *NotifyCredentialPageIterator) Next(ctx context.Context) (*NotifyCredentialPage, error) {
	rp := new(NotifyCredentialPage)
	err := r.p.Next(ctx, rp)
	if err != nil {
		return nil, err
	}
	r.p.SetNextPageURI(rp.NextPageURI)
	return rp, nil
}
