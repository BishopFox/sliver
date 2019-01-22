package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

func startMutualTLSListener(bindIface string, port uint16, events chan Event) (net.Listener, error) {
	log.Printf("Starting Raw TCP/TLS listener on %s:%d", bindIface, port)

	tlsConfig := getServerTLSConfig(sliversCertDir, bindIface)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	go acceptConnections(ln, events)
	return ln, nil
}

func acceptConnections(ln net.Listener, events chan Event) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			log.Printf("Accept failed: %v", err)
			continue
		}
		go handleSliverConnection(conn, events)
	}
}

func handleSliverConnection(conn net.Conn, events chan Event) {
	log.Printf("Accepted incoming connection: %s", conn.RemoteAddr())

	envelope, err := socketReadEnvelope(conn)
	if err != nil {
		log.Printf("Socket read error: %v", err)
		return
	}
	registerSliver := &pb.RegisterSliver{}
	proto.Unmarshal(envelope.Data, registerSliver)
	send := make(chan pb.Envelope)

	sliver := &Sliver{
		Id:            getHiveID(),
		Name:          registerSliver.Name,
		Hostname:      registerSliver.Hostname,
		Username:      registerSliver.Username,
		Uid:           registerSliver.Uid,
		Gid:           registerSliver.Gid,
		Os:            registerSliver.Os,
		Arch:          registerSliver.Arch,
		Pid:           registerSliver.Pid,
		RemoteAddress: fmt.Sprintf("%s", conn.RemoteAddr()),
		Send:          send,
		Resp:          map[string]chan pb.Envelope{},
	}

	hiveMutex.Lock()
	(*hive)[sliver.Id] = sliver
	hiveMutex.Unlock()

	defer func() {
		log.Printf("Cleaning up for %s", sliver.Name)
		hiveMutex.Lock()
		delete(*hive, sliver.Id)
		hiveMutex.Unlock()
		conn.Close()
		events <- Event{Sliver: sliver, EventType: "disconnected"}
	}()

	events <- Event{Sliver: sliver, EventType: "connected"}

	go func() {
		defer func() {
			for _, resp := range sliver.Resp {
				close(resp)
			}
			close(sliver.Send)
		}()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				log.Printf("Socket read error %v", err)
				return
			}
			if envelope.Id != "" {
				if resp, ok := sliver.Resp[envelope.Id]; ok {
					resp <- envelope
				}
			}
		}
	}()

	for envelope := range sliver.Send {
		err := socketWriteEnvelope(conn, envelope)
		if err != nil {
			log.Printf("Socket write failed %v", err)
			return
		}
	}
	log.Printf("Closing connection to sliver %s", sliver.Name)
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func socketWriteEnvelope(connection net.Conn, envelope pb.Envelope) error {
	data, err := proto.Marshal(&envelope)
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
// returns messageType, message, and error
func socketReadEnvelope(connection net.Conn) (pb.Envelope, error) {

	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		log.Printf("Socket error (read msg-length): %v", err)
		return pb.Envelope{}, err
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
			log.Printf("Read error: %s", err)
			break
		}
	}

	if err != nil {
		log.Printf("Socket error (read data): %v", err)
		return pb.Envelope{}, err
	}
	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		log.Printf("unmarshaling envelope error: %v", err)
		return pb.Envelope{}, err
	}
	return *envelope, nil
}

// getServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getServerTLSConfig(caType string, host string) *tls.Config {
	caCertPtr, _, err := GetCertificateAuthority(caType)
	if err != nil {
		log.Fatalf("Invalid ca type (%s): %v", caType, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	certPEM, keyPEM, _ := GetServerCertificatePEM(caType, host)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:                  caCertPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caCertPool,
		Certificates:             []tls.Certificate{cert},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
