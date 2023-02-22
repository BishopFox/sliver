package wasm

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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
)

// wasmMaxModuleSize - Arbitrary 1.5Gb limit to put us well under the 2Gb max gRPC message size
// this is also the *compressed size* limit, so it's pretty generous
const gb = 1024 * 1024 * 1024
const wasmMaxModuleSize = gb + (gb / 2)

// WasmCmd - session/beacon id -> list of loaded wasm extension names
var wasmRegistrationCache = make(map[string][]string)

// WasmCmd - Execute a WASM module extension
func WasmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Wasm module file path
	wasmFilePath := ctx.Args.String("filepath")
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		con.PrintErrorf("File does not exist: %s", wasmFilePath)
		return
	}

	// Wasm module args
	wasmArgs := ctx.Args.StringList("arguments")
	interactive := !ctx.Flags.Bool("non-interactive")

	if !ctx.Flags.Bool("skip-registration") && !isRegistered(filepath.Base(wasmFilePath), ctx, con) {
		con.PrintInfof("Registering wasm extension '%s' ...\n", wasmFilePath)
		err := registerWasmExtension(wasmFilePath, ctx, con)
		if err != nil {
			con.PrintErrorf("Failed to register wasm extension '%s': %s\n", wasmFilePath, err)
			return
		}
	}

	execWasmReq := &sliverpb.ExecWasmExtensionReq{
		Name:        filepath.Base(wasmFilePath),
		Args:        wasmArgs,
		Interactive: interactive,
		Request:     con.ActiveTarget.Request(ctx),
	}
	if interactive {
		runInteractive(ctx, execWasmReq, con)
	} else {
		runNonInteractive(execWasmReq, con)
	}
}

func isRegistered(name string, ctx *grumble.Context, con *console.SliverConsoleClient) bool {

	// Check if we have already registered this wasm module
	if wasmRegistrationCache[idOf(con)] != nil {
		if util.Contains(wasmRegistrationCache[idOf(con)], name) {
			return true
		}
	}

	grpcCtx, cancel := con.GrpcContext(ctx)
	defer cancel()
	loaded, err := con.Rpc.ListWasmExtensions(grpcCtx, &sliverpb.ListWasmExtensionsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return false
	}
	for _, extName := range loaded.Names {
		wasmRegistrationCache[idOf(con)] = append(wasmRegistrationCache[idOf(con)], extName)
		if extName == name {
			return true
		}
	}
	return false
}

// idOf - Quickly return the id of the current session or beacon
func idOf(con *console.SliverConsoleClient) string {
	if con.ActiveTarget != nil {
		if session := con.ActiveTarget.GetSession(); session != nil {
			return session.ID
		}
		if beacon := con.ActiveTarget.GetBeacon(); beacon != nil {
			return beacon.ID
		}
	}
	return ""
}

func runNonInteractive(execWasmReq *sliverpb.ExecWasmExtensionReq, con *console.SliverConsoleClient) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	execWasmResp, err := con.Rpc.ExecWasmExtension(grpcCtx, execWasmReq)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if execWasmResp.Response != nil && execWasmResp.Response.Err != "" {
		con.PrintErrorf("%s\n", execWasmResp.Response.Err)
		return
	}
	con.PrintInfof("Executed wasm extension '%s' successfully\n", execWasmReq.Name)

	con.App.Stdout().Write(execWasmResp.Stdout)
	con.App.Stderr().Write(execWasmResp.Stderr)
}

func runInteractive(ctx *grumble.Context, execWasmReq *sliverpb.ExecWasmExtensionReq, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		con.PrintErrorf("No active session\n")
		return
	}

	// Create an RPC tunnel
	ctxTunnel, cancelTunnel := context.WithCancel(context.Background())
	rpcTunnel, err := con.Rpc.CreateTunnel(ctxTunnel, &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	defer cancelTunnel()
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Created new tunnel: %d, binding to '%s' wasm extension ...\n", rpcTunnel.TunnelID, execWasmReq.Name)

	// Create tunnel
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)
	execWasmReq.TunnelID = rpcTunnel.TunnelID
	defer tunnel.Close()

	// Send the exec request
	wasmExt, err := con.Rpc.ExecWasmExtension(context.Background(), execWasmReq)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if wasmExt.Response != nil && wasmExt.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", wasmExt.Response.Err)
		_, err = con.Rpc.CloseTunnel(context.Background(), &sliverpb.Tunnel{
			TunnelID:  tunnel.ID,
			SessionID: session.ID,
		})
		if err != nil {
			con.PrintErrorf("RPC Error: %s\n", err)
		}
		return
	}

	// Save & restore terminal state
	// var oldState *term.State
	// oldState, err = term.MakeRaw(0)
	// if err != nil {
	// 	con.PrintErrorf("Failed to save terminal state")
	// 	return
	// }
	// defer func() {
	// 	term.Restore(0, oldState)
	// 	bufio.NewWriter(os.Stdout).Flush()
	// }()

	// Setup routines to copy data back an forth
	go func() {
		_, err := io.Copy(os.Stdout, tunnel)
		if err == io.EOF {
			return
		}
		if err != nil {
			con.PrintErrorf("Error writing to stdout: %s", err)
			return
		}
	}()

	// Copy stdin to the tunnel
	_, err = io.Copy(tunnel, os.Stdin)
	if err == io.EOF {
		return
	}
	if err != nil {
		con.PrintErrorf("Error reading from stdin: %s\n", err)
	}
}

func registerWasmExtension(wasmFilePath string, ctx *grumble.Context, con *console.SliverConsoleClient) error {
	grpcCtx, cancel := con.GrpcContext(ctx)
	defer cancel()
	data, err := os.ReadFile(wasmFilePath)
	if err != nil {
		return err
	}
	data = encoders.GzipBufBestCompression(data)
	if len(data) > wasmMaxModuleSize {
		return fmt.Errorf("wasm module is too big %s (max %s)",
			util.ByteCountBinary(int64(len(data))),
			util.ByteCountBinary(int64(wasmMaxModuleSize)),
		)
	}
	_, err = con.Rpc.RegisterWasmExtension(grpcCtx, &sliverpb.RegisterWasmExtensionReq{
		Request: con.ActiveTarget.Request(ctx),
		Name:    filepath.Base(wasmFilePath),
		WasmGz:  data,
	})
	if err != nil {
		return err
	}
	wasmRegistrationCache[idOf(con)] = append(wasmRegistrationCache[idOf(con)], filepath.Base(wasmFilePath))
	return nil
}

// WasmLsCmd - Execute a WASM module extension
func WasmLsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	grpcCtx, cancel := con.GrpcContext(ctx)
	defer cancel()
	loaded, err := con.Rpc.ListWasmExtensions(grpcCtx, &sliverpb.ListWasmExtensionsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	if len(loaded.Names) < 1 {
		con.PrintInfof("No wasm extensions registered\n")
	} else {
		for _, extName := range loaded.Names {
			cacheLine := ""
			if util.Contains(wasmRegistrationCache[idOf(con)], extName) {
				cacheLine = " (cached)"
			} else {
				wasmRegistrationCache[idOf(con)] = append(wasmRegistrationCache[idOf(con)], extName)
				cacheLine = console.Bold + console.Green + " +++" + console.Normal
			}
			con.PrintInfof("  %s%s\n", extName, cacheLine)
		}
	}
}
