package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/backdoor"
	"github.com/bishopfox/sliver/client/command/cursed"
	"github.com/bishopfox/sliver/client/command/dllhijack"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/pivots"
	"github.com/bishopfox/sliver/client/command/portfwd"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/reconfig"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/rportfwd"
	"github.com/bishopfox/sliver/client/command/screenshot"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/shell"
	"github.com/bishopfox/sliver/client/command/socks"
	"github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/client/command/wasm"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// SliverCommands returns all commands bound to the implant menu.
func SliverCommands(con *client.SliverConsoleClient) console.Commands {
	sliverCommands := func() *cobra.Command {
		sliver := &cobra.Command{
			Short: "Implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		// [ Core ]
		bind(consts.SliverCoreHelpGroup, sliver, con,
			reconfig.Commands,
			// sessions.Commands,
			sessions.SliverCommands,
			kill.Commands,
			// use.Commands,
			tasks.Commands,
			pivots.Commands,
		)

		// [ Info ]
		bind(consts.InfoHelpGroup, sliver, con,
			// info.Commands,
			info.SliverCommands,
			screenshot.Commands,
			environment.Commands,
			registry.Commands,
		)

		// [ Filesystem ]
		bind(consts.FilesystemHelpGroup, sliver, con,
			filesystem.Commands,
		)

		// [ Network tools ]
		bind(consts.NetworkHelpGroup, sliver, con,
			network.Commands,
			rportfwd.Commands,
			portfwd.Commands,
			socks.Commands,
			wireguard.SliverCommands,
		)

		// [ Execution ]
		bind(consts.ExecutionHelpGroup, sliver, con,
			shell.Commands,
			exec.Commands,
			backdoor.Commands,
			dllhijack.Commands,
			cursed.Commands,
			wasm.Commands,
		)

		// [ Privileges ]
		bind(consts.PrivilegesHelpGroup, sliver, con,
			privilege.Commands,
		)

		// [ Processes ]
		bind(consts.ProcessHelpGroup, sliver, con,
			processes.Commands,
		)

		// [ Aliases ]
		bind(consts.AliasHelpGroup, sliver, con)

		// [ Extensions ]
		bind(consts.ExtensionHelpGroup, sliver, con,
			extensions.Commands,
		)

		// [ Post-command declaration setup ]----------------------------------------

		// Load Aliases
		aliasManifests := assets.GetInstalledAliasManifests()
		for _, manifest := range aliasManifests {
			_, err := alias.LoadAlias(manifest, sliver, con)
			if err != nil {
				con.PrintErrorf("Failed to load alias: %s", err)
				continue
			}
		}

		// Load Extensions
		extensionManifests := assets.GetInstalledExtensionManifests()
		for _, manifest := range extensionManifests {
			ext, err := extensions.LoadExtensionManifest(manifest)
			// Absorb error in case there's no extensions manifest
			if err != nil {
				con.PrintErrorf("Failed to load extension: %s", err)
				continue
			}

			extensions.ExtensionRegisterCommand(ext, sliver, con)
		}

		// [ Post-command declaration setup ]----------------------------------------

		// Everything below this line should preferably not be any command binding
		// (although you can do so without fear). If there are any final modifications
		// to make to the server menu command tree, it time to do them here.

		sliver.InitDefaultHelpCmd()
		sliver.SetHelpCommandGroupID(consts.SliverCoreHelpGroup)

		// Compute which commands should be available based on the current session/beacon.
		con.ExposeCommands()

		return sliver
	}

	return sliverCommands
}
