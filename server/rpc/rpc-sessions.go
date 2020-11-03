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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/golang/protobuf/proto"
)

// GetSessions - Get a list of sessions
func (rpc *Server) GetSessions(ctx context.Context, _ *commonpb.Empty) (*clientpb.Sessions, error) {
	resp := &clientpb.Sessions{
		Sessions: []*clientpb.Session{},
	}
	for _, session := range core.Sessions.All() {
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

// UpdateSession - Update a session
func (rpc *Server) UpdateSession(ctx context.Context, update *clientpb.UpdateSession) (*clientpb.Session, error) {
	resp := &clientpb.Session{}
	session := core.Sessions.Get(update.SessionID)
	if session == nil {
		return resp, ErrInvalidSessionID
	}
	session.Name = update.Name
	core.Sessions.UpdateSession(session)
	resp = session.ToProtobuf()
	return resp, nil
}
