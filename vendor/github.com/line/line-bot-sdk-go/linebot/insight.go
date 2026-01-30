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

// InsightType type
type InsightType string

// InsightType constants
const (
	InsightTypeMessageDelivery      InsightType = "message/delivery"
	InsightTypeUserInteractionStats InsightType = "message/event"
	InsightTypeFollowers            InsightType = "followers"
	InsightTypeDemographic          InsightType = "demographic"
)

// GetNumberMessagesDeliveryCall type
type GetNumberMessagesDeliveryCall struct {
	c   *Client
	ctx context.Context

	date        string
	insightType InsightType
}

// GetNumberMessagesDelivery method
func (client *Client) GetNumberMessagesDelivery(date string) *GetNumberMessagesDeliveryCall {
	return &GetNumberMessagesDeliveryCall{
		c:           client,
		date:        date,
		insightType: InsightTypeMessageDelivery,
	}
}

// WithContext method
func (call *GetNumberMessagesDeliveryCall) WithContext(ctx context.Context) *GetNumberMessagesDeliveryCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetNumberMessagesDeliveryCall) Do() (*MessagesNumberDeliveryResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointInsight, call.insightType)
	q := url.Values{}
	if call.date != "" {
		q.Add("date", call.date)
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesNumberDeliveryResponse(res)
}

// GetNumberFollowersCall type
type GetNumberFollowersCall struct {
	c   *Client
	ctx context.Context

	date        string
	insightType InsightType
}

// GetNumberFollowers method
func (client *Client) GetNumberFollowers(date string) *GetNumberFollowersCall {
	return &GetNumberFollowersCall{
		c:           client,
		date:        date,
		insightType: InsightTypeFollowers,
	}
}

// WithContext method
func (call *GetNumberFollowersCall) WithContext(ctx context.Context) *GetNumberFollowersCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetNumberFollowersCall) Do() (*MessagesNumberFollowersResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointInsight, call.insightType)
	q := url.Values{}
	if call.date != "" {
		q.Add("date", call.date)
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesNumberFollowersResponse(res)
}

// GetFriendDemographicsCall type
type GetFriendDemographicsCall struct {
	c   *Client
	ctx context.Context

	insightType InsightType
}

// GetFriendDemographics method
func (client *Client) GetFriendDemographics() *GetFriendDemographicsCall {
	return &GetFriendDemographicsCall{
		c:           client,
		insightType: InsightTypeDemographic,
	}
}

// WithContext method
func (call *GetFriendDemographicsCall) WithContext(ctx context.Context) *GetFriendDemographicsCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetFriendDemographicsCall) Do() (*MessagesFriendDemographicsResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointInsight, call.insightType)
	var q url.Values

	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesFriendDemographicsResponse(res)
}

// GetUserInteractionStatsCall type
type GetUserInteractionStatsCall struct {
	c   *Client
	ctx context.Context

	requestID   string
	insightType InsightType
}

// GetUserInteractionStats method
func (client *Client) GetUserInteractionStats(requestID string) *GetUserInteractionStatsCall {
	return &GetUserInteractionStatsCall{
		c:           client,
		requestID:   requestID,
		insightType: InsightTypeUserInteractionStats,
	}
}

// WithContext method
func (call *GetUserInteractionStatsCall) WithContext(ctx context.Context) *GetUserInteractionStatsCall {
	call.ctx = ctx
	return call
}

// Do method, returns MessagesUserInteractionStatsResponse
func (call *GetUserInteractionStatsCall) Do() (*MessagesUserInteractionStatsResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointInsight, call.insightType)
	q := url.Values{}
	if call.requestID != "" {
		q.Add("requestId", call.requestID)
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessagesUserInteractionStatsResponse(res)
}
