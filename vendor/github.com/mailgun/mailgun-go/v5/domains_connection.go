package mailgun

import (
	"context"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// GetDomainConnection returns delivery connection settings for the defined domain
//
// Deprecated: use GetDomain instead
//
// TODO(v6): remove
func (mg *Client) GetDomainConnection(ctx context.Context, domain string) (mtypes.DomainConnection, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/connection")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var resp mtypes.DomainConnectionResponse
	err := getResponseFromJSON(ctx, r, &resp)
	return resp.Connection, err
}

// UpdateDomainConnection updates the specified delivery connection settings for the defined domain
//
// Deprecated: use UpdateDomain instead
//
// TODO(v6): remove
func (mg *Client) UpdateDomainConnection(ctx context.Context, domain string, settings mtypes.DomainConnection) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/connection")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("require_tls", boolToString(settings.RequireTLS))
	payload.addValue("skip_verification", boolToString(settings.SkipVerification))
	_, err := makePutRequest(ctx, r, payload)
	return err
}
