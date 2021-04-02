package completers

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

// Command/option argument choices
var (
	// Logs & components
	logLevels = []string{"trace", "debug", "info", "warning", "error"}
	loggers   = []string{"client"}

	// Stages
	implantFmt = []string{"exe", "shared", "service", "shellcode"}

	// Reference:
	// https://github.com/golang/go/blob/master/src/go/build/syslist.go
	osArchs = []string{
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

	// Stagers
	stageListenerProtocols = []string{"tcp", "http", "https"}

	// Assembly architectures
	assemblyArchs = []string{"x86", "x64", "x84"}

	// Comm network protocols
	portfwdProtocols   = []string{"tcp", "udp"}
	transportProtocols = []string{"tcp", "udp", "ip"}
)
