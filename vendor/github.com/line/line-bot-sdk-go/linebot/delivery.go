// Copyright 2019 LINE Corporation
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

// DeliveryType type
type DeliveryType string

// DeliveryType constants
const (
	DeliveryTypeMulticast DeliveryType = "multicast"
	DeliveryTypePush      DeliveryType = "push"
	DeliveryTypeReply     DeliveryType = "reply"
	DeliveryTypeBroadcast DeliveryType = "broadcast"
)

// GetNumberReplyMessages method
func (client *Client) GetNumberReplyMessages(date string) *GetNumberMessagesCall {
	return &GetNumberMessagesCall{
		c:            client,
		date:         date,
		deliveryType: DeliveryTypeReply,
	}
}

// GetNumberPushMessages method
func (client *Client) GetNumberPushMessages(date string) *GetNumberMessagesCall {
	return &GetNumberMessagesCall{
		c:            client,
		date:         date,
		deliveryType: DeliveryTypePush,
	}
}

// GetNumberMulticastMessages method
func (client *Client) GetNumberMulticastMessages(date string) *GetNumberMessagesCall {
	return &GetNumberMessagesCall{
		c:            client,
		date:         date,
		deliveryType: DeliveryTypeMulticast,
	}
}

// GetNumberBroadcastMessages method
func (client *Client) GetNumberBroadcastMessages(date string) *GetNumberMessagesCall {
	return &GetNumberMessagesCall{
		c:            client,
		date:         date,
		deliveryType: DeliveryTypeBroadcast,
	}
}

// GetNumberMessagesCall type
type GetNumberMessagesCall struct {
	c   *Client
	ctx context.Context

	date         string
	deliveryType DeliveryType
}

// WithContext method
func (call *GetNumberMessagesCall) WithContext(ctx context.Context) *GetNumberMessagesCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetNumberMessagesCall) Do() (*MessagesNumberResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetMessageDelivery, call.deliveryType)
	var q url.Values
	if call.date != "" {
		q = url.Values{"date": []string{call.date}}
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesNumberResponse(res)
}
