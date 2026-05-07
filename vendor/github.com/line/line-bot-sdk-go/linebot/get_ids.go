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
	"net/url"
)

// GetGroupMemberIDs method
func (client *Client) GetGroupMemberIDs(groupID, continuationToken string) *GetGroupMemberIDsCall {
	return &GetGroupMemberIDsCall{
		c:                 client,
		groupID:           groupID,
		continuationToken: continuationToken,
	}
}

// GetGroupMemberIDsCall type
type GetGroupMemberIDsCall struct {
	c   *Client
	ctx context.Context

	groupID           string
	continuationToken string
}

// WithContext method
func (call *GetGroupMemberIDsCall) WithContext(ctx context.Context) *GetGroupMemberIDsCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetGroupMemberIDsCall) Do() (*MemberIDsResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetGroupMemberIDs, call.groupID)
	var q url.Values
	if call.continuationToken != "" {
		q = url.Values{"start": []string{call.continuationToken}}
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMemberIDsResponse(res)
}

// GetRoomMemberIDs method
func (client *Client) GetRoomMemberIDs(roomID, continuationToken string) *GetRoomMemberIDsCall {
	return &GetRoomMemberIDsCall{
		c:                 client,
		roomID:            roomID,
		continuationToken: continuationToken,
	}
}

// GetRoomMemberIDsCall type
type GetRoomMemberIDsCall struct {
	c   *Client
	ctx context.Context

	roomID            string
	continuationToken string
}

// WithContext method
func (call *GetRoomMemberIDsCall) WithContext(ctx context.Context) *GetRoomMemberIDsCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetRoomMemberIDsCall) Do() (*MemberIDsResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointGetRoomMemberIDs, call.roomID)
	var q url.Values
	if call.continuationToken != "" {
		q = url.Values{"start": []string{call.continuationToken}}
	}
	res, err := call.c.get(call.ctx, call.c.endpointBase, endpoint, q)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToMemberIDsResponse(res)
}

// NewScanner returns Group IDs scanner.
func (call *GetGroupMemberIDsCall) NewScanner() *IDsScanner {
	var ctx context.Context
	if call.ctx != nil {
		ctx = call.ctx
	} else {
		ctx = context.Background()
	}

	c2 := &GetGroupMemberIDsCall{}
	*c2 = *call
	c2.ctx = ctx

	return &IDsScanner{
		caller: c2,
		ctx:    ctx,
	}
}

func (call *GetGroupMemberIDsCall) setContinuationToken(token string) {
	call.continuationToken = token
}

// NewScanner returns Room IDs scanner.
func (call *GetRoomMemberIDsCall) NewScanner() *IDsScanner {
	var ctx context.Context
	if call.ctx != nil {
		ctx = call.ctx
	} else {
		ctx = context.Background()
	}

	c2 := &GetRoomMemberIDsCall{}
	*c2 = *call
	c2.ctx = ctx

	return &IDsScanner{
		caller: c2,
		ctx:    ctx,
	}
}
func (call *GetRoomMemberIDsCall) setContinuationToken(token string) {
	call.continuationToken = token
}

type memberIDsCaller interface {
	Do() (*MemberIDsResponse, error)
	setContinuationToken(string)
}

// IDsScanner type
type IDsScanner struct {
	caller memberIDsCaller
	ctx    context.Context
	start  int
	ids    []string
	next   string
	called bool
	done   bool
	err    error
}

// Scan method
func (s *IDsScanner) Scan() bool {
	if s.done {
		return false
	}

	select {
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		s.done = true
		return false
	default:
	}

	s.start++
	if len(s.ids) > 0 && len(s.ids) > s.start {
		return true
	}

	if s.next == "" && s.called {
		s.done = true
		return false
	}

	s.start = 0
	res, err := s.caller.Do()
	if err != nil {
		s.err = err
		s.done = true
		return false
	}

	s.called = true
	s.ids = res.MemberIDs
	s.next = res.Next
	s.caller.setContinuationToken(s.next)

	return true
}

// ID returns member id.
func (s *IDsScanner) ID() string {
	if len(s.ids) == 0 {
		return ""
	}
	return s.ids[s.start : s.start+1][0]
}

// Err returns scan error.
func (s *IDsScanner) Err() error {
	return s.err
}
