// Copyright 2020 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"context"
	"net/url"
	"strings"
)

const clientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// IssueAccessTokenV2 method
func (client *Client) IssueAccessTokenV2(clientAssertion string) *IssueAccessTokenV2Call {
	return &IssueAccessTokenV2Call{
		c:               client,
		clientAssertion: clientAssertion,
	}
}

// IssueAccessTokenV2Call type
type IssueAccessTokenV2Call struct {
	c   *Client
	ctx context.Context

	clientAssertion string
}

// WithContext method
func (call *IssueAccessTokenV2Call) WithContext(ctx context.Context) *IssueAccessTokenV2Call {
	call.ctx = ctx
	return call
}

// Do method
func (call *IssueAccessTokenV2Call) Do() (*AccessTokenResponse, error) {
	vs := url.Values{}
	vs.Set("grant_type", "client_credentials")
	vs.Set("client_assertion_type", clientAssertionTypeJWT)
	vs.Set("client_assertion", call.clientAssertion)
	body := strings.NewReader(vs.Encode())

	res, err := call.c.postform(call.ctx, APIEndpointIssueAccessTokenV2, body)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToAccessTokenResponse(res)
}

// GetAccessTokensV2 method
func (client *Client) GetAccessTokensV2(clientAssertion string) *GetAccessTokensV2Call {
	return &GetAccessTokensV2Call{
		c:               client,
		clientAssertion: clientAssertion,
	}
}

// GetAccessTokensV2Call type
type GetAccessTokensV2Call struct {
	c   *Client
	ctx context.Context

	clientAssertion string
}

// WithContext method
func (call *GetAccessTokensV2Call) WithContext(ctx context.Context) *GetAccessTokensV2Call {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetAccessTokensV2Call) Do() (*AccessTokensResponse, error) {
	vs := url.Values{}
	vs.Set("client_assertion_type", clientAssertionTypeJWT)
	vs.Set("client_assertion", call.clientAssertion)

	res, err := call.c.get(call.ctx, call.c.endpointBase, APIEndpointGetAccessTokensV2, vs)
	//	body := strings.NewReader(vs.Encode())

	//	res, err := call.c.postform(call.ctx, APIEndpointGetAccessTokensV2, body)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToAccessTokensResponse(res)
}

// RevokeAccessTokenV2 method
func (client *Client) RevokeAccessTokenV2(channelID, channelSecret, accessToken string) *RevokeAccessTokenV2Call {
	return &RevokeAccessTokenV2Call{
		c:             client,
		accessToken:   accessToken,
		channelID:     channelID,
		channelSecret: channelSecret,
	}
}

// RevokeAccessTokenV2Call type
type RevokeAccessTokenV2Call struct {
	c   *Client
	ctx context.Context

	accessToken   string
	channelID     string
	channelSecret string
}

// WithContext method
func (call *RevokeAccessTokenV2Call) WithContext(ctx context.Context) *RevokeAccessTokenV2Call {
	call.ctx = ctx
	return call
}

// Do method
func (call *RevokeAccessTokenV2Call) Do() (*BasicResponse, error) {
	vs := url.Values{}
	vs.Set("access_token", call.accessToken)
	vs.Set("client_id", call.channelID)
	vs.Set("client_secret", call.channelSecret)
	body := strings.NewReader(vs.Encode())

	res, err := call.c.postform(call.ctx, APIEndpointRevokeAccessTokenV2, body)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}
