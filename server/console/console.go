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

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/help"
	clientconsole "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/configs"
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
		fmt.Printf(Warn+"Failed to dial bufnet: %s\n", err)
		return
	}
	defer conn.Close()
	localRPC := rpcpb.NewSliverRPCClient(conn)
	if err := configs.CheckHTTPC2ConfigErrors(); err != nil {
		fmt.Printf(Warn+"Error in HTTP C2 config: %s\n", err)
	}
	clientconsole.Start(localRPC, command.BindCommands, serverOnlyCmds, true)
}

// ServerOnlyCmds - Server only commands
func serverOnlyCmds(console *clientconsole.SliverConsoleClient) {

	// [ Multiplayer ] -----------------------------------------------------------------

	console.App.AddCommand(&grumble.Command{
		Name:     consts.MultiplayerModeStr,
		Help:     "Enable multiplayer mode",
		LongHelp: help.GetHelpFor([]string{consts.MultiplayerModeStr}),
		Flags: func(f *grumble.Flags) {
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", 31337, "tcp listen port")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startMultiplayerModeCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	console.App.AddCommand(&grumble.Command{
		Name:     consts.NewOperatorStr,
		Help:     "Create a new operator config file",
		LongHelp: help.GetHelpFor([]string{consts.NewOperatorStr}),
		Flags: func(f *grumble.Flags) {
			f.String("l", "lhost", "", "listen host")
			f.Int("p", "lport", 31337, "listen port")
			f.String("s", "save", "", "directory/file to the binary to")
			f.String("n", "name", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			newOperatorCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	console.App.AddCommand(&grumble.Command{
		Name:     consts.KickOperatorStr,
		Help:     "Kick an operator from the server",
		LongHelp: help.GetHelpFor([]string{consts.KickOperatorStr}),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "operator name")
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
