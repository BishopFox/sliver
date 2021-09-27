package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/protobuf/proto"
)

const maxNameLength = 32

var (
	// ErrInvalidName - Invalid name
	ErrInvalidName = errors.New("invalid session name, alphanumerics only")
)

// GetSessions - Get a list of sessions
func (rpc *Server) GetSessions(ctx context.Context, _ *commonpb.Empty) (*clientpb.Sessions, error) {
	resp := &clientpb.Sessions{
		Sessions: []*clientpb.Session{},
	}
	for _, session := range core.Sessions.All() {
		build, err := db.ImplantBuildByName(session.Name)
		if err == nil && build != nil {
			if build.Burned {
				session.Burned = true
			}
		}
		resp.Sessions = append(resp.Sessions, session.ToProtobuf())
	}
	return resp, nil
}

// KillSession - Kill a session
func (rpc *Server) KillSession(ctx context.Context, kill *sliverpb.KillSessionReq) (*commonpb.Empty, error) {
	session := core.Sessions.Get(kill.Request.SessionID)
	if session == nil {
		return &commonpb.Empty{}, ErrInvalidSessionID
	}
	core.Sessions.Remove(session.ID)
	data, err := proto.Marshal(kill)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(kill.Request.GetTimeout())
	session.Request(sliverpb.MsgNumber(kill), timeout, data)
	return &commonpb.Empty{}, nil
}

// UpdateSession - Update a session name
func (rpc *Server) UpdateSession(ctx context.Context, update *clientpb.UpdateSession) (*clientpb.Session, error) {
	resp := &clientpb.Session{}
	session := core.Sessions.Get(update.SessionID)
	if session == nil {
		return resp, ErrInvalidSessionID
	}
	var maxLen int
	if update.Name != "" {
		if len(update.Name) < maxNameLength {
			maxLen = len(update.Name)
		} else {
			maxLen = maxNameLength
		}
		name := update.Name[:maxLen]
		if !regexp.MustCompile(`^[[:alnum:]]+$`).MatchString(name) {
			return resp, ErrInvalidName
		}
		session.Name = name
	}
	//Update reconnect interval if set
	if update.ReconnectInterval != -1 {
		session.ReconnectInterval = update.ReconnectInterval

		//Create protobuf msg
		req := sliverpb.ReconnectIntervalReq{
			Request: &commonpb.Request{
				SessionID: session.ID,
				Timeout:   int64(0),
			},
			ReconnectInterval: update.ReconnectInterval,
		}

		data, err := proto.Marshal(&req)
		if err != nil {
			return nil, err
		}
		session.Request(sliverpb.MsgNumber(&req), rpc.getTimeout(&req), data)
	}
	//Update poll interval if set
	if update.PollInterval != -1 {
		session.PollTimeout = update.PollInterval

		//Create protobuf msg
		req := sliverpb.PollIntervalReq{
			Request: &commonpb.Request{
				SessionID: session.ID,
				Timeout:   int64(0),
			},
			PollInterval: update.PollInterval,
		}

		data, err := proto.Marshal(&req)
		if err != nil {
			return nil, err
		}
		session.Request(sliverpb.MsgNumber(&req), rpc.getTimeout(&req), data)
	}
	if len(update.Extensions) != 0 {
		session.Extensions = update.Extensions
	}
	core.Sessions.UpdateSession(session)
	resp = session.ToProtobuf()
	return resp, nil
}
