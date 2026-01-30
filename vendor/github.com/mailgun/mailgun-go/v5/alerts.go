package mailgun

// https://documentation.mailgun.com/docs/inboxready/openapi-final/tag/Alerts/

import (
	"context"

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
