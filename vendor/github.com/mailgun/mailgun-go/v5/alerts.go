package mailgun

// https://documentation.mailgun.com/docs/inboxready/api-reference/optimize/inboxready/alerts

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListAlertsEventsOptions struct{}

// ListAlertsEvents list of events that you can choose to receive alerts for.
func (mg *Client) ListAlertsEvents(ctx context.Context, _ *ListAlertsEventsOptions,
) (*mtypes.AlertsEventsResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.AlertsVersion, mtypes.AlertsEndpoint))
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())

	var resp mtypes.AlertsEventsResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type ListAlertsOptions struct{}

// ListAlerts returns a list of all configured alert settings for your account.
func (mg *Client) ListAlerts(ctx context.Context, _ *ListAlertsOptions,
) (*mtypes.AlertsSettingsResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.AlertsVersion, mtypes.AlertsSettingsEndpoint))
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())

	var resp mtypes.AlertsSettingsResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (mg *Client) AddAlert(ctx context.Context, req mtypes.AlertsEventSettingRequest,
) (*mtypes.AlertsEventSettingResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.AlertsVersion, mtypes.AlertsSettingsEndpoint))
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())

	payload := newJSONEncodedPayload(req)
	var resp mtypes.AlertsEventSettingResponse
	if err := postResponseFromJSON(ctx, r, payload, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (mg *Client) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.AlertsVersion, mtypes.AlertsSettingsEndpoint+"/"+id.String()))
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())

	_, err := makeDeleteRequest(ctx, r)

	return err
}

// VerifyAlertsWebhookSignFromRequest verifies if the request was sent from the Mailgun.
// This is optional.
// Alerts webhooks are using another method to validate the webhook, not same as Mailgun Send webhooks.
//
// `webhookSigningKey` - is a Webhooks.SigningKey from (*Client).ListAlerts (GET /v1/alerts/settings) response.
func VerifyAlertsWebhookSignFromRequest(r *http.Request, webhookSigningKey string) (verified bool, err error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("reading request body: %w", err)
	}

	// put the body back to the request so it can be read again later
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	signHeader := r.Header.Get(mtypes.AlertsWebhookSignHeader)

	return VerifyAlertsWebhookSign(body, signHeader, webhookSigningKey)
}

// VerifyAlertsWebhookSign verifies if the request was sent from the Mailgun.
// This is optional.
// Alerts webhooks are using another method to validate the webhook, not same as Mailgun Send webhooks.
//
// `body` is a raw HTTP request body sent by Mailgun to your webhook URL.
//
// `signHeader` is an "X-Sign" header from the Mailgun request.
//
// `webhookSigningKey` - is a Webhooks.SigningKey from (*Client).ListAlerts (GET /v1/alerts/settings) response.
func VerifyAlertsWebhookSign(body []byte, signHeader, webhookSigningKey string) (verified bool, err error) {
	calculatedSignature, err := CalcAlertsHMAC(body, webhookSigningKey)
	if err != nil {
		return false, fmt.Errorf("calculating HMAC: %w", err)
	}

	signature, err := hex.DecodeString(signHeader)
	if err != nil {
		return false, fmt.Errorf("invalid sign: %w", err)
	}

	return subtle.ConstantTimeCompare(signature, calculatedSignature) == 1, nil
}

// CalcAlertsHMAC calculates Alerts webhook HMAC.
// Alerts webhooks are using another method to validate the webhook, not the same as Mailgun Send webhooks.
//
// `body` is a raw HTTP request body sent by Mailgun to your webhook URL.
//
// `webhookSigningKey` - is a Webhooks.SigningKey from (*Client).ListAlerts (GET /v1/alerts/settings) response.
func CalcAlertsHMAC(body []byte, webhookSigningKey string) (sign []byte, err error) {
	if webhookSigningKey == "" {
		return nil, fmt.Errorf("webhook signing key is not set")
	}

	h := hmac.New(sha256.New, []byte(webhookSigningKey))
	_, err = h.Write(body)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
