package mailgun

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ValidateEmail performs various checks on the email address provided to ensure it's correctly formatted.
// It may also be used to break an email address into its sub-components.
// https://documentation.mailgun.com/docs/validate/single-valid-ir/
func (mg *Client) ValidateEmail(ctx context.Context, email string, providerLookup bool) (mtypes.ValidateEmailResponse, error) {
	r := newHTTPRequest(fmt.Sprintf("%s/v4/address/validate", mg.APIBase()))
	r.setClient(mg.HTTPClient())
	r.addParameter("address", email)
	r.addParameter("provider_lookup", strconv.FormatBool(providerLookup))
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var res mtypes.ValidateEmailResponse
	err := getResponseFromJSON(ctx, r, &res)
	if err != nil {
		return mtypes.ValidateEmailResponse{}, err
	}

	return res, nil
}
