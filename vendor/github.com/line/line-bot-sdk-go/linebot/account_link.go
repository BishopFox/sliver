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
	"context"
	"fmt"
)

// IssueLinkToken method
// https://developers.line.me/en/reference/messaging-api/#issue-link-token
func (client *Client) IssueLinkToken(userID string) *IssueLinkTokenCall {
	return &IssueLinkTokenCall{
		c:      client,
		userID: userID,
	}
}

// IssueLinkTokenCall type
type IssueLinkTokenCall struct {
	c   *Client
	ctx context.Context

	userID string
}

// WithContext method
func (call *IssueLinkTokenCall) WithContext(ctx context.Context) *IssueLinkTokenCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *IssueLinkTokenCall) Do() (*LinkTokenResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointLinkToken, call.userID)
	res, err := call.c.post(call.ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToLinkTokenResponse(res)
}
