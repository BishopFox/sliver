package mailgun

import (
	"context"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListAPIKeysOptions struct {
	DomainName string
	Kind       string
}

type CreateAPIKeyOptions struct {
	Description string
	DomainName  string
	Email       string
	Expiration  uint64 // The key's lifetime in seconds.
	Kind        string
	UserID      string
	UserName    string
}

func (mg *Client) ListAPIKeys(ctx context.Context, opts *ListAPIKeysOptions) ([]mtypes.APIKey, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.APIKeysVersion, mtypes.APIKeysEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	if opts != nil {
		if opts.DomainName != "" {
			r.addParameter("domain_name", opts.DomainName)
		}

		if opts.Kind != "" {
			r.addParameter("kind", opts.Kind)
		}
	}

	var resp mtypes.GetAPIKeyListResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}

	return resp.Items, nil
}

func (mg *Client) CreateAPIKey(ctx context.Context, role string, opts *CreateAPIKeyOptions) (mtypes.APIKey, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.APIKeysVersion, mtypes.APIKeysEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("role", role)

	if opts != nil {
		if opts.Description != "" {
			payload.addValue("description", opts.Description)
		}

		if opts.DomainName != "" {
			payload.addValue("domain_name", opts.DomainName)
		}

		if opts.Email != "" {
			payload.addValue("email", opts.Email)
		}

		if opts.Expiration != 0 {
			payload.addValue("expiration", strconv.FormatUint(opts.Expiration, 10))
		}

		if opts.Kind != "" {
			payload.addValue("kind", opts.Kind)
		}

		if opts.UserID != "" {
			payload.addValue("user_id", opts.UserID)
		}

		if opts.UserName != "" {
			payload.addValue("user_name", opts.UserName)
		}
	}

	var resp mtypes.CreateAPIKeyResponse
	err := postResponseFromJSON(ctx, r, payload, &resp)
	return resp.Key, err
}

func (mg *Client) DeleteAPIKey(ctx context.Context, id string) error {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.APIKeysVersion, mtypes.APIKeysEndpoint+"/"+id))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	_, err := makeDeleteRequest(ctx, r)
	return err
}

func (mg *Client) RegeneratePublicAPIKey(ctx context.Context) (mtypes.RegeneratePublicAPIKeyResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.APIKeysVersion, mtypes.APIKeysRegenerateEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.RegeneratePublicAPIKeyResponse
	err := postResponseFromJSON(ctx, r, nil, &resp)
	return resp, err
}
