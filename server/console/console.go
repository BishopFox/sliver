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
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/maxlandon/gonsole"
	"google.golang.org/grpc"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	clientconsole "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/transport"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
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
	checkForLegacyDB()
	clientconsole.Start(localRPC, serverOnlyCmds, &clientAssets.ClientConfig{})
}

func checkForLegacyDB() {
	legacyDBPath := filepath.Join(assets.GetRootAppDir(), "db")
	if _, err := os.Stat(legacyDBPath); !os.IsNotExist(err) {
		fmt.Println("\n" + Warn + bold + "Compatability Warning: " + normal)
		fmt.Println(Warn + "It looks like this server was upgraded from an older version of Sliver.")
		fmt.Println(Warn + "We have switched to using SQL for internal data (before we used BadgerDB)")
		fmt.Printf(Warn+"Regrettably this means there's %sno backwards compatibility%s for implants, etc.\n", bold, normal)
		fmt.Println(Warn + "If you need to use existing implants, stick with any version up to 1.0.9")
		fmt.Println()
		confirm := false
		survey.AskOne(&survey.Confirm{
			Message: "Delete old database (CANNOT BE UNDONE)",
		}, &confirm)
		if confirm {
			os.RemoveAll(legacyDBPath)
		}
	}
}

// serverOnlyCmds - We bind commands only available to the server admin to the console command parser.
// Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// There is a namespace field, however it messes up with the option printing/detection/parsing.
func serverOnlyCmds(cc *gonsole.Menu) {

	cc.AddCommand(constants.NewPlayerStr,
		"Create a new player config file",
		help.GetHelpFor(constants.NewPlayerStr),
		constants.AdminGroup,
		[]string{""},
		func() interface{} { return &NewOperator{} })

	cc.AddCommand(constants.KickPlayerStr,
		"Kick a player from the server",
		help.GetHelpFor(constants.KickPlayerStr),
		constants.AdminGroup,
		[]string{""},
		func() interface{} { return &KickOperator{} })

	cc.AddCommand(constants.MultiplayerModeStr,
		"Enable multiplayer mode on this server",
		help.GetHelpFor(constants.MultiplayerModeStr),
		constants.AdminGroup,
		[]string{""},
		func() interface{} { return &MultiplayerMode{} })

	return
}
