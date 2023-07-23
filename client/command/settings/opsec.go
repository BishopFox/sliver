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
	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// SettingsAutoAdultCmd - The client settings command.
func SettingsAutoAdultCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error
	if con.Settings == nil {
		con.Settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	con.Settings.AutoAdult = !con.Settings.AutoAdult
	con.PrintInfof("Auto Adult = %v\n", con.Settings.AutoAdult)
}

// IsUserAnAdult - This should be called for any dangerous (OPSEC-wise) functions.
func IsUserAnAdult(con *console.SliverClient) bool {
	if GetAutoAdult(con) {
		return true
	}
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

// GetAutoAdult - Get the current auto adult setting.
func GetAutoAdult(con *console.SliverClient) bool {
	if con.Settings == nil {
		con.Settings, _ = assets.LoadSettings()
	}
	if con.Settings != nil {
		return con.Settings.AutoAdult
	}
	return false
}
