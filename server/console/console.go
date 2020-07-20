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

	"github.com/desertbit/grumble"

	clientconsole "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/transport"
	"google.golang.org/grpc"
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
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(clienttransport.ClientMaxReceiveMessageSize)),
	}
	conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
	if err != nil {
		fmt.Printf(Warn+"Failed to dial bufnet: %s", err)
		return
	}
	defer conn.Close()
	localRPC := rpcpb.NewSliverRPCClient(conn)
	clientconsole.Start(localRPC, serverOnlyCmds)
}

// ServerOnlyCmds - Server only commands
func serverOnlyCmds(app *grumble.App, _ rpcpb.SliverRPCClient) {

	// [ Multiplayer ] -----------------------------------------------------------------

	app.AddCommand(&grumble.Command{
		Name:     consts.MultiplayerModeStr,
		Help:     "Enable multiplayer mode",
		LongHelp: help.GetHelpFor(consts.MultiplayerModeStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", 31337, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startMultiplayerModeCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.NewPlayerStr,
		Help:     "Create a new player config file",
		LongHelp: help.GetHelpFor(consts.NewPlayerStr),
		Flags: func(f *grumble.Flags) {
			f.String("l", "lhost", "", "listen host")
			f.Int("p", "lport", 31337, "listen port")
			f.String("s", "save", "", "directory/file to the binary to")
			f.String("n", "operator", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			newOperatorCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.KickPlayerStr,
		Help:     "Kick a player from the server",
		LongHelp: help.GetHelpFor(consts.KickPlayerStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "operator", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			kickOperatorCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

}
