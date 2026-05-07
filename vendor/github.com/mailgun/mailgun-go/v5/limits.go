package mailgun

import (
	"context"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// GetTagLimits returns tracking settings for a domain
func (mg *Client) GetTagLimits(ctx context.Context, domain string) (mtypes.TagLimits, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/limits/tag")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var resp mtypes.TagLimits
	err := getResponseFromJSON(ctx, r, &resp)
	return resp, err
}
