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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/golang/protobuf/proto"
)

var (
	// ErrTunnelInitFailure - Returned when a tunnel cannot be initialized
	ErrTunnelInitFailure = errors.New("Failed to initialize tunnel")
)

// Shell - Open an interactive shell
func (s *Server) Shell(ctx context.Context, req *sliverpb.ShellReq) (*sliverpb.Shell, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.Tunnels.Get(req.TunnelID)
	if tunnel == nil {
		return nil, core.ErrInvalidTunnelID
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	data, err := session.Request(sliverpb.MsgNumber(req), s.getTimeout(req), reqData)
	if err != nil {
		return nil, err
	}
	shell := &sliverpb.Shell{}
	err = proto.Unmarshal(data, shell)
	return shell, err
}
