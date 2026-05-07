//go:build linux

package version

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
	"bytes"
	//{{if .Config.Debug}}
	"log"
	//{{end}}

	"golang.org/x/sys/unix"
)

func linuxNullTerminatedString(input []byte) string {
	if index := bytes.IndexByte(input, 0); index >= 0 {
		input = input[:index]
	}
	return string(input)
}

// GetVersion returns the os version information
func GetVersion() string {
	osRelease := readLinuxOSRelease()
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		//{{if .Config.Debug}}
		log.Printf("error getting OS version: %v", err)
		//{{end}}
		return osRelease
	}

	return formatLinuxDetailedVersion(osRelease, linuxVersionInfo{
		Sysname: linuxNullTerminatedString(uname.Sysname[:]),
		Release: linuxNullTerminatedString(uname.Release[:]),
		Version: linuxNullTerminatedString(uname.Version[:]),
		Machine: linuxNullTerminatedString(uname.Machine[:]),
	})
}
