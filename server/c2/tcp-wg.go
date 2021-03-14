package c2

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/netstack"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/golang/protobuf/proto"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
)

var (
	wgLog = log.NamedLogger("c2", "wg")
	tunIP = "100.64.0.1" // Don't let user configure this for now
)

func StartWGListener(port uint16, netstackPort uint16) (net.Listener, *device.Device, *bytes.Buffer, error) {
	StartPivotListener()
	wgLog.Infof("Starting Wireguard listener on port: %d", port)

	tun, tnet, err := netstack.CreateNetTUN(
		[]net.IP{net.ParseIP(tunIP)},
		[]net.IP{net.ParseIP("127.0.0.1")}, // We don't use DNS in the WG listener
		1420,
	)
	if err != nil {
		wgLog.Panic(err)
	}

	// Get existing keys
	privateKey, _, err := certs.GetWGServerKeys()

	if err != nil {
		isPeer := false
		privateKey, _, err = certs.GenerateWGKeys(isPeer, "")
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Set this to device.LogLevelVerbose when debugging
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

	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		return nil, nil, nil, err
	}

	dev.Up()

	// Open up key exchange TCP socket
	keyExchangeListener, err := tnet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(tunIP), Port: 1337})
	if err != nil {
		wgLog.Panic("Failed to setup up wg key exchange listener: %s", err)
	}
	wgLog.Printf("Successfully setup up wg key exchange listener")
	go acceptKeyExchangeConnection(keyExchangeListener)

	// Open up listener TCP socket
	listener, err := tnet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(tunIP), Port: int(netstackPort)})
	if err != nil {
		wgLog.Panic("Failed to setup up wg sliver listener: %s", err)
	}
	wgLog.Printf("Successfully setup up wg sliver listener")
	go acceptWGSliverConnections(listener)
	return listener, dev, wgConf, nil
}

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

func handleKeyExchangeConnection(conn net.Conn) {
	wgLog.Infof("Handling connection to key exchange listener")

	defer conn.Close()
	ip, err := command.GenUniqueIP()
	if err != nil {
		wgLog.Errorf("Failed to generate unique IP: %s", err)
	}

	implantPrivKey, _, err := certs.ImplantGenerateWGKeys(ip.String())
	_, serverPubKey, err := certs.GetWGServerKeys()

	if err != nil {
		wgLog.Errorf("Failed to generate new wg keys: %s", err)
	} else {
		wgLog.Infof("Successfully generated new wg keys")
		message := implantPrivKey + "|" + serverPubKey + "|" + string(ip)

		wgLog.Infof("Sending new wg keys and IP: %s", message)
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

	session := &core.Session{
		Transport:     "wg",
		RemoteAddress: fmt.Sprintf("%s", conn.RemoteAddr()),
		Send:          make(chan *sliverpb.Envelope),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
	}
	session.UpdateCheckin()

	defer func() {
		wgLog.Debugf("Cleaning up for %s", session.Name)
		core.Sessions.Remove(session.ID)
		conn.Close()
	}()

	done := make(chan bool)

	go func() {
		defer func() {
			done <- true
		}()
		handlers := serverHandlers.GetSessionHandlers()
		for {
			envelope, err := socketWGReadEnvelope(conn)
			if err != nil {
				wgLog.Errorf("Socket read error %v", err)
				return
			}
			session.UpdateCheckin()
			if envelope.ID != 0 {
				session.RespMutex.RLock()
				if resp, ok := session.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				session.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				go handler.(func(*core.Session, []byte))(session, envelope.Data)
			}
		}
	}()

Loop:
	for {
		select {
		case envelope := <-session.Send:
			err := socketWGWriteEnvelope(conn, envelope)
			if err != nil {
				wgLog.Errorf("Socket write failed %v", err)
				break Loop
			}
		case <-done:
			break Loop
		}
	}
	wgLog.Infof("Closing connection to session %s", session.Name)
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
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		wgLog.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	// Read the length of the data, keep in mind each call to .Read() may not
	// fill the entire buffer length that we specify, so instead we use two buffers
	// readBuf is the result of each .Read() operation, which is then concatinated
	// onto dataBuf which contains all of data read so far and we keep calling
	// .Read() until the running total is equal to the length of the message that
	// we're expecting or we get an error.
	readBuf := make([]byte, readBufSize)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := connection.Read(readBuf)
		dataBuf = append(dataBuf, readBuf[:n]...)
		totalRead += n
		if totalRead == dataLength {
			break
		}
		if err != nil {
			wgLog.Errorf("Read error: %s", err)
			break
		}
	}

	if err != nil {
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
