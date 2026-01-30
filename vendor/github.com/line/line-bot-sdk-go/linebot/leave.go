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

// LeaveGroup method
func (client *Client) LeaveGroup(groupID string) *LeaveGroupCall {
	return &LeaveGroupCall{
		c:       client,
		groupID: groupID,
	}
}

// LeaveGroupCall type
type LeaveGroupCall struct {
	c   *Client
	ctx context.Context

	groupID string
}

// WithContext method
func (call *LeaveGroupCall) WithContext(ctx context.Context) *LeaveGroupCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *LeaveGroupCall) Do() (*BasicResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointLeaveGroup, call.groupID)
	res, err := call.c.post(call.ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// LeaveRoom method
func (client *Client) LeaveRoom(roomID string) *LeaveRoomCall {
	return &LeaveRoomCall{
		c:      client,
		roomID: roomID,
	}
}

// LeaveRoomCall type
type LeaveRoomCall struct {
	c   *Client
	ctx context.Context

	roomID string
}

// WithContext method
func (call *LeaveRoomCall) WithContext(ctx context.Context) *LeaveRoomCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *LeaveRoomCall) Do() (*BasicResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointLeaveRoom, call.roomID)
	res, err := call.c.post(call.ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}
