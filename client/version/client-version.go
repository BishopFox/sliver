package version

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

import "fmt"

const (
	// ClientVersion - Client version number
	ClientVersion = "0.0.7"
	normal        = "\033[0m"
	bold          = "\033[1m"
)

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s - %s", ClientVersion, GitVersion)
	if GitDirty != "" {
		ver += " - " + bold + GitDirty + normal
	}
	return ver
}
