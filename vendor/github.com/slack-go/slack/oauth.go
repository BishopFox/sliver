package slack

import (
	"context"
	"net/url"
)

// OAuthResponseIncomingWebhook ...
type OAuthResponseIncomingWebhook struct {
	URL              string `json:"url"`
	Channel          string `json:"channel"`
	ChannelID        string `json:"channel_id,omitempty"`
	ConfigurationURL string `json:"configuration_url"`
}

// OAuthResponseBot ...
type OAuthResponseBot struct {
	BotUserID      string `json:"bot_user_id"`
	BotAccessToken string `json:"bot_access_token"`
}

// OAuthResponse ...
type OAuthResponse struct {
	AccessToken     string                       `json:"access_token"`
	Scope           string                       `json:"scope"`
	TeamName        string                       `json:"team_name"`
	TeamID          string                       `json:"team_id"`
	IncomingWebhook OAuthResponseIncomingWebhook `json:"incoming_webhook"`
	Bot             OAuthResponseBot             `json:"bot"`
	UserID          string                       `json:"user_id,omitempty"`
	SlackResponse
}

// OAuthV2Response ...
type OAuthV2Response struct {
	AccessToken         string                       `json:"access_token"`
	TokenType           string                       `json:"token_type"`
	Scope               string                       `json:"scope"`
	BotUserID           string                       `json:"bot_user_id"`
	AppID               string                       `json:"app_id"`
	Team                OAuthV2ResponseTeam          `json:"team"`
	IncomingWebhook     OAuthResponseIncomingWebhook `json:"incoming_webhook"`
	Enterprise          OAuthV2ResponseEnterprise    `json:"enterprise"`
	IsEnterpriseInstall bool                         `json:"is_enterprise_install"`
	AuthedUser          OAuthV2ResponseAuthedUser    `json:"authed_user"`
	RefreshToken        string                       `json:"refresh_token"`
	ExpiresIn           int                          `json:"expires_in"`
	SlackResponse
}

// OAuthV2ResponseTeam ...
type OAuthV2ResponseTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OAuthV2ResponseEnterprise ...
type OAuthV2ResponseEnterprise struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OAuthV2ResponseAuthedUser ...
type OAuthV2ResponseAuthedUser struct {
	ID           string `json:"id"`
	Scope        string `json:"scope"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// OpenIDConnectResponse ...
type OpenIDConnectResponse struct {
	Ok          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	IdToken     string `json:"id_token"`
	SlackResponse
}

// GetOAuthToken retrieves an AccessToken.
// For more details, see GetOAuthTokenContext documentation.
func GetOAuthToken(client httpClient, clientID, clientSecret, code, redirectURI string) (accessToken string, scope string, err error) {
	return GetOAuthTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI)
}

// GetOAuthTokenContext retrieves an AccessToken with a custom context.
// For more details, see GetOAuthResponseContext documentation.
func GetOAuthTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string) (accessToken string, scope string, err error) {
	response, err := GetOAuthResponseContext(ctx, client, clientID, clientSecret, code, redirectURI)
	if err != nil {
		return "", "", err
	}
	return response.AccessToken, response.Scope, nil
}

// GetBotOAuthToken retrieves top-level and bot AccessToken - https://api.slack.com/legacy/oauth#bot_user_access_tokens
// For more details, see GetBotOAuthTokenContext documentation.
func GetBotOAuthToken(client httpClient, clientID, clientSecret, code, redirectURI string) (accessToken string, scope string, bot OAuthResponseBot, err error) {
	return GetBotOAuthTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI)
}

// GetBotOAuthTokenContext retrieves top-level and bot AccessToken with a custom context.
// For more details, see GetOAuthResponseContext documentation.
func GetBotOAuthTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string) (accessToken string, scope string, bot OAuthResponseBot, err error) {
	response, err := GetOAuthResponseContext(ctx, client, clientID, clientSecret, code, redirectURI)
	if err != nil {
		return "", "", OAuthResponseBot{}, err
	}
	return response.AccessToken, response.Scope, response.Bot, nil
}

// GetOAuthResponse retrieves OAuth response.
// For more details, see GetOAuthResponseContext documentation.
func GetOAuthResponse(client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OAuthResponse, err error) {
	return GetOAuthResponseContext(context.Background(), client, clientID, clientSecret, code, redirectURI)
}

// GetOAuthResponseContext retrieves OAuth response with custom context.
// Slack API docs: https://api.slack.com/methods/oauth.access
func GetOAuthResponseContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OAuthResponse, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	response := &OAuthResponse{}
	if err = postForm(ctx, client, APIURL+"oauth.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GetOAuthV2Response gets a V2 OAuth access token response.
// For more details, see GetOAuthV2ResponseContext documentation.
func GetOAuthV2Response(client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OAuthV2Response, err error) {
	return GetOAuthV2ResponseContext(context.Background(), client, clientID, clientSecret, code, redirectURI)
}

// GetOAuthV2ResponseContext with a context, gets a V2 OAuth access token response.
// Slack API docs: https://api.slack.com/methods/oauth.v2.access
func GetOAuthV2ResponseContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OAuthV2Response, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	response := &OAuthV2Response{}
	if err = postForm(ctx, client, APIURL+"oauth.v2.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// RefreshOAuthV2Token with a context, gets a V2 OAuth access token response.
// For more details, see RefreshOAuthV2TokenContext documentation.
func RefreshOAuthV2Token(client httpClient, clientID, clientSecret, refreshToken string) (resp *OAuthV2Response, err error) {
	return RefreshOAuthV2TokenContext(context.Background(), client, clientID, clientSecret, refreshToken)
}

// RefreshOAuthV2TokenContext with a context, gets a V2 OAuth access token response.
// Slack API docs: https://api.slack.com/methods/oauth.v2.access
func RefreshOAuthV2TokenContext(ctx context.Context, client httpClient, clientID, clientSecret, refreshToken string) (resp *OAuthV2Response, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	response := &OAuthV2Response{}
	if err = postForm(ctx, client, APIURL+"oauth.v2.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GetOpenIDConnectToken exchanges a temporary OAuth verifier code for an access token for Sign in with Slack.
// For more details, see GetOpenIDConnectTokenContext documentation.
func GetOpenIDConnectToken(client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OpenIDConnectResponse, err error) {
	return GetOpenIDConnectTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI)
}

// GetOpenIDConnectTokenContext with a context, gets an access token for Sign in with Slack.
// Slack API docs: https://api.slack.com/methods/openid.connect.token
func GetOpenIDConnectTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string) (resp *OpenIDConnectResponse, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	response := &OpenIDConnectResponse{}
	if err = postForm(ctx, client, APIURL+"openid.connect.token", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}
