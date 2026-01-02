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
	"errors"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
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

	if shouldRunSettingsForm(cmd, con, args) {
		tableStyleOptions := make([]string, 0, len(tableStyles))
		for option := range tableStyles {
			tableStyleOptions = append(tableStyleOptions, option)
		}
		result, err := forms.SettingsForm(con.Settings, tableStyleOptions)
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Settings form failed: %s\n", err)
			return
		}
		if err := applySettingsForm(con.Settings, result); err != nil {
			con.PrintErrorf("Failed to apply settings form values: %s\n", err)
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
	tw.AppendRow(table.Row{"User Connect", con.Settings.UserConnect, "Show operator connect/disconnect events"})
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
	err = forms.Input("Set small width:", &result)
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
	err = forms.Select("Choose a style:", options, &style)
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

func shouldRunSettingsForm(cmd *cobra.Command, con *console.SliverClient, args []string) bool {
	if con == nil || con.IsCLI {
		return false
	}
	if len(args) != 0 {
		return false
	}
	if show, err := cmd.Flags().GetBool("show"); err == nil && show {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func applySettingsForm(settings *assets.ClientSettings, result *forms.SettingsFormResult) error {
	if settings == nil {
		return errors.New("settings are required")
	}
	if result == nil {
		return errors.New("settings form result is required")
	}
	if _, ok := tableStyles[result.TableStyle]; !ok {
		return errors.New("invalid table style")
	}
	width, err := strconv.Atoi(strings.TrimSpace(result.SmallTermWidth))
	if err != nil {
		return err
	}
	if width < 1 {
		return errors.New("small terminal width must be 1 or greater")
	}

	settings.TableStyle = result.TableStyle
	settings.AutoAdult = result.AutoAdult
	settings.BeaconAutoResults = result.BeaconAutoResults
	settings.SmallTermWidth = width
	settings.AlwaysOverflow = result.AlwaysOverflow
	settings.VimMode = result.VimMode
	settings.UserConnect = result.UserConnect
	settings.ConsoleLogs = result.ConsoleLogs
	return nil
}
