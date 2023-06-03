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

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
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

	con := console.NewConsole(false)
	console.StartClient(con, localRPC, command.ServerCommands(con, nil), command.SliverCommands(con), true)

	con.App.Start()
}

// serverOnlyCmds - Server only commands
func serverOnlyCmds() (commands []*cobra.Command) {
	// [ Multiplayer ] -----------------------------------------------------------------

	startMultiplayer := &cobra.Command{
		Use:     consts.MultiplayerModeStr,
		Short:   "Enable multiplayer mode",
		Long:    help.GetHelpFor([]string{consts.MultiplayerModeStr}),
		Run:     startMultiplayerModeCmd,
		GroupID: consts.MultiplayerHelpGroup,
	}
	command.Flags("multiplayer", false, startMultiplayer, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint16P("lport", "l", 31337, "tcp listen port")
		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})

	commands = append(commands, startMultiplayer)

	newOperator := &cobra.Command{
		Use:     consts.NewOperatorStr,
		Short:   "Create a new operator config file",
		Long:    help.GetHelpFor([]string{consts.NewOperatorStr}),
		Run:     newOperatorCmd,
		GroupID: consts.MultiplayerHelpGroup,
	}
	command.Flags("operator", false, newOperator, func(f *pflag.FlagSet) {
		f.StringP("lhost", "l", "", "listen host")
		f.Uint16P("lport", "p", 31337, "listen port")
		f.StringP("save", "s", "", "directory/file in which to save config")
		f.StringP("name", "n", "", "operator name")
	})
	command.FlagComps(newOperator, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionDirectories()
	})
	commands = append(commands, newOperator)

	kickOperator := &cobra.Command{
		Use:     consts.KickOperatorStr,
		Short:   "Kick an operator from the server",
		Long:    help.GetHelpFor([]string{consts.KickOperatorStr}),
		Run:     kickOperatorCmd,
		GroupID: consts.MultiplayerHelpGroup,
	}

	command.Flags("operator", false, kickOperator, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "operator name")
	})
	commands = append(commands, kickOperator)

	return
}
