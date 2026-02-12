package settings

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// SettingsCmd - The client settings command.
// SettingsCmd - The 客户端设置 command.
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
		if err := assets.SaveSettings(con.Settings); err != nil {
			con.PrintErrorf("Failed to save settings: %s\n", err)
			return
		}
		if assets.NormalizePromptStyle(result.PromptStyle) == assets.PromptStyleCustom {
			settingsPath := filepath.Join(assets.GetRootAppDir(), "tui-settings.yaml")
			con.PrintInfof("Prompt style is %q. Edit %s to set %q.\n", assets.PromptStyleCustom, settingsPath, "prompt_template")
			con.Println("")
		}
	}

	tw := table.NewWriter()
	tw.SetStyle(GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Value", "Description"})
	tw.AppendRow(table.Row{"Tables", con.Settings.TableStyle, "Set the stylization of tables"})
	tw.AppendRow(table.Row{"Prompt", con.Settings.PromptStyle, "Set the console prompt prefix style"})
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
// SettingsAlwaysOverflow - Toggle 始终 overflow.
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
// SettingsConsoleLogs - Toggle 控制台 logs.
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
// SettingsSmallTerm - Modify 小端子宽度 value.
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
// SettingsTablesCmd - The 客户端设置 command.
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
// SettingsSaveCmd - The 客户端设置 command.
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
// SettingsAlwaysOverflow - Toggle 始终 overflow.
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
	promptStyle := assets.NormalizePromptStyle(result.PromptStyle)

	settings.TableStyle = result.TableStyle
	settings.AutoAdult = result.AutoAdult
	settings.BeaconAutoResults = result.BeaconAutoResults
	settings.SmallTermWidth = width
	settings.AlwaysOverflow = result.AlwaysOverflow
	settings.VimMode = result.VimMode
	settings.UserConnect = result.UserConnect
	settings.ConsoleLogs = result.ConsoleLogs
	settings.PromptStyle = promptStyle
	return nil
}
