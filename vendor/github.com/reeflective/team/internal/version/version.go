package version

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	semVerLen = 3
)

// ErrNoBuildInfo is an error indicating that we could not fetch any binary build info.
var ErrNoBuildInfo = errors.New("No binary build info")

// Semantic - Get the structured semantic
// version of the application binary.
func Semantic() []int {
	semVer := make([]int, semVerLen)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return semVer
	}

	version := info.Main.Version

	for i, part := range strings.Split(version, ".") {
		number, _ := strconv.ParseInt(part, 10, 32)
		semVer[i] = int(number)
	}

	return semVer
}

// Compiled - Get time this binary was compiled.
func Compiled() (time.Time, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return time.Unix(0, 0), ErrNoBuildInfo
	}

	var compiledAt string

	for _, set := range info.Settings {
		if set.Key == "vcs.time" {
			compiledAt = set.Value
			break
		}
	}

	compiled, err := strconv.ParseInt(compiledAt, 10, 64)
	if err != nil {
		return time.Unix(0, 0), err
	}

	return time.Unix(compiled, 0), nil
}

// GitCommit returns the last commit hash.
func GitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, set := range info.Settings {
		if set.Key == "vcs.revision" {
			return set.Value
		}
	}

	return ""
}

// GitDirty returns true if the binary was compiled
// with modified files in the VCS working area.
func GitDirty() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}

	for _, set := range info.Settings {
		if set.Key == "vcs.modified" {
			return set.Key == "true"
		}
	}

	return false
}
