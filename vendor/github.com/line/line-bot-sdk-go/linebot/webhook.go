// Copyright 2016 LINE Corporation
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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

// ParseRequest method
func (client *Client) ParseRequest(r *http.Request) ([]*Event, error) {
	return ParseRequest(client.channelSecret, r)
}

// ParseRequest func
func ParseRequest(channelSecret string, r *http.Request) ([]*Event, error) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if !validateSignature(channelSecret, r.Header.Get("X-Line-Signature"), body) {
		return nil, ErrInvalidSignature
	}

	request := &struct {
		Events []*Event `json:"events"`
	}{}
	if err = json.Unmarshal(body, request); err != nil {
		return nil, err
	}
	return request.Events, nil
}

func validateSignature(channelSecret, signature string, body []byte) bool {
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	hash := hmac.New(sha256.New, []byte(channelSecret))

	_, err = hash.Write(body)
	if err != nil {
		return false
	}

	return hmac.Equal(decoded, hash.Sum(nil))
}

// GetWebhookInfo method
func (client *Client) GetWebhookInfo() *GetWebhookInfo {
	return &GetWebhookInfo{
		c:        client,
		endpoint: APIEndpointGetWebhookInfo,
	}
}

// GetWebhookInfo type
type GetWebhookInfo struct {
	c        *Client
	ctx      context.Context
	endpoint string
}

// WithContext method
func (call *GetWebhookInfo) WithContext(ctx context.Context) *GetWebhookInfo {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetWebhookInfo) Do() (*WebhookInfoResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, call.endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToWebhookInfoResponse(res)
}

// TestWebhook type
type TestWebhook struct {
	c        *Client
	ctx      context.Context
	endpoint string
}

// SetWebhookEndpointURLCall type
type SetWebhookEndpointURLCall struct {
	c   *Client
	ctx context.Context

	endpoint string
}

// SetWebhookEndpointURL method
func (client *Client) SetWebhookEndpointURL(webhookEndpoint string) *SetWebhookEndpointURLCall {
	return &SetWebhookEndpointURLCall{
		c:        client,
		endpoint: webhookEndpoint,
	}
}

// WithContext method
func (call *SetWebhookEndpointURLCall) WithContext(ctx context.Context) *SetWebhookEndpointURLCall {
	call.ctx = ctx
	return call
}

func (call *SetWebhookEndpointURLCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		Endpoint string `json:"endpoint"`
	}{
		Endpoint: call.endpoint,
	})
}

// Do method
func (call *SetWebhookEndpointURLCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.put(call.ctx, APIEndpointSetWebhookEndpoint, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// TestWebhook method
func (client *Client) TestWebhook() *TestWebhook {
	return &TestWebhook{
		c:        client,
		endpoint: APIEndpointTestWebhook,
	}
}

// WithContext method
func (call *TestWebhook) WithContext(ctx context.Context) *TestWebhook {
	call.ctx = ctx
	return call
}

// Do method
func (call *TestWebhook) Do() (*TestWebhookResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, call.endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToTestWebhookResponsee(res)
}
