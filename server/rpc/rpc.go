package rpc

import (
	consts "sliver/client/constants"
	"sliver/server/core"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
)

// RPCResponse - Called with response data, mapped back to reqID
type RPCResponse func([]byte, error)

// RPCHandler - RPC handlers accept bytes and return bytes
type RPCHandler func([]byte, RPCResponse)
type TUNHandler func(*core.Client, []byte, RPCResponse)

var (
	rpcHandlers = &map[string]RPCHandler{
		consts.JobsStr: rpcJobs,
		consts.MtlsStr: rpcStartMTLSListener,
		consts.DnsStr:  rpcStartDNSListener,

		consts.SessionsStr:   rpcSessions,
		consts.GenerateStr:   rpcGenerate,
		consts.ProfilesStr:   rpcProfiles,
		consts.NewProfileStr: rpcNewProfile,

		consts.MsfStr:    rpcMsf,
		consts.InjectStr: rpcMsfInject,

		consts.PsStr:       rpcPs,
		consts.ProcdumpStr: rpcProcdump,

		consts.LsStr:       rpcLs,
		consts.RmStr:       rpcRm,
		consts.MkdirStr:    rpcMkdir,
		consts.CdStr:       rpcCd,
		consts.PwdStr:      rpcPwd,
		consts.DownloadStr: rpcDownload,
		consts.UploadStr:   rpcUpload,

		consts.ShellStr: rpcShell,
	}

	tunHandlers = &map[string]TUNHandler{
		consts.CreateTUNStr: tunCreate,
	}
)

// GetRPCHandlers - Returns a map of server-side msg handlers
func GetRPCHandlers() *map[string]RPCHandler {
	return rpcHandlers
}

// GetTUNHandlers - Returns a map of tunnel handlers
func GetTUNHandlers() *map[string]TUNHandler {
	return tunHandlers
}
