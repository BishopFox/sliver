package mailgun

// https://documentation.mailgun.com/docs/mailgun/api-reference/openapi-final/tag/Webhooks/#tag/Webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ListWebhooks returns the complete set of webhooks configured for your domain.
// Note that a zero-length mapping is not an error.
func (mg *Client) ListWebhooks(ctx context.Context, domain string) (map[string][]string, error) {
	r := newHTTPRequest(generateV3DomainsApiUrl(mg, webhooksEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var body mtypes.WebHooksListResponse
	err := getResponseFromJSON(ctx, r, &body)
	if err != nil {
		return nil, err
	}

	hooks := make(map[string][]string, 0)
	for k, v := range body.Webhooks {
		if v.Url != "" {
			hooks[k] = []string{v.Url}
		}
		if len(v.Urls) != 0 {
			hooks[k] = append(hooks[k], v.Urls...)
		}
	}
	return hooks, nil
}

// CreateWebhook installs a new webhook for your domain.
func (mg *Client) CreateWebhook(ctx context.Context, domain, id string, urls []string) error {
	r := newHTTPRequest(generateV3DomainsApiUrl(mg, webhooksEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	p.addValue("id", id)
	for _, url := range urls {
		p.addValue("url", url)
	}
	_, err := makePostRequest(ctx, r, p)
	return err
}

// DeleteWebhook removes the specified webhook from your domain's configuration.
func (mg *Client) DeleteWebhook(ctx context.Context, domain, name string) error {
	r := newHTTPRequest(generateV3DomainsApiUrl(mg, webhooksEndpoint, domain) + "/" + name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// GetWebhook retrieves the currently assigned webhook URL associated with the provided type of webhook.
func (mg *Client) GetWebhook(ctx context.Context, domain, name string) ([]string, error) {
	r := newHTTPRequest(generateV3DomainsApiUrl(mg, webhooksEndpoint, domain) + "/" + name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var body mtypes.WebHookResponse
	if err := getResponseFromJSON(ctx, r, &body); err != nil {
		return nil, err
	}

	if body.Webhook.Url != "" {
		return []string{body.Webhook.Url}, nil
	}
	if len(body.Webhook.Urls) != 0 {
		return body.Webhook.Urls, nil
	}
	return nil, fmt.Errorf("webhook '%s' returned no urls", name)
}

// UpdateWebhook replaces one webhook setting for another.
func (mg *Client) UpdateWebhook(ctx context.Context, domain, name string, urls []string) error {
	r := newHTTPRequest(generateV3DomainsApiUrl(mg, webhooksEndpoint, domain) + "/" + name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	for _, url := range urls {
		p.addValue("url", url)
	}
	_, err := makePutRequest(ctx, r, p)
	return err
}

// VerifyWebhookSignature - use this method to parse the webhook signature given as JSON in the webhook response
func (mg *Client) VerifyWebhookSignature(sig mtypes.Signature) (verified bool, err error) {
	webhookSigningKey := mg.WebhookSigningKey()
	if webhookSigningKey == "" {
		return false, fmt.Errorf("webhook signing key is not set")
	}

	h := hmac.New(sha256.New, []byte(webhookSigningKey))

	_, err = io.WriteString(h, sig.TimeStamp)
	if err != nil {
		return false, err
	}
	_, err = io.WriteString(h, sig.Token)
	if err != nil {
		return false, err
	}

	calculatedSignature := h.Sum(nil)
	signature, err := hex.DecodeString(sig.Signature)
	if err != nil {
		return false, err
	}
	if len(calculatedSignature) != len(signature) {
		return false, nil
	}

	return subtle.ConstantTimeCompare(signature, calculatedSignature) == 1, nil
}
