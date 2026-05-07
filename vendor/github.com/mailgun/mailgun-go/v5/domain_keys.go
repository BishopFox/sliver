package mailgun

import (
	"context"
)

// UpdateDomainDkimSelector updates the DKIM selector for a domain
func (mg *Client) UpdateDomainDkimSelector(ctx context.Context, domain, dkimSelector string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/dkim_selector")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("dkim_selector", dkimSelector)
	_, err := makePutRequest(ctx, r, payload)
	return err
}
