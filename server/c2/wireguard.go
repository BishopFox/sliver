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
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/generate"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/netstack"
	"github.com/bishopfox/sliver/util/minisign"
	"github.com/hashicorp/yamux"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"google.golang.org/protobuf/proto"
)

var (
	wgLog = log.NamedLogger("c2", "wg")
	tunIP = "100.64.0.1" // Don't let user configure this for now
)

const (
	wgYamuxPreface = mtlsYamuxPreface
)

var (
	wgYamuxPrefaceBytes = []byte(wgYamuxPreface)
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

	tunIPAddr, err := netip.ParseAddr(tunIP)
	if err != nil {
		wgLog.Errorf("ParseAddr failed: %v", err)
		return nil, nil, nil, err
	}

	// Allow netstack to listen on the ports we need
	if err := tNet.AllowTCPPort(tunIPAddr, netstackPort); err != nil {
		wgLog.Errorf("AllowTCPPort failed for netstackPort: %v", err)
		return nil, nil, nil, err
	}

	if err := tNet.AllowTCPPort(tunIPAddr, keyExchangeListenPort); err != nil {
		wgLog.Errorf("AllowTCPPort failed for keyExchangeListenPort: %v", err)
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

	validPeerCount := 0
	for k, v := range peers {
		tunPeerIP := strings.TrimSpace(v)
		if tunPeerIP == "" {
			wgLog.Warnf("Skipping wireguard peer %q with empty tunnel IP in the database", k)
			continue
		}
		if _, err := netip.ParseAddr(tunPeerIP); err != nil {
			wgLog.Warnf("Skipping wireguard peer %q with invalid tunnel IP %q: %v", k, tunPeerIP, err)
			continue
		}
		validPeerCount++
		fmt.Fprintf(wgConf, "public_key=%s\n", k)
		fmt.Fprintf(wgConf, "allowed_ip=%s/32\n", tunPeerIP)
	}

	// Set wg device config
	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to apply wireguard interface config (%d valid peer entries): %w", validPeerCount, err)
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
	defer func() {
		wgLog.Debugf("wireguard connection closing")
		implantConn.Cleanup()
		conn.Close()
	}()

	br := bufio.NewReader(conn)
	bufferedConn := &wgBufferedConn{Conn: conn, r: br}

	preface, err := br.Peek(len(wgYamuxPrefaceBytes))
	if err == nil && bytes.Equal(preface, wgYamuxPrefaceBytes) {
		if _, err := br.Discard(len(wgYamuxPrefaceBytes)); err != nil {
			wgLog.Errorf("Failed to discard yamux preface: %v", err)
			return
		}
		handleWGSliverConnectionYamux(bufferedConn, implantConn)
		return
	}

	wgLog.Warnf("Rejecting legacy wireguard connection (missing yamux preface) from %s", conn.RemoteAddr())
}

type wgBufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *wgBufferedConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func handleWGSliverConnectionYamux(conn net.Conn, implantConn *core.ImplantConnection) {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		wgLog.Errorf("Failed to initialize yamux session: %v", err)
		return
	}
	defer session.Close()

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
			session.Close()
		})
	}

	streamSem := make(chan struct{}, mtlsYamuxMaxConcurrentStreams)
	sendSem := make(chan struct{}, mtlsYamuxMaxConcurrentSends)
	handlers := serverHandlers.GetHandlers()

	go func() {
		defer closeDone()
		for {
			stream, err := session.Accept()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					wgLog.Errorf("yamux accept error: %v", err)
				}
				return
			}

			select {
			case streamSem <- struct{}{}:
			case <-done:
				stream.Close()
				return
			}

			go func(stream net.Conn) {
				defer func() {
					<-streamSem
				}()
				defer stream.Close()

				envelope, err := socketWGReadEnvelope(stream)
				if err != nil {
					wgLog.Errorf("Stream read error %v", err)
					closeDone()
					return
				}
				implantConn.UpdateLastMessage()

				if envelope.ID != 0 {
					implantConn.RespMutex.RLock()
					resp, ok := implantConn.Resp[envelope.ID]
					implantConn.RespMutex.RUnlock()
					if ok {
						resp <- envelope
					}
					return
				}

				if handler, ok := handlers[envelope.Type]; ok {
					go func(envelope *sliverpb.Envelope) {
						respEnvelope := handler(implantConn, envelope.Data)
						if respEnvelope != nil {
							implantConn.Send <- respEnvelope
						}
					}(envelope)
				}
			}(stream)
		}
	}()

	go func() {
		defer closeDone()
		for {
			select {
			case envelope := <-implantConn.Send:
				select {
				case sendSem <- struct{}{}:
				case <-done:
					return
				}

				go func(envelope *sliverpb.Envelope) {
					defer func() {
						<-sendSem
					}()

					stream, err := session.Open()
					if err != nil {
						wgLog.Errorf("yamux open stream error: %v", err)
						closeDone()
						return
					}
					defer stream.Close()

					if err := socketWGWriteEnvelope(stream, envelope); err != nil {
						wgLog.Errorf("Stream write failed %v", err)
						closeDone()
						return
					}
				}(envelope)

			case <-done:
				return
			}
		}
	}()

	<-done
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

	// Prepend a fixed-length raw minisign signature (binary) so the implant can
	// verify messages independent of the WireGuard layer.
	rawSig := minisign.SignRawBuf(*serverCrypto.MinisignServerPrivateKey(), data)
	if _, err := connection.Write(rawSig[:]); err != nil {
		return err
	}

	dataLengthBuf := new(bytes.Buffer)
	if err := binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data))); err != nil {
		wgLog.Errorf("Envelope marshaling error: %v", err)
		return err
	}
	if _, err := connection.Write(dataLengthBuf.Bytes()); err != nil {
		return err
	}
	if _, err := connection.Write(data); err != nil {
		return err
	}
	return nil
}

// socketWGReadEnvelope - Reads a message from the wireguard connection using length prefix framing
// returns messageType, message, and error
func socketWGReadEnvelope(connection net.Conn) (*sliverpb.Envelope, error) {
	rawSigBuf := make([]byte, minisign.RawSigSize)

	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32

	n, err := io.ReadFull(connection, rawSigBuf)
	if err != nil || n != len(rawSigBuf) {
		wgLog.Errorf("Socket error (read raw signature): %v", err)
		return nil, err
	}

	n, err = io.ReadFull(connection, dataLengthBuf)

	if err != nil || n != 4 {
		wgLog.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	if dataLength <= 0 || ServerMaxMessageSize < dataLength {
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

	algorithm := binary.LittleEndian.Uint16(rawSigBuf[:2])
	if algorithm != minisign.EdDSA {
		return nil, errors.New("[wireguard] unsupported signature algorithm")
	}
	keyID := binary.LittleEndian.Uint64(rawSigBuf[2:10])

	pubKey, _, err := lookupImplantSigKey(keyID)
	if err != nil {
		return nil, err
	}
	signature := rawSigBuf[10:]
	if !ed25519.Verify(pubKey, dataBuf, signature) {
		return nil, errors.New("[wireguard] invalid signature")
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
