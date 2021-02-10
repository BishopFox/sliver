package context

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
	"sync"
	"time"

	"github.com/evilsocket/islazy/tui"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// Context - The console context object
	Context *ConsoleContext

	// Config - The console configuration.
	Config *ConsoleConfig
)

// Menu Contexts
const (
	// Server - "Main menu" of wiregost, gives all commands and completion system
	// available for interacting with server.
	Server = "server"
	// Sliver - Available only when interacting with a sliver implant
	Sliver = "sliver"
)

// ConsoleContext - Stores all variables needed for console context
type ConsoleContext struct {
	Menu                string   // Current shell menu
	Sliver              *Session // The current implant we're interacting with
	Jobs                int      // Number of jobs
	Slivers             int      // Number of connected implants
	NeedsCommandRefresh bool     // A command might or has set this to true.
	mutex               *sync.Mutex
}

// Initialize - The console calls to initialize a new context object, to be shared by
// many components of the console system (completion, command dispatch, prompt, etc.)
func Initialize(rpc rpcpb.SliverRPCClient) {
	Context = &ConsoleContext{
		Menu:  Server,
		mutex: &sync.Mutex{},
	}

	// Normally the config should have been loaded, but if any errors arised
	// the config should still be nil: initialize it with default values.
	if Config == nil {
		loadDefaultConsoleConfig()
	}

	// Get several values from the server.
	// Jobs
	req := &commonpb.Empty{}
	res, _ := rpc.GetJobs(context.Background(), req, grpc.EmptyCallOption{})
	Context.Jobs = len(res.Active)

	// Sessions
	sReq := &commonpb.Empty{}
	sRes, _ := rpc.GetSessions(context.Background(), sReq, grpc.EmptyCallOption{})
	Context.Slivers = len(sRes.Sessions)

	return
}

// Session - An implant session we are interacting with.
// This is a wrapper for some utility methods.
type Session struct {
	*clientpb.Session
	WorkingDir string // The implant working directory, stored to limit calls.
}

// Request - Prepare a RPC request for the current Session.
func (s *Session) Request(timeOut int) *commonpb.Request {
	if s.Session == nil {
		return nil
	}
	timeout := int(time.Second) * timeOut
	return &commonpb.Request{
		SessionID: s.ID,
		Timeout:   int64(timeout),
	}
}

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
func LoadConsoleConfig(rpc rpcpb.SliverRPCClient) (err error) {

	req := &clientpb.GetConsoleConfigReq{}
	res, err := rpc.LoadConsoleConfig(context.Background(), req, grpc.EmptyCallOption{})
	if err != nil {
		return fmt.Errorf("RPC Error: %s", err.Error())
	}
	if res.Response.Err != "" {
		return fmt.Errorf("%s", res.Response.Err)
	}
	conf := res.Config

	// Populate the configuration
	Config = &ConsoleConfig{
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
	ps += tui.RESET
	if assets.Config.LHost == "" {
		ps += "{bddg}{y} server {fw}@{local_ip} {reset}"
	} else {
		ps += "{bddg}@{server_ip}{reset}"
	}
	// Current working directory
	ps += " {dim}in {bold}{b}{cwd}"
	ps += tui.RESET

	Config = &ConsoleConfig{
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
			Left:  "{bddg} {fr}{session_name} {reset}{bold} {user}{dim}@{reset}{bold}{host}{reset}{dim} in{reset} {bold}{b}{cwd}",
			Right: "{dim}[{reset}{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}{dim}]",
		},
		Hints: true,
		Vim:   false,
	}
}
