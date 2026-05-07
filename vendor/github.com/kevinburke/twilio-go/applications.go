package twilio

import (
	"context"
	"net/url"
)

const applicationPathPart = "Applications"

type ApplicationService struct {
	client *Client
}

// A Twilio Application. For more documentation, see
// https://www.twilio.com/docs/api/rest/applications#instance
type Application struct {
	AccountSid            string     `json:"account_sid"`
	APIVersion            string     `json:"api_version"`
	DateCreated           TwilioTime `json:"date_created"`
	DateUpdated           TwilioTime `json:"date_updated"`
	FriendlyName          string     `json:"friendly_name"`
	MessageStatusCallback string     `json:"message_status_callback"`
	Sid                   string     `json:"sid"`
	SMSFallbackMethod     string     `json:"sms_fallback_method"`
	SMSFallbackURL        string     `json:"sms_fallback_url"`
	SMSURL                string     `json:"sms_url"`
	StatusCallback        string     `json:"status_callback"`
	StatusCallbackMethod  string     `json:"status_callback_method"`
	URI                   string     `json:"uri"`
	VoiceCallerIDLookup   bool       `json:"voice_caller_id_lookup"`
	VoiceFallbackMethod   string     `json:"voice_fallback_method"`
	VoiceFallbackURL      string     `json:"voice_fallback_url"`
	VoiceMethod           string     `json:"voice_method"`
	VoiceURL              string     `json:"voice_url"`
}

type ApplicationPage struct {
	Page
	Applications []*Application `json:"applications"`
}

func (c *ApplicationService) Get(ctx context.Context, sid string) (*Application, error) {
	application := new(Application)
	err := c.client.GetResource(ctx, applicationPathPart, sid, application)
	return application, err
}

func (c *ApplicationService) GetPage(ctx context.Context, data url.Values) (*ApplicationPage, error) {
	iter := c.GetPageIterator(data)
	return iter.Next(ctx)
}

// Create a new Application. This request must include a FriendlyName, and can
// include these values:
// https://www.twilio.com/docs/api/rest/applications#list-post-optional-parameters
func (c *ApplicationService) Create(ctx context.Context, data url.Values) (*Application, error) {
	application := new(Application)
	err := c.client.CreateResource(ctx, applicationPathPart, data, application)
	return application, err
}

// Update the application with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/rest/applications#instance-post
func (a *ApplicationService) Update(ctx context.Context, sid string, data url.Values) (*Application, error) {
	application := new(Application)
	err := a.client.UpdateResource(ctx, applicationPathPart, sid, data, application)
	return application, err
}

// Delete the Application with the given sid. If the Application has already been
// deleted, or does not exist, Delete returns nil. If another error or a
// timeout occurs, the error is returned.
func (r *ApplicationService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, applicationPathPart, sid)
}

// ApplicationPageIterator lets you retrieve consecutive pages of resources.
type ApplicationPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a ApplicationPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (c *ApplicationService) GetPageIterator(data url.Values) *ApplicationPageIterator {
	iter := NewPageIterator(c.client, data, applicationPathPart)
	return &ApplicationPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *ApplicationPageIterator) Next(ctx context.Context) (*ApplicationPage, error) {
	ap := new(ApplicationPage)
	err := c.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(ap.NextPageURI)
	return ap, nil
}
