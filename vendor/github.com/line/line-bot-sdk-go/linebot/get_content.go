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

// GetMessageContent method
func (client *Client) GetMessageContent(messageID string) *GetMessageContentCall {
	return &GetMessageContentCall{
		c:         client,
		messageID: messageID,
	}
}

// GetMessageContentCall type
type GetMessageContentCall struct {
	c   *Client
	ctx context.Context

	messageID string
}

// WithContext method
func (call *GetMessageContentCall) WithContext(ctx context.Context) *GetMessageContentCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetMessageContentCall) Do() (*MessageContentResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetMessageContent, call.messageID)
	res, err := call.c.get(call.ctx, call.c.endpointBaseData, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessageContentResponse(res)
}
