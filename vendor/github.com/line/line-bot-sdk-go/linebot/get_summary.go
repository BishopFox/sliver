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
)

// GetGroupSummary method
func (client *Client) GetGroupSummary(groupID string) *GetGroupSummaryCall {
	return &GetGroupSummaryCall{
		c:       client,
		groupID: groupID,
	}
}

// GetGroupSummaryCall type
type GetGroupSummaryCall struct {
	c   *Client
	ctx context.Context

	groupID string
}

// WithContext method
func (call *GetGroupSummaryCall) WithContext(ctx context.Context) *GetGroupSummaryCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetGroupSummaryCall) Do() (*GroupSummaryResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetGroupSummary, call.groupID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToGroupSummaryResponse(res)
}
