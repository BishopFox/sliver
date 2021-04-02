package assets

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

	"github.com/maxlandon/readline"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// ClientConfig - The console configuration.
	ClientConfig *ConsoleConfig
)

// ConsoleConfig - The console configuration (prompts, hints, modes, etc)
type ConsoleConfig struct {
	ServerPrompt struct { // server prompt
		Right string
		Left  string
	}
	SliverPrompt struct { // session prompt
		Right string
		Left  string
	}
	Hints bool // Show hints ?
	Vim   bool // Input mode
}

// ToProtobuf - The config is exchanged between the client and the server via RPC
func (cc *ConsoleConfig) ToProtobuf() *clientpb.ConsoleConfig {
	conf := &clientpb.ConsoleConfig{
		ServerPromptRight: cc.ServerPrompt.Right,
		ServerPromptLeft:  cc.ServerPrompt.Left,
		SliverPromptRight: cc.SliverPrompt.Right,
		SliverPromptLeft:  cc.SliverPrompt.Left,
		Hints:             cc.Hints,
		Vim:               cc.Vim,
	}
	return conf
}

// LoadConsoleConfig - Once the client is connected, it receives a console
// configuration from the server, according to the user profile.
// In case of errors, it loads the builtin console configuration below.
func LoadConsoleConfig(rpc rpcpb.SliverRPCClient) (err error) {

	req := &clientpb.GetConsoleConfigReq{}
	res, err := rpc.LoadConsoleConfig(context.Background(), req, grpc.EmptyCallOption{})
	if err != nil {
		loadDefaultConsoleConfig()
		return fmt.Errorf("RPC Error: %s", err.Error())
	}
	if res.Response.Err != "" {
		loadDefaultConsoleConfig()
		return fmt.Errorf("%s", res.Response.Err)
	}
	conf := res.Config

	// Populate the configuration
	ClientConfig = &ConsoleConfig{
		ServerPrompt: struct {
			Right string
			Left  string
		}{
			Right: conf.ServerPromptRight,
			Left:  conf.ServerPromptLeft,
		},
		SliverPrompt: struct {
			Right string
			Left  string
		}{
			Right: conf.SliverPromptRight,
			Left:  conf.SliverPromptLeft,
		},
		Hints: conf.Hints,
		Vim:   conf.Vim,
	}

	return
}

// Initialize a console configuration with default values
func loadDefaultConsoleConfig() {

	// Make little adjustements to default server prompt, depending on server/client
	var ps string
	ps += readline.RESET
	if Config.LHost == "" {
		ps += "{bddg}{y} server {fw}@{local_ip} {reset}"
	} else {
		ps += "{bddg}@{server_ip}{reset}"
	}
	// Current working directory
	ps += " {dim}in {bold}{b}{cwd}"
	ps += readline.RESET

	ClientConfig = &ConsoleConfig{
		ServerPrompt: struct {
			Right string
			Left  string
		}{
			Left:  ps,
			Right: "{dim}[{reset}{y}{jobs}{fw} jobs, {b}{sessions}{fw} sessions{dim}]",
		},
		SliverPrompt: struct {
			Right string
			Left  string
		}{
			Left:  "{bddg} {fr}{session_name} {reset}{bold} {user}{dim}@{reset}{bold}{host}{reset}{dim} in{reset} {bold}{b}{wd}",
			Right: "{dim}[{reset}{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}{dim}]",
		},
		Hints: true,
		Vim:   false,
	}
}
