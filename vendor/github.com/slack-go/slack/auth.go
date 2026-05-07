package slack

import (
	"context"
	"net/url"
	"strconv"
)

// AuthRevokeResponse contains our Auth response from the auth.revoke endpoint
type AuthRevokeResponse struct {
	SlackResponse      // Contains the "ok", and "Error", if any
	Revoked       bool `json:"revoked,omitempty"`
}

// authRequest sends the actual request, and unmarshals the response
func (api *Client) authRequest(ctx context.Context, path string, values url.Values) (*AuthRevokeResponse, error) {
	response := &AuthRevokeResponse{}
	err := api.postMethod(ctx, path, values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// SendAuthRevoke will send a revocation for our token.
// For more details, see SendAuthRevokeContext documentation.
func (api *Client) SendAuthRevoke(token string) (*AuthRevokeResponse, error) {
	return api.SendAuthRevokeContext(context.Background(), token)
}

// SendAuthRevokeContext will send a revocation request for our token to api.revoke with a custom context.
// Slack API docs: https://api.slack.com/methods/auth.revoke
func (api *Client) SendAuthRevokeContext(ctx context.Context, token string) (*AuthRevokeResponse, error) {
	if token == "" {
		token = api.token
	}
	values := url.Values{
		"token": {token},
	}

	return api.authRequest(ctx, "auth.revoke", values)
}

type listTeamsResponse struct {
	Teams []Team `json:"teams"`
	SlackResponse
}

type ListTeamsParameters struct {
	Limit       int
	Cursor      string
	IncludeIcon *bool
}

// ListTeams returns all workspaces a token can access.
// For more details, see ListTeamsContext documentation.
func (api *Client) ListTeams(params ListTeamsParameters) ([]Team, string, error) {
	return api.ListTeamsContext(context.Background(), params)
}

// ListTeamsContext returns all workspaces a token can access with a custom context.
// Slack API docs: https://api.slack.com/methods/auth.teams.list
func (api *Client) ListTeamsContext(ctx context.Context, params ListTeamsParameters) ([]Team, string, error) {
	values := url.Values{
		"token": {api.token},
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}
	if params.IncludeIcon != nil {
		values.Add("include_icon", strconv.FormatBool(*params.IncludeIcon))
	}

	response := &listTeamsResponse{}
	err := api.postMethod(ctx, "auth.teams.list", values, response)
	if err != nil {
		return nil, "", err
	}

	return response.Teams, response.ResponseMetadata.Cursor, response.Err()
}
