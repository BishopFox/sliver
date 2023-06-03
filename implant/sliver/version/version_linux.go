//go:build amd64 || 386

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
	"fmt"
	"log"
	"strings"
	"syscall"
)

func getString(input [65]int8) string {
	var buf [65]byte
	for i, b := range input {
		buf[i] = byte(b)
	}
	ver := string(buf[:])
	if i := strings.Index(ver, "\x00"); i != -1 {
		ver = ver[:i]
	}
	return ver
}

// GetVersion returns the os version information
func GetVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s %s %s", getString(uname.Sysname), getString(uname.Nodename), getString(uname.Release))
}
