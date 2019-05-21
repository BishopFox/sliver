package rpc

import (
	"sliver/server/core"
	"sliver/server/log"
	"time"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
)

const (
	defaultTimeout = 30 * time.Second
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

		clientpb.MsgSessions:         rpcSessions,
		clientpb.MsgGenerate:         rpcGenerate,
		clientpb.MsgRegenerate:       rpcRegenerate,
		clientpb.MsgListSliverBuilds: rpcListSliverBuilds,
		clientpb.MsgListCanaries:     rpcListCanaries,
		clientpb.MsgProfiles:         rpcProfiles,
		clientpb.MsgNewProfile:       rpcNewProfile,
		clientpb.MsgPlayers:          rpcPlayers,

		clientpb.MsgMsf:          rpcMsf,
		clientpb.MsgMsfInject:    rpcMsfInject,
		clientpb.MsgGetSystemReq: rpcGetSystem,

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

		sliverpb.MsgShellReq: rpcShell,

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
