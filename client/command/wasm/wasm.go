package wasm

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox
	Copyright (C) 2023 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// wasmMaxModuleSize - Arbitrary 1.5Gb limit to put us well under the 2Gb max gRPC message size
// wasmMaxModuleSize - Arbitrary 1.5Gb 限制，使我们远低于 2Gb 最大 gRPC 消息大小
// this is also the *compressed size* limit, so it's pretty generous.
// 这也是*压缩大小*限制，所以它相当 generous.
const (
	gb                = 1024 * 1024 * 1024
	wasmMaxModuleSize = gb + (gb / 2)
)

// WasmCmd - session/beacon id -> list of loaded wasm extension names.
// WasmCmd - session/beacon id -> 加载的 wasm 扩展列表 names.
var wasmRegistrationCache = make(map[string][]string)

// WasmCmd - Execute a WASM module extension.
// WasmCmd - Execute 一个 WASM 模块 extension.
func WasmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Wasm module file path
	// Wasm 模块文件路径
	wasmFilePath := args[0]
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		con.PrintErrorf("File does not exist: %s", wasmFilePath)
		return
	}

	// Parse memfs args and build memfs map
	// Parse memfs 参数并构建 memfs 映射
	memfs, err := parseMemFS(cmd, con, args)
	if err != nil {
		con.PrintErrorf("memfs error: %s", err)
		return
	}

	// Wasm module args
	// Wasm 模块参数
	wasmArgs := args[1:]
	interactive, _ := cmd.Flags().GetBool("pipe")

	skipRegistration, _ := cmd.Flags().GetBool("skip-registration")

	if !skipRegistration && !isRegistered(filepath.Base(wasmFilePath), cmd, con) {
		con.PrintInfof("Registering wasm extension '%s' ...\n", wasmFilePath)
		err := registerWasmExtension(wasmFilePath, cmd, con)
		if err != nil {
			con.PrintErrorf("Failed to register wasm extension '%s': %s\n", wasmFilePath, err)
			return
		}
	}

	execWasmReq := &sliverpb.ExecWasmExtensionReq{
		Name:        filepath.Base(wasmFilePath),
		Args:        wasmArgs,
		MemFS:       memfs,
		Interactive: interactive,
		Request:     con.ActiveTarget.Request(cmd),
	}
	if interactive {
		runInteractive(cmd, execWasmReq, con)
	} else {
		runNonInteractive(execWasmReq, con)
	}
}

func isRegistered(name string, cmd *cobra.Command, con *console.SliverClient) bool {
	// Check if we have already registered this wasm module
	// Check 如果我们已经注册了这个 wasm 模块
	if wasmRegistrationCache[idOf(con)] != nil {
		if util.Contains(wasmRegistrationCache[idOf(con)], name) {
			return true
		}
	}

	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	loaded, err := con.Rpc.ListWasmExtensions(grpcCtx, &sliverpb.ListWasmExtensionsReq{
		Request: con.ActiveTarget.Request(cmd),
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

// idOf - Quickly return the id of the current session or beacon.
// idOf - Quickly 返回当前 session 或 beacon. 的 id
func idOf(con *console.SliverClient) string {
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

func runNonInteractive(execWasmReq *sliverpb.ExecWasmExtensionReq, con *console.SliverClient) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	execWasmResp, err := con.Rpc.ExecWasmExtension(grpcCtx, execWasmReq)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if execWasmResp.Response != nil && execWasmResp.Response.Async {
		con.AddBeaconCallback(execWasmResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execWasmResp)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			con.PrintInfof("Executed wasm extension '%s' successfully\n", execWasmReq.Name)
			os.Stdout.Write(execWasmResp.Stdout)
			os.Stderr.Write(execWasmResp.Stderr)
		})
	} else {
		con.PrintInfof("Executed wasm extension '%s' successfully\n", execWasmReq.Name)
		os.Stdout.Write(execWasmResp.Stdout)
		os.Stderr.Write(execWasmResp.Stderr)
	}
}

func runInteractive(cmd *cobra.Command, execWasmReq *sliverpb.ExecWasmExtensionReq, con *console.SliverClient) {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		con.PrintErrorf("No active session\n")
		if beacon := con.ActiveTarget.GetBeacon(); beacon != nil {
			con.PrintWarnf("Wasm modules cannot be executed with --pipe in beacon mode\n")
		}
		return
	}

	// Create an RPC tunnel
	// Create 和 RPC 隧道
	ctxTunnel, cancelTunnel := context.WithCancel(context.Background())
	rpcTunnel, err := con.Rpc.CreateTunnel(ctxTunnel, &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	defer cancelTunnel()
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Wait approximately 10 seconds after exit, and press <enter> to continue\n")
	con.PrintInfof("Streaming output from '%s' wasm extension ...\n", execWasmReq.Name)

	// Create tunnel
	// Create 隧道
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)
	execWasmReq.TunnelID = rpcTunnel.TunnelID
	defer tunnel.Close()

	// Send the exec request
	// Send 执行请求
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

	// Setup routines to copy data back an forth
	// Setup 来回复制数据的例程
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
	// Copy stdin 到隧道
	_, err = io.Copy(tunnel, os.Stdin)
	if err == io.EOF {
		return
	}
	if err != nil {
		con.PrintErrorf("Error reading from stdin: %s\n", err)
	}
}

func registerWasmExtension(wasmFilePath string, cmd *cobra.Command, con *console.SliverClient) error {
	grpcCtx, cancel := con.GrpcContext(cmd)
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
		Request: con.ActiveTarget.Request(cmd),
		Name:    filepath.Base(wasmFilePath),
		WasmGz:  data,
	})
	if err != nil {
		return err
	}
	wasmRegistrationCache[idOf(con)] = append(wasmRegistrationCache[idOf(con)], filepath.Base(wasmFilePath))
	return nil
}

// WasmLsCmd - Execute a WASM module extension.
// WasmLsCmd - Execute 一个 WASM 模块 extension.
func WasmLsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	loaded, err := con.Rpc.ListWasmExtensions(grpcCtx, &sliverpb.ListWasmExtensionsReq{
		Request: con.ActiveTarget.Request(cmd),
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
				cacheLine = console.StyleBoldGreen.Render(" +++")
			}
			con.PrintInfof("  %s%s\n", extName, cacheLine)
		}
	}
}
