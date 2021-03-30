package main

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
	"flag"
	"os"

	"github.com/bishopfox/sliver/client/assets"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
)

// admin - We are starting a console client which will reach back to a remote server.
var admin = false

func main() {

	// Process flags passed to this binary (os.Flags). All flag variables are
	// in their respective files (but, of course, in this package only).
	flag.Parse()

	// Load all necessary configurations (server connection details, TLS security,
	// console configuration, etc.). This function automatically determines if the
	// console binary has a builtin server configuration or not, and forges a configuration
	// depending on this. The configuration is then accessible to all client packages.
	assets.LoadServerConfig()

	// Get a gRPC client connection (Mutual TLS)
	grpcConn, err := transport.ConnectTLS()
	if err != nil {
		os.Exit(1)
	}

	// Register RPC service clients. This function is called here because both the server
	// and the client binary use the same function, but the server binary uses an in-memory
	// gRPC connection, contrary to this client.
	client.Console.Connect(grpcConn)

	// Setup prompt/command/completion, event loop listening, logging, etc.
	client.Console.Init()

	// Run the console. Any error is handled internally
	client.Console.Run()
}
