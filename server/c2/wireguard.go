package c2

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/netstack"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"google.golang.org/protobuf/proto"
)

var (
	wgLog = log.NamedLogger("c2", "wg")
	tunIP = "100.64.0.1" // Don't let user configure this for now
)

// StartWGListener - First creates an inet.af network stack.
// then creates a Wireguard device/interface and applies configuration.
// Go routines are spun up to handle key exchange connections, as well
// as c2 comms connections.
func StartWGListener(port uint16, netstackPort uint16, keyExchangeListenPort uint16) (net.Listener, *device.Device, *bytes.Buffer, error) {
	wgLog.Infof("Starting Wireguard listener on port: %d", port)

	tun, tNet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(tunIP)},
		[]netip.Addr{netip.MustParseAddr("127.0.0.1")}, // We don't use DNS in the WG listener. Yet.
		1420,
	)
	if err != nil {
		wgLog.Errorf("CreateNetTUN failed: %v", err)
		return nil, nil, nil, err
	}

	// Get existing server wg keys
	privateKey, _, err := certs.GetWGServerKeys()

	if err != nil {
		isPeer := false
		privateKey, _, err = certs.GenerateWGKeys(isPeer, "")
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// This is currently set to silence all logs from the wg device
	// Set this to device.LogLevelVerbose when debugging for verbose logs
	// We should probably set this to LogLevelError and figure out how to
	// redirect the logs from stdout
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, "[c2/wg] "))

	wgConf := bytes.NewBuffer(nil)
	fmt.Fprintf(wgConf, "private_key=%s\n", privateKey)
	fmt.Fprintf(wgConf, "listen_port=%d\n", port)

	peers, err := certs.GetWGPeers()
	if err != nil && err != certs.ErrWGPeerDoesNotExist {
		return nil, nil, nil, err
	}

	for k, v := range peers {
		fmt.Fprintf(wgConf, "public_key=%s\n", k)
		fmt.Fprintf(wgConf, "allowed_ip=%s/32\n", v)
	}

	// Set wg device config
	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		return nil, nil, nil, err
	}

	err = dev.Up()
	if err != nil {
		wgLog.Errorf("Could not set up the device: %v", err)
		return nil, nil, nil, err
	}

	// Open up key exchange TCP socket
	keyExchangeListener, err := tNet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(tunIP), Port: int(keyExchangeListenPort)})
	if err != nil {
		wgLog.Errorf("Failed to setup up wg key exchange listener: %v", err)
		return nil, nil, nil, err
	}
	wgLog.Printf("Successfully setup up wg key exchange listener")
	go acceptKeyExchangeConnection(keyExchangeListener)

	// Open up c2 commincation listener TCP socket
	listener, err := tNet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(tunIP), Port: int(netstackPort)})
	if err != nil {
		wgLog.Errorf("Failed to setup up wg sliver listener: %v", err)
		return nil, nil, nil, err
	}
	wgLog.Printf("Successfully setup up wg sliver listener")
	go acceptWGSliverConnections(listener)
	return listener, dev, wgConf, nil
}

// acceptKeyExchangeConnection - accept connections to key exchange socket
func acceptKeyExchangeConnection(ln net.Listener) {
	wgLog.Printf("Polling for connections to key exchange listener")
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				wgLog.Errorf("Accept failed: %v", err)
				break
			}
			wgLog.Errorf("Accept failed: %v", err)
			continue
		}
		wgLog.Infof("Accepted connection to wg key exchange listener: %s", conn.RemoteAddr())
		go handleKeyExchangeConnection(conn)
	}
}

// handleKeyExchangeConnection - Retrieve current wg server pub key.
// Generate new implant wg keys. Generate new unique IP for implant.
// Write all retrieved data to socket connection.
func handleKeyExchangeConnection(conn net.Conn) {
	wgLog.Infof("Handling connection to key exchange listener")

	defer conn.Close()
	ip, err := generate.GenerateUniqueIP()
	if err != nil {
		wgLog.Errorf("Failed to generate unique IP: %s", err)
	}

	implantPrivKey, _, err := certs.ImplantGenerateWGKeys(ip.String())
	if err != nil {
		wgLog.Errorf("Failed to generate new wg keys: %s", err)
	}

	_, serverPubKey, err := certs.GetWGServerKeys()
	if err != nil {
		wgLog.Errorf("Failed to retrieve existing wg server keys: %s", err)
	} else {
		wgLog.Infof("Successfully generated new wg keys")
		message := implantPrivKey + "|" + serverPubKey + "|" + string(ip)
		wgLog.Debugf("Sending new wg keys and IP: %s", message)
		conn.Write([]byte(message))
	}
}

func acceptWGSliverConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			wgLog.Errorf("Accept failed: %v", err)
			continue
		}
		go handleWGSliverConnection(conn)
	}
}

func handleWGSliverConnection(conn net.Conn) {
	wgLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())

	implantConn := core.NewImplantConnection("wg", conn.RemoteAddr().String())
	implantConn.UpdateLastMessage()
	defer func() {
		implantConn.Cleanup()
		conn.Close()
	}()

	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		handlers := serverHandlers.GetHandlers()
		for {
			envelope, err := socketWGReadEnvelope(conn)
			if err != nil {
				wgLog.Errorf("Socket read error %s", err)
				return
			}
			implantConn.UpdateLastMessage()
			if envelope.ID != 0 {
				implantConn.RespMutex.RLock()
				if resp, ok := implantConn.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				implantConn.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				go func() {
					respEnvelope := handler(implantConn, envelope.Data)
					if respEnvelope != nil {
						implantConn.Send <- respEnvelope
					}
				}()
			}
		}
	}()

Loop:
	for {
		select {
		case envelope := <-implantConn.Send:
			err := socketWGWriteEnvelope(conn, envelope)
			if err != nil {
				wgLog.Errorf("Socket write failed %s", err)
				break Loop
			}
		case <-done:
			break Loop
		}
	}
	wgLog.Debugf("Closing connection to implant %s", implantConn.ID)
}

// socketWGWriteEnvelope - Writes a message to the wireguard socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWGWriteEnvelope(connection net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		wgLog.Errorf("Envelope marshaling error: %v", err)
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	connection.Write(dataLengthBuf.Bytes())
	connection.Write(data)
	return nil
}

// socketWGReadEnvelope - Reads a message from the wireguard connection using length prefix framing
// returns messageType, message, and error
func socketWGReadEnvelope(connection net.Conn) (*sliverpb.Envelope, error) {

	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32

	n, err := io.ReadFull(connection, dataLengthBuf)

	if err != nil || n != 4 {
		wgLog.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	if dataLength <= 0 {
		// {{if .Config.Debug}}
		wgLog.Errorf("[wireguard] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[wireguard] zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(connection, dataBuf)

	if err != nil || n != dataLength {
		wgLog.Errorf("Socket error (read data): %v", err)
		return nil, err
	}
	// Unmarshal the protobuf envelope
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		wgLog.Errorf("Un-marshaling envelope error: %v", err)
		return nil, err
	}
	return envelope, nil
}
