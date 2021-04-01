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
	"sync"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

var (
	// Context - The console context object
	Context *ConsoleContext
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

// InitializeConsole - The console calls to initialize a new context object, to be shared by
// many components of the console system (completion, command dispatch, prompt, etc.)
func InitializeConsole(rpc rpcpb.SliverRPCClient) {
	Context = &ConsoleContext{
		Menu:  Server,
		mutex: &sync.Mutex{},
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
