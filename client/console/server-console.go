package console

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
	"errors"
	"log"

	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

// StartServerConsole - This function is called by the server binary, which also and obviously needs a full
// client access, but locally. This function is therefore a modified/condensed version of Console.Start()
func (c *console) StartServerConsole(conn *grpc.ClientConn) (err error) {

	// Register RPC Service Client through local conn parameter
	connection.RPC = rpcpb.NewSliverRPCClient(conn)
	if connection.RPC == nil {
		return errors.New("Could not register gRPC Client, instance is nil.")
	}

	// Listen for incoming server/implant events. If an error occurs in this
	// loop, the console will exit after logging it.
	go c.startEventHandler()

	// Start message tunnel loop.
	go connection.TunnelLoop()

	// Print banner and version information.
	// This will also check the last update time.
	printLogo()

	// Setup console elements
	err = c.setup()
	if err != nil {
		log.Fatalf(Error+"Console setup failed: %s", err)
	}

	// Start input loop
	for {
		// Recompute prompt each time, before anything.
		Prompt.Compute()

		// Read input line
		line, _ := c.Readline()

		// Split and sanitize input
		sanitized, empty := sanitizeInput(line)
		if empty {
			continue
		}

		// Process various tokens on input (environment variables, paths, etc.)
		parsed, _ := util.ParseEnvironmentVariables(sanitized)

		// Execute the command input: all input is passed to the current
		// context parser, which will deal with it on its own. We never return
		// errors from this call, as any of them happening follows a certain
		// number of fallbacks (special commands, error printing, etc.).
		// We should not have to exit the console because of an error here.
		c.ExecuteCommand(parsed)
	}
}
