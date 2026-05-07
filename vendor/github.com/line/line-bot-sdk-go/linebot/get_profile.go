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

// GetProfile method
func (client *Client) GetProfile(userID string) *GetProfileCall {
	return &GetProfileCall{
		c:      client,
		userID: userID,
	}
}

// GetProfileCall type
type GetProfileCall struct {
	c   *Client
	ctx context.Context

	userID string
}

// WithContext method
func (call *GetProfileCall) WithContext(ctx context.Context) *GetProfileCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetProfileCall) Do() (*UserProfileResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetProfile, call.userID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToUserProfileResponse(res)
}

// GetGroupMemberProfile method
func (client *Client) GetGroupMemberProfile(groupID, userID string) *GetGroupMemberProfileCall {
	return &GetGroupMemberProfileCall{
		c:       client,
		groupID: groupID,
		userID:  userID,
	}
}

// GetGroupMemberProfileCall type
type GetGroupMemberProfileCall struct {
	c   *Client
	ctx context.Context

	groupID string
	userID  string
}

// WithContext method
func (call *GetGroupMemberProfileCall) WithContext(ctx context.Context) *GetGroupMemberProfileCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetGroupMemberProfileCall) Do() (*UserProfileResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetGroupMemberProfile, call.groupID, call.userID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToUserProfileResponse(res)
}

// GetRoomMemberProfile method
func (client *Client) GetRoomMemberProfile(roomID, userID string) *GetRoomMemberProfileCall {
	return &GetRoomMemberProfileCall{
		c:      client,
		roomID: roomID,
		userID: userID,
	}
}

// GetRoomMemberProfileCall type
type GetRoomMemberProfileCall struct {
	c   *Client
	ctx context.Context

	roomID string
	userID string
}

// WithContext method
func (call *GetRoomMemberProfileCall) WithContext(ctx context.Context) *GetRoomMemberProfileCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetRoomMemberProfileCall) Do() (*UserProfileResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetRoomMemberProfile, call.roomID, call.userID)
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToUserProfileResponse(res)
}
