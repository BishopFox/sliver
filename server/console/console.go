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
	"context"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/commands"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/server/transport"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = 2 * gb
)

// Start - Starts the server console
func Start() {
	_, ln, _ := transport.LocalListener()
	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return ln.Dial()
	})

	options := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to dial bufnet: %s", err)
		return
	}
	defer conn.Close()

	// Bind admin commands to the client console.
	// NOTE: it's normal that this is called before the console is instantiated:
	// it's because command parsers have a life of their own.
	// This needs to be called now, as the call just next is a blocking one.
	addServerAdminCommands()

	// We use a custom version of the client.Console.Start()
	// function, accomodating for needs server-side.
	client.Console.StartServerConsole(conn)

}

// addServerAdminCommands - We bind commands only available to the server admin to the console command parser.
// Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// There is a namespace field, however it messes up with the option printing/detection/parsing.
func addServerAdminCommands() (err error) {

	np, err := commands.Server.AddCommand(constants.NewPlayerStr, "Create a new player config file",
		help.GetHelpFor(constants.NewPlayerStr), &NewOperator{})
	np.Aliases = []string{"Admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	kp, err := commands.Server.AddCommand(constants.KickPlayerStr, "Kick a player from the server",
		help.GetHelpFor(constants.KickPlayerStr), &KickOperator{})
	kp.Aliases = []string{"Admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	mm, err := commands.Server.AddCommand(constants.MultiplayerModeStr, "Enable multiplayer mode on this server",
		help.GetHelpFor(constants.MultiplayerModeStr), &MultiplayerMode{})
	mm.Aliases = []string{"Admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	return
}
