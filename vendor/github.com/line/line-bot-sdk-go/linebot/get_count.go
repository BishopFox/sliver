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

// GetGroupMemberCount method
func (client *Client) GetGroupMemberCount(groupID string) *GetGroupMemberCountCall {
	return &GetGroupMemberCountCall{
		c:       client,
		groupID: groupID,
	}
}

// GetGroupMemberCountCall type
type GetGroupMemberCountCall struct {
	c   *Client
	ctx context.Context

	groupID string
}

// WithContext method
func (call *GetGroupMemberCountCall) WithContext(ctx context.Context) *GetGroupMemberCountCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetGroupMemberCountCall) Do() (*MemberCountResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetGroupMemberCount, call.groupID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMemberCountResponse(res)
}

// GetRoomMemberCount method
func (client *Client) GetRoomMemberCount(roomID string) *GetRoomMemberCountCall {
	return &GetRoomMemberCountCall{
		c:      client,
		roomID: roomID,
	}
}

// GetRoomMemberCountCall type
type GetRoomMemberCountCall struct {
	c   *Client
	ctx context.Context

	roomID string
}

// WithContext method
func (call *GetRoomMemberCountCall) WithContext(ctx context.Context) *GetRoomMemberCountCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetRoomMemberCountCall) Do() (*MemberCountResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetRoomMemberCount, call.roomID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMemberCountResponse(res)
}
