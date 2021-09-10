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
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// SettingsBeaconsAutoResultCmd - The client settings command
func SettingsBeaconsAutoResultCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var err error
	if settings == nil {
		settings, err = assets.LoadSettings()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	settings.BeaconAutoResults = !settings.BeaconAutoResults
	con.PrintInfof("Beacon Auto Result = %v\n", settings.BeaconAutoResults)
}

// GetBeaconAutoResults - Get the current auto adult setting
func GetBeaconAutoResults() bool {
	if settings == nil {
		settings, _ = assets.LoadSettings()
	}
	if settings != nil {
		return settings.BeaconAutoResults
	}
	return true
}
