// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mautrix

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/random"

	"maunium.net/go/mautrix/oauth"
)

const (
	oauthRequestTypeMetadata       = "metadata"
	oauthRequestTypeRegister       = "register"
	oauthRequestTypeRefresh        = "refresh"
	oauthRequestTypeExchangeCode   = "exchangecode"
	oauthRequestTypeGetDeviceCode  = "devicecode"
	oauthRequestTypePollDeviceCode = "devicecode"
	oauthRequestTypeRevoke         = "revoke"
)

func (cli *Client) makeOAuthRequest(ctx context.Context, reqType, url string, payload url.Values, resp any) (err error) {
	reqParams := FullRequest{
		Method:       http.MethodPost,
		URL:          url,
		RequestBytes: []byte(payload.Encode()),
		Headers:      http.Header{},
		ResponseJSON: resp,
	}
	if payload != nil {
		reqParams.RequestBytes = []byte(payload.Encode())
		reqParams.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	_, err = cli.MakeFullRequest(context.WithValue(ctx, oauthReqContextKey, reqType), reqParams)
	return err
}

func (cli *Client) OAuthGetServerMetadata(ctx context.Context) (meta *oauth.ServerMetadata, err error) {
	cli.oauthMetadataLock.Lock()
	defer cli.oauthMetadataLock.Unlock()
	if cli.oauthMetadata != nil {
		return cli.oauthMetadata, nil
	}
	urlPath := cli.BuildClientURL("v1", "auth_metadata")
	ctx = context.WithValue(ctx, oauthReqContextKey, oauthRequestTypeMetadata)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &meta)
	if meta != nil {
		cli.oauthMetadata = meta
	}
	return
}

func (cli *Client) OAuthSetServerMetadata(meta *oauth.ServerMetadata) {
	cli.oauthMetadataLock.Lock()
	cli.oauthMetadata = meta
	cli.oauthMetadataLock.Unlock()
}

func (cli *Client) OAuthRegisterClient(ctx context.Context, clientMeta *oauth.ClientMetadata) (resp *oauth.ClientMetadata, err error) {
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return
	}
	_, err = cli.MakeFullRequest(context.WithValue(ctx, oauthReqContextKey, oauthRequestTypeRegister), FullRequest{
		Method:       http.MethodPost,
		URL:          authMeta.RegistrationEndpoint,
		RequestJSON:  clientMeta,
		ResponseJSON: &resp,
	})
	if resp != nil {
		cli.oauthClientID = resp.ClientID
	}
	return
}

func (cli *Client) OAuthGetAuthorizationURL(ctx context.Context, params oauth.GetAuthorizationURLParams) (*oauth.AuthorizationCodeResponse, error) {
	clientID := cmp.Or(params.ClientID, cli.oauthClientID)
	if clientID == "" {
		return nil, ErrClientIDNotSet
	}

	state := cli.TxnID()
	codeVerifier := base64.RawURLEncoding.EncodeToString(random.Bytes(32))
	codeChallengeBytes := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(codeChallengeBytes[:])

	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	authURL, err := url.Parse(authMeta.AuthorizationEndpoint)
	if err != nil {
		return nil, err
	}
	q := authURL.Query()
	q.Set("response_type", string(oauth.ResponseTypeCode))
	q.Set("client_id", clientID)
	q.Set("redirect_uri", params.RedirectURI)
	q.Set("scope", params.Scopes.String())
	q.Set("state", state)
	q.Set("response_mode", string(params.ResponseMode))
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", string(oauth.CodeChallengeMethodS256))
	if params.UserIDHint != "" {
		q.Set("login_hint", fmt.Sprintf("mxid:%s", params.UserIDHint))
	}
	authURL.RawQuery = q.Encode()
	return &oauth.AuthorizationCodeResponse{
		State:        state,
		CodeVerifier: codeVerifier,
		URL:          authURL.String(),
	}, nil
}

func (cli *Client) OAuthGenerateDeviceCode(ctx context.Context, params oauth.GenerateDeviceCodeParams) (resp *oauth.DeviceCodeResponse, err error) {
	clientID := cmp.Or(params.ClientID, cli.oauthClientID)
	if clientID == "" {
		return nil, ErrClientIDNotSet
	}
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("scope", params.Scopes.String())
	if params.UserIDHint != "" {
		q.Set("login_hint", fmt.Sprintf("mxid:%s", params.UserIDHint))
	}
	err = cli.makeOAuthRequest(ctx, oauthRequestTypeGetDeviceCode, authMeta.DeviceAuthorizationEndpoint, q, &resp)
	return
}

func (cli *Client) OAuthExchangeToken(ctx context.Context, params oauth.ExchangeTokenParams) (resp *oauth.TokenResponse, err error) {
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	if params.StoreCredentials {
		cli.refreshLock.Lock()
		defer cli.refreshLock.Unlock()
	}
	clientID := cmp.Or(params.ClientID, cli.oauthClientID)
	if clientID == "" {
		return nil, ErrClientIDNotSet
	}

	start := time.Now()
	err = cli.makeOAuthRequest(ctx, oauthRequestTypeExchangeCode, authMeta.TokenEndpoint, url.Values{
		"grant_type":    []string{string(oauth.GrantTypeAuthorizationCode)},
		"code":          []string{params.Code},
		"redirect_uri":  []string{params.RedirectURI},
		"client_id":     []string{clientID},
		"code_verifier": []string{params.CodeVerifier},
	}, &resp)
	if err == nil && params.StoreCredentials {
		cli.refreshToken = resp.RefreshToken
		cli.AccessToken = resp.AccessToken
		cli.accessTokenExpiry = start.Add(resp.ExpiresIn.Duration)
	}
	return
}

func (cli *Client) OAuthPollDeviceCode(ctx context.Context, params oauth.PollDeviceCodeParams) (resp *oauth.TokenResponse, err error) {
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	if params.StoreCredentials {
		cli.refreshLock.Lock()
		defer cli.refreshLock.Unlock()
	}
	clientID := cmp.Or(params.ClientID, cli.oauthClientID)
	if clientID == "" {
		return nil, ErrClientIDNotSet
	}

	start := time.Now()
	err = cli.makeOAuthRequest(ctx, oauthRequestTypePollDeviceCode, authMeta.TokenEndpoint, url.Values{
		"grant_type":  []string{string(oauth.GrantTypeDeviceCode)},
		"device_code": []string{params.DeviceCode},
		"client_id":   []string{clientID},
	}, &resp)
	if err == nil && params.StoreCredentials {
		cli.refreshToken = resp.RefreshToken
		cli.AccessToken = resp.AccessToken
		cli.accessTokenExpiry = start.Add(resp.ExpiresIn.Duration)
		cli.longTokenLifetime = resp.ExpiresIn.Duration > 1*time.Hour
	}
	return
}

const (
	tokenRefreshBuffer     = 10 * time.Second
	syncTokenRefreshBuffer = 60 * time.Second
)

func (cli *Client) OAuthSetTokens(clientID, refreshToken, accessToken string, expiry time.Time) {
	cli.refreshLock.Lock()
	cli.oauthClientID = clientID
	cli.refreshToken = refreshToken
	cli.AccessToken = accessToken
	cli.accessTokenExpiry = expiry
	cli.longTokenLifetime = time.Until(expiry) > 1*time.Hour
	cli.refreshLock.Unlock()
}

func (cli *Client) shouldRetryWithRefreshedToken(ctx context.Context, prevToken string) bool {
	if ctx.Value(oauthReqContextKey) != nil {
		return false
	}
	cli.refreshLock.RLock()
	defer cli.refreshLock.RUnlock()
	return cli.refreshToken != "" && (prevToken != cli.AccessToken || time.Until(cli.accessTokenExpiry) < 0)
}

func (cli *Client) refreshTokenIfNeeded(ctx context.Context, isSync bool) (string, error) {
	buffer := tokenRefreshBuffer
	if isSync {
		buffer = syncTokenRefreshBuffer
	}
	if cli.longTokenLifetime {
		buffer *= 5
	}
	cli.refreshLock.RLock()
	needed := cli.refreshToken != "" && time.Until(cli.accessTokenExpiry) < buffer
	token := cli.AccessToken
	cli.refreshLock.RUnlock()
	if needed {
		err := cli.OAuthRefreshToken(ctx, buffer)
		if err == nil {
			cli.refreshLock.RLock()
			token = cli.AccessToken
			cli.refreshLock.RUnlock()
		}
		return token, err
	}
	return token, nil
}

func (cli *Client) OAuthRefreshToken(ctx context.Context, buffer time.Duration) (err error) {
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return
	}
	cli.refreshLock.Lock()
	defer cli.refreshLock.Unlock()
	if cli.refreshToken == "" {
		return nil
	} else if cli.oauthClientID == "" {
		return ErrClientIDNotSet
	}
	if time.Until(cli.accessTokenExpiry) > buffer {
		zerolog.Ctx(ctx).Debug().
			Time("expires_at", cli.accessTokenExpiry).
			Msg("Not refreshing OAuth token because it is still valid")
		return
	}
	zerolog.Ctx(ctx).Debug().
		Time("expires_at", cli.accessTokenExpiry).
		Msg("Refreshing OAuth token")
	var resp oauth.TokenResponse
	start := time.Now()
	err = cli.makeOAuthRequest(ctx, oauthRequestTypeRefresh, authMeta.TokenEndpoint, url.Values{
		"grant_type":    []string{string(oauth.GrantTypeRefreshToken)},
		"refresh_token": []string{cli.refreshToken},
		"client_id":     []string{cli.oauthClientID},
	}, &resp)
	if err == nil {
		if resp.RefreshToken == "" {
			resp.RefreshToken = cli.refreshToken
		}
		expiry := start.Add(resp.ExpiresIn.Duration)
		err = cli.SaveNewToken(ctx, resp.RefreshToken, resp.AccessToken, expiry)
		if err == nil {
			cli.refreshToken = resp.RefreshToken
			cli.AccessToken = resp.AccessToken
			cli.accessTokenExpiry = expiry
			cli.longTokenLifetime = resp.ExpiresIn.Duration > 1*time.Hour
		}
	}
	return
}

func (cli *Client) OAuthRevokeToken(ctx context.Context) (err error) {
	authMeta, err := cli.OAuthGetServerMetadata(ctx)
	if err != nil {
		return
	}
	cli.refreshLock.Lock()
	defer cli.refreshLock.Unlock()
	values := url.Values{}
	if cli.refreshToken != "" {
		values.Set("token", cli.refreshToken)
		values.Set("token_type_hint", "refresh_token")
	} else if cli.AccessToken != "" {
		values.Set("token", cli.AccessToken)
		values.Set("token_type_hint", "access_token")
	} else {
		return nil
	}
	if clientID := cli.oauthClientID; clientID != "" {
		values.Set("client_id", clientID)
	}
	err = cli.makeOAuthRequest(ctx, oauthRequestTypeRevoke, authMeta.RevocationEndpoint, values, nil)
	if httpErr := (HTTPError{}); errors.As(err, &httpErr) && httpErr.IsStatus(http.StatusUnauthorized) {
		err = nil
	}
	return
}
