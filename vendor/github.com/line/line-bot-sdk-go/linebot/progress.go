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
	"fmt"
	"net/url"
)

// ProgressType type
type ProgressType string

// ProgressType constants
const (
	ProgressTypeNarrowcast ProgressType = "narrowcast"
)

// GetProgressNarrowcastMessages method
func (client *Client) GetProgressNarrowcastMessages(requestID string) *GetProgressMessagesCall {
	return &GetProgressMessagesCall{
		c:            client,
		requestID:    requestID,
		progressType: ProgressTypeNarrowcast,
	}
}

// GetProgressMessagesCall type
type GetProgressMessagesCall struct {
	c   *Client
	ctx context.Context

	requestID    string
	progressType ProgressType
}

// WithContext method
func (call *GetProgressMessagesCall) WithContext(ctx context.Context) *GetProgressMessagesCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetProgressMessagesCall) Do() (*MessagesProgressResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetMessageProgress, call.progressType)
	var q url.Values
	if call.requestID != "" {
		q = url.Values{"requestId": []string{call.requestID}}
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesProgressResponse(res)
}
