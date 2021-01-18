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
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
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

func Start() {
	// Process flags passed to this binary (os.Flags). All flag variables are
	// in their respective files (but, of course, in this package only).
	flag.Parse()

	// Make a fingerprint of the implant's private key, for SSH-layer authentication
	_, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	signer, _ := ssh.ParsePrivateKey(serverCAKey)
	keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
	fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])

	// Load only needed fields in the client assets (config) package.
	clientAssets.ServerPrivateKey = string(serverCAKey)
	clientAssets.CommFingerprint = fingerprint

	// Get a gRPC client connection (in-memory listener)
	grpcConn, err := connectLocal()
	if err != nil {
		os.Exit(1)
	}

	// Register RPC service clients, monitor incoming events and start the client's Comm System.
	client.Console.Connect(grpcConn, true)

	// Start the client console. The latter automatically performs server connection,
	// prompt/command/completion setup, event loop listening, logging, etc. We start as admin = true
	client.Console.Start()
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
