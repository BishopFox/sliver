package mailgun

import (
	"context"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// GetDomainTracking returns tracking settings for a domain
func (mg *Client) GetDomainTracking(ctx context.Context, domain string) (mtypes.DomainTracking, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/tracking")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var resp mtypes.DomainTrackingResponse
	err := getResponseFromJSON(ctx, r, &resp)
	return resp.Tracking, err
}

func (mg *Client) UpdateClickTracking(ctx context.Context, domain, active string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/tracking/click")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("active", active)
	_, err := makePutRequest(ctx, r, payload)
	return err
}

func (mg *Client) UpdateUnsubscribeTracking(ctx context.Context, domain, active, htmlFooter, textFooter string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/tracking/unsubscribe")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("active", active)
	payload.addValue("html_footer", htmlFooter)
	payload.addValue("text_footer", textFooter)
	_, err := makePutRequest(ctx, r, payload)
	return err
}

func (mg *Client) UpdateOpenTracking(ctx context.Context, domain, active string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/tracking/open")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("active", active)
	_, err := makePutRequest(ctx, r, payload)
	return err
}
