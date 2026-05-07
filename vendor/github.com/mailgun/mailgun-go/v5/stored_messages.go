package mailgun

import (
	"context"
	"errors"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// GetStoredMessage retrieves information about a received e-mail message.
// This provides visibility into, e.g., replies to a message sent to a mailing list.
func (mg *Client) GetStoredMessage(ctx context.Context, url string) (mtypes.StoredMessage, error) {
	r := newHTTPRequest(url)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var response mtypes.StoredMessage
	err := getResponseFromJSON(ctx, r, &response)
	return response, err
}

// ReSend given a storage url resend the stored message to the specified recipients
func (mg *Client) ReSend(ctx context.Context, url string, recipients ...string) (mtypes.SendMessageResponse, error) {
	var resp mtypes.SendMessageResponse

	r := newHTTPRequest(url)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := NewFormDataPayload()

	if len(recipients) == 0 {
		return resp, errors.New("must provide at least one recipient")
	}

	for _, to := range recipients {
		payload.addValue("to", to)
	}

	err := postResponseFromJSON(ctx, r, payload, &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetStoredMessageRaw retrieves the raw MIME body of a received e-mail message.
// Compared to GetStoredMessage, it gives access to the unparsed MIME body, and
// thus delegates to the caller the required parsing.
func (mg *Client) GetStoredMessageRaw(ctx context.Context, url string) (mtypes.StoredMessageRaw, error) {
	r := newHTTPRequest(url)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.addHeader("Accept", "message/rfc2822")

	var response mtypes.StoredMessageRaw
	err := getResponseFromJSON(ctx, r, &response)
	return response, err
}

// GetStoredAttachment retrieves the raw MIME body of a received e-mail message attachment.
func (mg *Client) GetStoredAttachment(ctx context.Context, url string) ([]byte, error) {
	r := newHTTPRequest(url)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.addHeader("Accept", "message/rfc2822")

	response, err := makeGetRequest(ctx, r)
	if err != nil {
		return nil, err
	}

	return response.Data, err
}
