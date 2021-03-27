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

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetTransports - Get all transports available to an implant.
func (rpc *Server) GetTransports(ctx context.Context, req *commpb.TransportsReq) (*commpb.Transports, error) {
	sess := core.Sessions.Get(req.Request.SessionID)

	data, _ := proto.Marshal(req)
	res := &commpb.Transports{}
	resp, err := sess.Request(sliverpb.MsgNumber(req), rpc.getTimeout(req), data)
	if err != nil {
		return nil, err
	}
	proto.Unmarshal(resp, res)

	return res, nil
}

// AddTransport - Add an available transport to an implant, within the range of transport stacks compiled in its binary.
func (rpc *Server) AddTransport(ctx context.Context, req *commpb.TransportAddReq) (*commpb.TransportAdd, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddTransport not implemented")
}

// DeleteTransport - Remove a transport from an implant.
func (rpc *Server) DeleteTransport(ctx context.Context, req *commpb.TransportDeleteReq) (*commpb.TransportDelete, error) {
	sess := core.Sessions.Get(req.Request.SessionID)

	data, _ := proto.Marshal(req)
	res := &commpb.TransportDelete{}
	resp, err := sess.Request(sliverpb.MsgNumber(req), rpc.getTimeout(req), data)
	if err != nil {
		return nil, err
	}
	proto.Unmarshal(resp, res)

	return res, nil
}

// SwitchTransport - Ask an implant to cut off its current connection to the server (including the underlying Comm system) and
// to start a new connection (new address and/or new protocol and/or new direction). The Comm system is restarted on top of it.
func (rpc *Server) SwitchTransport(ctx context.Context, req *commpb.TransportSwitchReq) (*commpb.TransportSwitch, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SwitchTransport not implemented")
}
