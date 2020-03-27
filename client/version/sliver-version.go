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

import (
	"fmt"
)

const (
	normal = "\033[0m"
	bold   = "\033[1m"
)

var (
	// SemanticVersion - Sliver version number
	SemanticVersion = []int{1, 0, 0}
)

// SliverVersion - Get sematic version as int slice
func SliverVersion() string {
	return fmt.Sprintf("%d.%d.%d",
		SemanticVersion[0],
		SemanticVersion[1],
		SemanticVersion[2],
	)
}

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s - %s", SliverVersion(), GitVersion)
	if GitDirty != "" {
		ver += " - " + bold + GitDirty + normal
	}
	return ver
}
