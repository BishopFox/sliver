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

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/transports"

	// {{if .IsSharedLib}}
	"runtime"
	"syscall"

	// {{end}}

	"github.com/golang/protobuf/proto"
)

var specialHandlers = map[uint32]SpecialHandler{
	pb.MsgKill: killHandler,
}

// GetSpecialHandlers returns the specialHandlers map
func GetSpecialHandlers() map[uint32]SpecialHandler {
	return specialHandlers
}

func killHandler(data []byte, connection *transports.Connection) error {
	killReq := &pb.KillReq{}
	err := proto.Unmarshal(data, killReq)
	// {{if .Debug}}
	println("KILL called")
	// {{end}}
	if err != nil {
		return err
	}
	// {{if .IsSharedLib}}
	if runtime.GOOS == "windows" {
		// Windows only: ExitThread() instead of os.Exit() for DLL/shellcode slivers
		// so that the parent process is not killed
		exitFunc := syscall.MustLoadDLL("kernel32.dll").MustFindProc("ExitThread")
		exitFunc.Call(uintptr(0))
		return nil
	}
	// {{else}}
	// Exit now if we've received a force request
	if killReq.Force {
		os.Exit(0)
	}
	//{{end}}
	// Cleanup connection
	connection.Cleanup()
	// {{if .Debug}}
	println("Let's exit!")
	// {{end}}
	os.Exit(0)
	return nil
}
