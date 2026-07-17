package slack

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

type oauthConfig struct {
	apiURL       string
	codeVerifier string
}

// OAuthOption configures package-level OAuth functions.
type OAuthOption func(*oauthConfig)

// OAuthOptionAPIURL overrides the default Slack API URL. Useful for testing.
func OAuthOptionAPIURL(url string) OAuthOption {
	return func(c *oauthConfig) { c.apiURL = url }
}

// OAuthOptionCodeVerifier sets the PKCE code_verifier for the OAuth token exchange.
// Use this when your authorization request included a code_challenge.
func OAuthOptionCodeVerifier(verifier string) OAuthOption {
	return func(c *oauthConfig) { c.codeVerifier = verifier }
}

func resolveOAuthConfig(opts []OAuthOption) oauthConfig {
	c := oauthConfig{apiURL: APIURL}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func resolveOAuthAPIURL(opts []OAuthOption) string {
	return resolveOAuthConfig(opts).apiURL
}

// GetOAuthToken retrieves an AccessToken.
// For more details, see GetOAuthTokenContext documentation.
func GetOAuthToken(client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (accessToken string, scope string, err error) {
	return GetOAuthTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI, opts...)
}

// GetOAuthTokenContext retrieves an AccessToken with a custom context.
// For more details, see GetOAuthResponseContext documentation.
func GetOAuthTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (accessToken string, scope string, err error) {
	response, err := GetOAuthResponseContext(ctx, client, clientID, clientSecret, code, redirectURI, opts...)
	if err != nil {
		return "", "", err
	}
	return response.AccessToken, response.Scope, nil
}

// GetBotOAuthToken retrieves top-level and bot AccessToken - https://api.slack.com/legacy/oauth#bot_user_access_tokens
// For more details, see GetBotOAuthTokenContext documentation.
func GetBotOAuthToken(client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (accessToken string, scope string, bot OAuthResponseBot, err error) {
	return GetBotOAuthTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI, opts...)
}

// GetBotOAuthTokenContext retrieves top-level and bot AccessToken with a custom context.
// For more details, see GetOAuthResponseContext documentation.
func GetBotOAuthTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (accessToken string, scope string, bot OAuthResponseBot, err error) {
	response, err := GetOAuthResponseContext(ctx, client, clientID, clientSecret, code, redirectURI, opts...)
	if err != nil {
		return "", "", OAuthResponseBot{}, err
	}
	return response.AccessToken, response.Scope, response.Bot, nil
}

// GetOAuthResponse retrieves OAuth response.
// For more details, see GetOAuthResponseContext documentation.
func GetOAuthResponse(client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OAuthResponse, err error) {
	return GetOAuthResponseContext(context.Background(), client, clientID, clientSecret, code, redirectURI, opts...)
}

// GetOAuthResponseContext retrieves OAuth response with custom context.
// Slack API docs: https://api.slack.com/methods/oauth.access
func GetOAuthResponseContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OAuthResponse, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	response := &OAuthResponse{}
	if _, err = postForm(ctx, client, resolveOAuthAPIURL(opts)+"oauth.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GetOAuthV2Response gets a V2 OAuth access token response.
// For more details, see GetOAuthV2ResponseContext documentation.
func GetOAuthV2Response(client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OAuthV2Response, err error) {
	return GetOAuthV2ResponseContext(context.Background(), client, clientID, clientSecret, code, redirectURI, opts...)
}

// GetOAuthV2ResponseContext with a context, gets a V2 OAuth access token response.
// For PKCE flows, pass OAuthOptionCodeVerifier and an empty clientSecret.
// Slack API docs: https://api.slack.com/methods/oauth.v2.access
func GetOAuthV2ResponseContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OAuthV2Response, err error) {
	cfg := resolveOAuthConfig(opts)
	values := url.Values{
		"client_id":    {clientID},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}
	if cfg.codeVerifier != "" {
		values.Set("code_verifier", cfg.codeVerifier)
	}
	response := &OAuthV2Response{}
	if _, err = postForm(ctx, client, cfg.apiURL+"oauth.v2.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// RefreshOAuthV2Token with a context, gets a V2 OAuth access token response.
// For more details, see RefreshOAuthV2TokenContext documentation.
func RefreshOAuthV2Token(client httpClient, clientID, clientSecret, refreshToken string, opts ...OAuthOption) (resp *OAuthV2Response, err error) {
	return RefreshOAuthV2TokenContext(context.Background(), client, clientID, clientSecret, refreshToken, opts...)
}

// RefreshOAuthV2TokenContext with a context, gets a V2 OAuth access token response.
// For PKCE public clients, pass an empty clientSecret.
// Slack API docs: https://api.slack.com/methods/oauth.v2.access
func RefreshOAuthV2TokenContext(ctx context.Context, client httpClient, clientID, clientSecret, refreshToken string, opts ...OAuthOption) (resp *OAuthV2Response, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}
	response := &OAuthV2Response{}
	if _, err = postForm(ctx, client, resolveOAuthAPIURL(opts)+"oauth.v2.access", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// OpenIDConnectUserInfoResponse contains the response from openid.connect.userInfo.
//
// Some of the fields in the response to this method are preceded with https://slack.com/.
// These fields are Slack-specific, and they're from the perspective of Slack.
type OpenIDConnectUserInfoResponse struct {
	Ok bool `json:"ok"`

	Sub string `json:"sub"`

	UserID string `json:"https://slack.com/user_id"`
	TeamID string `json:"https://slack.com/team_id"`

	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	DateEmailVerified int64  `json:"date_email_verified"`

	Name       string `json:"name"`
	Picture    string `json:"picture"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Locale     string `json:"locale"`

	TeamName     string `json:"https://slack.com/team_name"`
	TeamDomain   string `json:"https://slack.com/team_domain"`
	TeamImage34  string `json:"https://slack.com/team_image_34"`
	TeamImage44  string `json:"https://slack.com/team_image_44"`
	TeamImage68  string `json:"https://slack.com/team_image_68"`
	TeamImage88  string `json:"https://slack.com/team_image_88"`
	TeamImage102 string `json:"https://slack.com/team_image_102"`
	TeamImage132 string `json:"https://slack.com/team_image_132"`
	TeamImage230 string `json:"https://slack.com/team_image_230"`

	// `TeamImageDefault` indicates whether the image is a default one (true), or someone
	// uploaded their own (false).
	TeamImageDefault bool `json:"https://slack.com/team_image_default"`

	UserImage24       string `json:"https://slack.com/user_image_24"`
	UserImage32       string `json:"https://slack.com/user_image_32"`
	UserImage48       string `json:"https://slack.com/user_image_48"`
	UserImage72       string `json:"https://slack.com/user_image_72"`
	UserImage192      string `json:"https://slack.com/user_image_192"`
	UserImage512      string `json:"https://slack.com/user_image_512"`
	UserImage1024     string `json:"https://slack.com/user_image_1024"`
	UserImageOriginal string `json:"https://slack.com/user_image_original"`

	SlackResponse
}

// GetOpenIDConnectUserInfo returns the user info for the token.
// For more details, see GetOpenIDConnectUserInfoContext documentation.
func (api *Client) GetOpenIDConnectUserInfo() (*OpenIDConnectUserInfoResponse, error) {
	return api.GetOpenIDConnectUserInfoContext(context.Background())
}

// GetOpenIDConnectUserInfoContext returns identity information about the user associated with the token.
// Slack API docs: https://docs.slack.dev/reference/methods/openid.connect.userInfo
func (api *Client) GetOpenIDConnectUserInfoContext(ctx context.Context) (*OpenIDConnectUserInfoResponse, error) {
	values := url.Values{
		"token": {api.token},
	}
	response := &OpenIDConnectUserInfoResponse{}
	err := api.postMethod(ctx, "openid.connect.userInfo", values, response)
	if err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GetOpenIDConnectToken exchanges a temporary OAuth verifier code for an access token for Sign in with Slack.
// For more details, see GetOpenIDConnectTokenContext documentation.
func GetOpenIDConnectToken(client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OpenIDConnectResponse, err error) {
	return GetOpenIDConnectTokenContext(context.Background(), client, clientID, clientSecret, code, redirectURI, opts...)
}

// GetOpenIDConnectTokenContext with a context, gets an access token for Sign in with Slack.
// Slack API docs: https://api.slack.com/methods/openid.connect.token
func GetOpenIDConnectTokenContext(ctx context.Context, client httpClient, clientID, clientSecret, code, redirectURI string, opts ...OAuthOption) (resp *OpenIDConnectResponse, err error) {
	values := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	response := &OpenIDConnectResponse{}
	if _, err = postForm(ctx, client, resolveOAuthAPIURL(opts)+"openid.connect.token", values, response, discard{}); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GenerateCodeVerifier creates a cryptographically random PKCE code verifier
// string suitable for use with OAuth 2.0 PKCE flows. The returned string is
// 43 characters of URL-safe base64 (no padding).
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateCodeChallenge creates a PKCE code challenge from a code verifier
// using the S256 method (SHA-256 hash, base64url-encoded without padding).
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
