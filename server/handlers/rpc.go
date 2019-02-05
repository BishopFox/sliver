package handlers

import (
	consts "sliver/client/constants"
)

// RPCResponse - Called with response data, mapped back to reqID
type RPCResponse func([]byte, error)

// RPCHandler - RPC handlers accept bytes and return bytes
type RPCHandler func([]byte, RPCResponse)

var (
	rpcHandlers = &map[string]RPCHandler{
		consts.SessionsStr: rpcSessions,
		consts.GenerateStr: rpcGenerate,
		consts.PsStr:       rpcPs,
	}
)

// GetRPCHandlers - Returns a map of server-side msg handlers
func GetRPCHandlers() *map[string]RPCHandler {
	return rpcHandlers
}
