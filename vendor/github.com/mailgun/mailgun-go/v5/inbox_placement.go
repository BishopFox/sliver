package mailgun

import (
	"context"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// CreateInboxPlacementTest runs Inbox Placement Test
// https://documentation.mailgun.com/docs/inboxready/api-reference/optimize/inboxready/inbox-placement/post-v4-inbox-tests
func (mg *Client) CreateInboxPlacementTest(ctx context.Context, opts mtypes.CreateInboxPlacementTestOptions,
) (*mtypes.CreateInboxPlacementTestResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, mtypes.InboxPlacementVersion, mtypes.InboxPlacementTestsEndpoint))
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())

	payload := newJSONEncodedPayload(opts)
	var resp mtypes.CreateInboxPlacementTestResponse
	if err := postResponseFromJSON(ctx, r, payload, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
