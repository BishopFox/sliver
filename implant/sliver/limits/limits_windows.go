//go:build 386 || amd64 || arm64

package limits

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
	// {{if .Config.Debug}}
	"log"
	// {{else}}{{end}}
	"os"
	"syscall"

	// {{if .Config.LimitDomainJoined}}
	"unsafe"
	// {{else}}{{end}}
)

// {{if .Config.LimitDomainJoined}}

func isDomainJoined() (bool, error) {
	var domain *uint16
	var status uint32
	err := syscall.NetGetJoinInformation(nil, &domain, &status)
	if err != nil {
		return false, err
	}
	syscall.NetApiBufferFree((*byte)(unsafe.Pointer(domain)))
	return status == syscall.NetSetupDomainName, nil
}

// {{end}}

func PlatformLimits() {
	kernel32 := syscall.MustLoadDLL("kernel32.dll")
	isDebuggerPresent := kernel32.MustFindProc("IsDebuggerPresent")
	var nargs uintptr = 0
	ret, _, _ := isDebuggerPresent.Call(nargs)
	// {{if .Config.Debug}}
	log.Printf("IsDebuggerPresent = %#v\n", int32(ret))
	// {{end}}
	if int32(ret) != 0 {
		os.Exit(1)
	}
}
