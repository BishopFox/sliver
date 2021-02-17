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
	"time"
)

const (
	normal = "\033[0m"
	bold   = "\033[1m"
)

var (
	// Version - The semantic version in string form
	Version string

	// GoVersion - Go compiler version
	GoVersion string

	// GitCommit - The commit id at compile time
	GitCommit string

	// GitDirty - Was the commit dirty at compile time
	GitDirty string

	// CompiledAt - When was this binary compiled
	CompiledAt string
)

// SemanticVersion - Get the structured sematic version
func SemanticVersion() []int {
	semVer := []int{}
	version := Version
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	for _, part := range strings.Split(version, ".") {
		number, _ := strconv.Atoi(part)
		semVer = append(semVer, number)
	}
	return semVer
}

// Compiled - Get time this binary was compiled
func Compiled() (time.Time, error) {
	compiled, err := strconv.ParseInt(CompiledAt, 10, 64)
	if err != nil {
		return time.Unix(0, 0), err
	}
	return time.Unix(compiled, 0), nil
}

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s", Version)
	compiled, err := Compiled()
	if err != nil {
		ver += fmt.Sprintf(" - Compiled %s", compiled.String())
	}
	if GitCommit != "" {
		ver += fmt.Sprintf(" - %s", GitCommit)
	}
	return ver
}
