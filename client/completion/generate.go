package completion

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

import "github.com/maxlandon/readline"

// CompleteStagePlatforms - Returns a list of all valid NATIVE Go compiler targets, thus all valid
// target OS/arch combinations supported by Sliver stage implants. F*****g cool.
func CompleteStagePlatforms() (completions []*readline.CompletionGroup) {

	comp := &readline.CompletionGroup{
		Name:         "native target platforms",
		Suggestions:  platforms,
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
		MaxLength:    20,
	}

	return []*readline.CompletionGroup{comp}
}

// CompleteStageFormats - Returns the list of compatible stage binary formats.
func CompleteStageFormats() (completions []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "stage binary formats",
		Suggestions:  formats,
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
		MaxLength:    20,
	}
	return []*readline.CompletionGroup{comp}
}

var (
	// Stages
	formats = []string{"exe", "shared", "service", "shellcode"}

	// Reference:
	// https://github.com/golang/go/blob/master/src/go/build/syslist.go
	platforms = []string{
		"aix/ppc64",
		"darwin/amd64",
		"darwin/arm64",
		"dragonfly/amd64",
		"freebsd/386",
		"freebsd/amd64",
		"freebsd/arm",
		"freebsd/arm64",
		"illumos/amd64",
		"js/wasm",
		"linux/386",
		"linux/amd64",
		"linux/arm",
		"linux/arm64",
		"linux/ppc64",
		"linux/ppc64le",
		"linux/mips",
		"linux/mipsle",
		"linux/mips64",
		"linux/mips64le",
		"linux/riscv64",
		"linux/s390x",
		"netbsd/386",
		"netbsd/amd64",
		"netbsd/arm",
		"netbsd/arm64",
		"openbsd/386",
		"openbsd/amd64",
		"openbsd/arm",
		"openbsd/arm64",
		"plan9/386",
		"plan9/amd64",
		"plan9/arm",
		"solaris/amd64",
		"windows/386",
		"windows/amd64",
		"windows/arm",
	}
)
