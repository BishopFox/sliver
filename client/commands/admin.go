package commands

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
	"fmt"

	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// NewOperator - Command for creating a new operator user.
type NewOperator struct {
	Options OperatorOptions `group:"Operator options"`
}

// OperatorOptions - Options for operator creation.
type OperatorOptions struct {
	Operator string `long:"operator" description:"Name of operator" required:"true"`
	LHost    string `long:"lhost" description:"Server listening host (default: localhost)" default:"localhost"`
	LPort    int    `long:"lport" description:"Server listening port (default: 31337)" default:"31337"`
	Save     string `long:"save" description:"Directory/file to save configuration file"`
}

// Execute - Create new operator user.
func (n NewOperator) Execute(args []string) (err error) {

	req := &clientpb.NewPlayerReq{
		Name:  n.Options.Operator,
		LHost: n.Options.LHost,
		LPort: uint32(n.Options.LPort),
		Save:  n.Options.Save,
	}

	resp, err := transport.AdminRPC.CreatePlayer(context.Background(), req, &grpc.EmptyCallOption{})
	if err != nil {
		fmt.Printf(util.Warn+"RPC error: %s \n", err.Error())
		return
	}

	if resp.Success {
		fmt.Printf(util.Info+"Created player config for operator %s at %s:%d \n", req.Name, req.LHost, req.LPort)
	} else {
		fmt.Printf(util.Warn+"Failed to create player config: %s \n", resp.Response.Err)
	}

	return
}

// KickOperator - Kick an operator out of server and remove certificates.
type KickOperator struct {
	Positional struct {
		Operator string `description:"Name of operator to kick off"`
	} `positional-args:"yes"`
}

// Execute - Kick operator from server.
func (k *KickOperator) Execute(args []string) (err error) {

	req := &clientpb.RemovePlayerReq{
		Name: k.Positional.Operator,
	}

	resp, err := transport.AdminRPC.KickPlayer(context.Background(), req, &grpc.EmptyCallOption{})
	if err != nil {
		fmt.Printf(util.Warn+"RPC error: %s \n", err.Error())
		return
	}

	if resp.Success {
		fmt.Printf(util.Info+"Kicked player %s from the server \n", k.Positional.Operator)
	} else {
		fmt.Printf(util.Warn+"Failed to kick player: %s \n", resp.Response.Err)
	}
	return
}

// MultiplayerMode - Enable team playing on server
type MultiplayerMode struct {
	Options MultiplayerOptions `group:"Multiplayer options"`
}

// MultiplayerOptions - Available to server multiplayer mode.
type MultiplayerOptions struct {
	LHost string `long:"lhost" description:"Server listening host (default: localhost)" default:"localhost"`
	LPort int    `long:"lport" description:"Server listening port (default: 31337)" default:"31337"`
}

// Execute - Start multiplayer mode.
func (m *MultiplayerMode) Execute(args []string) (err error) {

	req := &clientpb.MultiplayerReq{
		LHost: m.Options.LHost,
		LPort: uint32(m.Options.LPort),
	}

	resp, err := transport.AdminRPC.StartMultiplayer(context.Background(), req, &grpc.EmptyCallOption{})
	if err != nil {
		fmt.Printf(util.Warn+"RPC error: %s \n", err.Error())
		return
	}

	if resp.Success {
		fmt.Printf(util.Info+"Started Client gRPC listener at %s:%d \n", req.LHost, req.LPort)
	} else {
		fmt.Printf(util.Warn+"Failed to start gRPC client listener: %s \n", resp.Response.Err)
	}

	return
}
