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
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"golang.org/x/crypto/ssh"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	clientconsole "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = 2 * gb
)

func StartAlt() {
	// Process flags passed to this binary (os.Flags). All flag variables are
	// in their respective files (but, of course, in this package only).
	flag.Parse()

	// Make a fingerprint of the implant's private key, for SSH-layer authentication
	_, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	signer, _ := ssh.ParsePrivateKey(serverCAKey)
	keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
	fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])

	clientAssets.ServerPrivateKey = string(serverCAKey)
	clientAssets.CommFingerprint = fingerprint

	// Load all necessary configurations (server connection details, TLS security,
	// console configuration, etc.). This function automatically determines if the
	// console binary has a builtin server configuration or not, and forges a configuration
	// depending on this. The configuration is then accessible to all client packages.
	// clientAssets.LoadServerConfig()

	// Start the client console. The latter automatically performs server connection,
	// prompt/command/completion setup, event loop listening, logging, etc. Any critical error
	// is handled from within this function, so we don't process the return error here.
	clientconsole.Console.Start(true)
}

// Start - Starts the server console
// func Start() {
//         _, ln, _ := transport.LocalListener()
//         ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
//                 return ln.Dial()
//         })
//
//         options := []grpc.DialOption{
//                 ctxDialer,
//                 grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
//                 grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
//         }
//         conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
//         if err != nil {
//                 fmt.Printf(util.Warn+"Failed to dial bufnet: %s", err)
//                 return
//         }
//         defer conn.Close()
//         // localRPC := rpcpb.NewSliverRPCClient(conn)
//         checkForLegacyDB()
//
//         // Make a fingerprint of the implant's private key, for SSH-layer authentication
//         _, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
//         signer, _ := ssh.ParsePrivateKey(serverCAKey)
//         keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
//         fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])
//
//         // Start the console locally with appropriate SSH credentials (server)
//         // clientconsole.Start(localRPC, serverOnlyCmds, serverCAKey, fingerprint)
//         // clientconsole.Start(localRPC, serverOnlyCmds, serverCAKey, fingerprint)
// }

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
			// startMultiplayerModeCmd(ctx)
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
			// newOperatorCmd(ctx)
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
			// kickOperatorCmd(ctx)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	// client.Console.StartServerConsole(conn)
}

//
// // BindServerAdminCommands - We bind commands only available to the server admin to the console command parser.
// // Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// // There is a namespace field, however it messes up with the option printing/detection/parsing.
// func BindServerAdminCommands() (err error) {
//
//         np, err := commands.Server.AddCommand(constants.NewPlayerStr, "Create a new player config file",
//                 help.GetHelpFor(constants.NewPlayerStr), &NewOperator{})
//         np.Aliases = []string{"admin"}
//         if err != nil {
//                 fmt.Println(util.Warn + err.Error())
//                 os.Exit(3)
//         }
//
//         kp, err := commands.Server.AddCommand(constants.KickPlayerStr, "Kick a player from the server",
//                 help.GetHelpFor(constants.KickPlayerStr), &KickOperator{})
//         kp.Aliases = []string{"admin"}
//         if err != nil {
//                 fmt.Println(util.Warn + err.Error())
//                 os.Exit(3)
//         }
//
//         mm, err := commands.Server.AddCommand(constants.MultiplayerModeStr, "Enable multiplayer mode on this server",
//                 help.GetHelpFor(constants.MultiplayerModeStr), &MultiplayerMode{})
//         mm.Aliases = []string{"admin"}
//         if err != nil {
//                 fmt.Println(util.Warn + err.Error())
//                 os.Exit(3)
//         }
//
//         return
// }
