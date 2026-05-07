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
)

// ChatMode type
type ChatMode string

// ChatMode constants
const (
	ChatModeChat ChatMode = "chat"
	ChatModeBot           = "bot"
)

// MarkAsReadMode type
type MarkAsReadMode string

// MarkAsReadMode constants
const (
	MarkAsReadModeManual MarkAsReadMode = "manual"
	MarkAsReadModeAuto                  = "auto"
)

// GetBotInfo method
func (client *Client) GetBotInfo() *GetBotInfoCall {
	return &GetBotInfoCall{
		c:        client,
		endpoint: APIEndpointGetBotInfo,
	}
}

// WithContext method
func (call *GetBotInfoCall) WithContext(ctx context.Context) *GetBotInfoCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetBotInfoCall) Do() (*BotInfoResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, call.endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBotInfoResponse(res)
}

// GetBotInfoCall type
type GetBotInfoCall struct {
	c        *Client
	ctx      context.Context
	endpoint string
}
