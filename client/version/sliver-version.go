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
	"strconv"
	"strings"
)

const (
	// SliverVersion - Sliver version number
	SliverVersion = "1.0.0"
	normal        = "\033[0m"
	bold          = "\033[1m"
)

// SemVer - Get sematic version as int slice
func SemVer() []int {
	semVer := []int{}
	for _, part := range strings.Split(SliverVersion, ".") {
		number, _ := strconv.Atoi(part)
		semVer = append(semVer, number)
	}
	if len(semVer) != 3 {
		panic("invalid semantic version")
	}
	return semVer
}

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s - %s", SliverVersion, GitVersion)
	if GitDirty != "" {
		ver += " - " + bold + GitDirty + normal
	}
	return ver
}
