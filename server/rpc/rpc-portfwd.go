package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

// Portfwd - Open an in-band port forward
func (s *Server) Portfwd(ctx context.Context, req *sliverpb.PortfwdReq) (*sliverpb.Portfwd, error) {
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
	portfwd := &sliverpb.Portfwd{}
	err = proto.Unmarshal(data, portfwd)
	return portfwd, err
}
