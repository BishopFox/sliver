package transport

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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
	pb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

const (
	readBufSize = 1024
)

var (
	once = &sync.Once{}
)

// MTLSConnect - Connect to the sliver server
func MTLSConnect(config *assets.ClientConfig) (chan *pb.Envelope, chan *pb.Envelope, error) {
	conn, err := tlsConnect(config.LHost, uint16(config.LPort), config)
	if err != nil {
		return nil, nil, err
	}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)

	go func() {
		defer once.Do(func() {
			close(send)
			close(recv)
			conn.Close()
		})

		for envelope := range send {
			err := socketWriteEnvelope(conn, envelope)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer once.Do(func() {
			close(send)
			close(recv)
			conn.Close()
		})
		for {
			envelope, err := socketReadEnvelope(conn)
			if err == io.EOF {
				log.Printf("Lost connection to server")
				return
			}
			if envelope == nil {
				log.Printf("Warning: nil envelope")
				continue
			}
			if err == nil && envelope != nil {
				recv <- envelope
			}
		}
	}()

	return send, recv, nil
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func socketWriteEnvelope(connection *tls.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Print("Envelope marshaling error: ", err)
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
		log.Printf("Socket error (read msg-length): %v\n", err)
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
			log.Printf("Read error: %s\n", err)
			break
		}
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		log.Printf("Unmarshaling envelope error: %v", err)
		return &pb.Envelope{}, err
	}

	return envelope, nil
}

// tlsConnect - Get a TLS connection or die trying
func tlsConnect(address string, port uint16, config *assets.ClientConfig) (*tls.Conn, error) {
	tlsConfig, err := getTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}
	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		log.Printf("Unable to connect: %v", err)
		return nil, err
	}
	return connection, nil
}

func getTLSConfig(caCertificate string, certificate string, privateKey string) (*tls.Config, error) {

	certPEM, err := tls.X509KeyPair([]byte(certificate), []byte(privateKey))
	if err != nil {
		log.Printf("Cannot parse client certificate: %v", err)
		return nil, err
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertificate))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return rootOnlyVerifyCertificate(caCertificate, rawCerts)
		},
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig, nil
}

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(caCertificate string, rawCerts [][]byte) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertificate))
	if !ok {
		log.Printf("Failed to parse root certificate")
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		log.Printf("Failed to parse certificate: " + err.Error())
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if _, err := cert.Verify(options); err != nil {
		log.Printf("Failed to verify certificate: " + err.Error())
		return err
	}

	return nil
}
