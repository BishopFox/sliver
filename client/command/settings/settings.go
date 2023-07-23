package settings

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// SettingsCmd - The client settings command.
func SettingsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	tw := table.NewWriter()
	tw.SetStyle(GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Value", "Description"})
	tw.AppendRow(table.Row{"Tables", con.Settings.TableStyle, "Set the stylization of tables"})
	tw.AppendRow(table.Row{"Auto Adult", con.Settings.AutoAdult, "Automatically accept OPSEC warnings"})
	tw.AppendRow(table.Row{"Auto Beacon Results", con.Settings.BeaconAutoResults, "Automatically display beacon results when tasks complete"})
	tw.AppendRow(table.Row{"Small Term Width", con.Settings.SmallTermWidth, "Omit some table columns when terminal width is less than this value"})
	tw.AppendRow(table.Row{"Always Overflow", con.Settings.AlwaysOverflow, "Disable table pagination"})
	tw.AppendRow(table.Row{"Vim Mode", con.Settings.VimMode, "Navigation mode, vim style"})
	tw.AppendRow(table.Row{"Console Logs", con.Settings.ConsoleLogs, "Log console output to disk"})
	con.Printf("%s\n", tw.Render())
}

// SettingsAlwaysOverflow - Toggle always overflow.
func SettingsAlwaysOverflow(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	con.Settings.AlwaysOverflow = !con.Settings.AlwaysOverflow
	con.PrintInfof("Always overflow = %v\n", con.Settings.AlwaysOverflow)
}

// SettingsConsoleLogs - Toggle console logs.
func SettingsConsoleLogs(cmd *cobra.Command, con *console.SliverClient) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	con.Settings.ConsoleLogs = !con.Settings.ConsoleLogs
	con.PrintInfof("Console Logs = %v\n", con.Settings.ConsoleLogs)
}

// SettingsSmallTerm - Modify small terminal width value.
func SettingsSmallTerm(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	result := ""
	prompt := &survey.Input{Message: "Set small width:"}
	err = survey.AskOne(prompt, &result)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	newWidth, err := strconv.Atoi(result)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if newWidth < 1 {
		con.PrintErrorf("Invalid width (too small)\n")
		return
	}
	con.Settings.SmallTermWidth = newWidth
	con.PrintInfof("Small terminal width set to %d\n", con.Settings.SmallTermWidth)
}

// SettingsTablesCmd - The client settings command.
func SettingsTablesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	options := []string{}
	for option := range tableStyles {
		options = append(options, option)
	}
	style := ""
	prompt := &survey.Select{
		Message: "Choose a style:",
		Options: options,
	}
	err = survey.AskOne(prompt, &style)
	if err != nil {
		con.PrintErrorf("No selection\n")
		return
	}
	if _, ok := tableStyles[style]; ok {
		con.Settings.TableStyle = style
	} else {
		con.PrintErrorf("Invalid style\n")
	}
}

// SettingsSaveCmd - The client settings command.
func SettingsSaveCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	err = assets.SaveSettings(con.Settings)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Settings saved to disk\n")
	}
}

// SettingsAlwaysOverflow - Toggle always overflow.
func SettingsUserConnect(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	con.Settings.UserConnect = !con.Settings.UserConnect
	con.PrintInfof("User connect events = %v\n", con.Settings.UserConnect)
}
