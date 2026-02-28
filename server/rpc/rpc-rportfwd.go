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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core/rtunnels"
)

// GetRportFwdListeners - Get a list of all reverse port forwards listeners from an implant
func (rpc *Server) GetRportFwdListeners(ctx context.Context, req *sliverpb.RportFwdListenersReq) (*sliverpb.RportFwdListeners, error) {
	resp := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StartRportfwdListener - Instruct the implant to start a reverse port forward
func (rpc *Server) StartRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStartListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Response.GetErr() == "" {
		addr := req.ForwardAddress
		if resp.ForwardAddress != "" {
			addr = resp.ForwardAddress
		}
		rtunnels.TrackListener(req.Request.SessionID, resp.ID, addr)
	}
	return resp, nil
}

// StopRportfwdListener - Instruct the implant to stop a reverse port forward
func (rpc *Server) StopRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStopListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Response.GetErr() == "" {
		rtunnels.UntrackListener(req.Request.SessionID, req.ID)
	}
	return resp, nil
}
