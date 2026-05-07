package slack

import (
	"context"
	"net/url"
)

// RotateTokens exchanges a refresh token for a new app configuration token.
// For more information see the RotateTokensContext documentation.
func (api *Client) RotateTokens(configToken string, refreshToken string) (*TokenResponse, error) {
	return api.RotateTokensContext(context.Background(), configToken, refreshToken)
}

// RotateTokensContext exchanges a refresh token for a new app configuration token with a custom context.
// Slack API docs: https://api.slack.com/methods/tooling.tokens.rotate
func (api *Client) RotateTokensContext(ctx context.Context, configToken string, refreshToken string) (*TokenResponse, error) {
	if configToken == "" {
		configToken = api.configToken
	}

	if refreshToken == "" {
		refreshToken = api.configRefreshToken
	}

	values := url.Values{
		"refresh_token": {refreshToken},
	}

	response := &TokenResponse{}
	err := api.getMethod(ctx, "tooling.tokens.rotate", configToken, values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// UpdateConfigTokens replaces the configuration tokens in the client with those returned by the API
func (api *Client) UpdateConfigTokens(response *TokenResponse) {
	api.configToken = response.Token
	api.configRefreshToken = response.RefreshToken
}

type TokenResponse struct {
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TeamId       string `json:"team_id,omitempty"`
	UserId       string `json:"user_id,omitempty"`
	IssuedAt     uint64 `json:"iat,omitempty"`
	ExpiresAt    uint64 `json:"exp,omitempty"`
	SlackResponse
}
