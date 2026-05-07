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

// GetMessageQuota method
func (client *Client) GetMessageQuota() *GetMessageQuotaCall {
	return &GetMessageQuotaCall{
		c:        client,
		endpoint: APIEndpointGetMessageQuota,
	}
}

// GetMessageQuotaConsumption method
func (client *Client) GetMessageQuotaConsumption() *GetMessageQuotaCall {
	return &GetMessageQuotaCall{
		c:        client,
		endpoint: APIEndpointGetMessageQuotaConsumption,
	}
}

// GetMessageQuotaCall type
type GetMessageQuotaCall struct {
	c        *Client
	ctx      context.Context
	endpoint string
}

// WithContext method
func (call *GetMessageQuotaCall) WithContext(ctx context.Context) *GetMessageQuotaCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetMessageQuotaCall) Do() (*MessageQuotaResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, call.endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessageQuotaResponse(res)
}

// GetMessageConsumption method
func (client *Client) GetMessageConsumption() *GetMessageConsumptionCall {
	return &GetMessageConsumptionCall{
		c: client,
	}
}

// GetMessageConsumptionCall type
type GetMessageConsumptionCall struct {
	c   *Client
	ctx context.Context
}

// WithContext method
func (call *GetMessageConsumptionCall) WithContext(ctx context.Context) *GetMessageConsumptionCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetMessageConsumptionCall) Do() (*MessageConsumptionResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, APIEndpointGetMessageConsumption, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMessageConsumptionResponse(res)
}
