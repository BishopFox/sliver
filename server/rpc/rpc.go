package rpc

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
	"time"

	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
)

var (
	rpcLog = log.NamedLogger("rpc", "server")
)

// RPCResponse - Called with response data, mapped back to reqID
type RPCResponse func([]byte, error)

// RPCHandler - RPC handlers accept bytes and return bytes
type RPCHandler func([]byte, time.Duration, RPCResponse)
type TunnelHandler func(*core.Client, []byte, RPCResponse)

var (
	rpcHandlers = &map[uint32]RPCHandler{
		clientpb.MsgJobs:    rpcJobs,
		clientpb.MsgJobKill: rpcJobKill,
		clientpb.MsgMtls:    rpcStartMTLSListener,
		clientpb.MsgDns:     rpcStartDNSListener,
		clientpb.MsgHttp:    rpcStartHTTPListener,
		clientpb.MsgHttps:   rpcStartHTTPSListener,

		clientpb.MsgWebsiteList:          rpcWebsiteList,
		clientpb.MsgWebsiteAddContent:    rpcWebsiteAddContent,
		clientpb.MsgWebsiteRemoveContent: rpcWebsiteRemoveContent,

		clientpb.MsgSessions:         rpcSessions,
		clientpb.MsgGenerate:         rpcGenerate,
		clientpb.MsgRegenerate:       rpcRegenerate,
		clientpb.MsgListSliverBuilds: rpcListSliverBuilds,
		clientpb.MsgListCanaries:     rpcListCanaries,
		clientpb.MsgProfiles:         rpcProfiles,
		clientpb.MsgNewProfile:       rpcNewProfile,
		clientpb.MsgPlayers:          rpcPlayers,

		clientpb.MsgMsf:       rpcMsf,
		clientpb.MsgMsfInject: rpcMsfInject,

		clientpb.MsgGetSystemReq: rpcGetSystem,

		clientpb.MsgEggReq: rpcEgg,

		// "Req"s directly map to responses
		sliverpb.MsgPsReq:          rpcPs,
		sliverpb.MsgKill:           rpcKill,
		sliverpb.MsgProcessDumpReq: rpcProcdump,

		sliverpb.MsgElevate:         rpcElevate,
		sliverpb.MsgImpersonate:     rpcImpersonate,
		sliverpb.MsgExecuteAssembly: rpcExecuteAssembly,

		sliverpb.MsgLsReq:       rpcLs,
		sliverpb.MsgRmReq:       rpcRm,
		sliverpb.MsgMkdirReq:    rpcMkdir,
		sliverpb.MsgCdReq:       rpcCd,
		sliverpb.MsgPwdReq:      rpcPwd,
		sliverpb.MsgDownloadReq: rpcDownload,
		sliverpb.MsgUploadReq:   rpcUpload,

		sliverpb.MsgIfconfigReq: rpcIfconfig,

		sliverpb.MsgShellReq:   rpcShell,
		sliverpb.MsgExecuteReq: rpcExecute,

		clientpb.MsgTask:    rpcLocalTask,
		clientpb.MsgMigrate: rpcMigrate,
	}

	tunHandlers = &map[uint32]TunnelHandler{
		clientpb.MsgTunnelCreate: tunnelCreate,
		sliverpb.MsgTunnelData:   tunnelData,
		sliverpb.MsgTunnelClose:  tunnelClose,
	}
)

// GetRPCHandlers - Returns a map of server-side msg handlers
func GetRPCHandlers() *map[uint32]RPCHandler {
	return rpcHandlers
}

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() *map[uint32]TunnelHandler {
	return tunHandlers
}
