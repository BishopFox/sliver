package c2

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	consts "sliver/client/constants"
	pb "sliver/protobuf/sliver"
	"sliver/server/assets"
	"sliver/server/core"
	"sync"

	"sliver/server/certs"
	serverHandlers "sliver/server/handlers"

	"github.com/golang/protobuf/proto"
)

const (
	// defaultServerCert - Default certificate name if bind is "" (all interfaces)
	defaultServerCert = "hive"
)

// StartMutualTLSListener - Start a mutual TLS listener
func StartMutualTLSListener(bindIface string, port uint16) (net.Listener, error) {
	log.Printf("Starting Raw TCP/mTLS listener on %s:%d", bindIface, port)
	hostCert := bindIface
	if hostCert == "" {
		hostCert = defaultServerCert
	}
	tlsConfig := getServerTLSConfig(certs.SliversCertDir, hostCert)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	go acceptSliverConnections(ln)
	return ln, nil
}

func acceptSliverConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			log.Printf("Accept failed: %v", err)
			continue
		}
		go handleSliverConnection(conn)
	}
}

func handleSliverConnection(conn net.Conn) {
	log.Printf("Accepted incoming connection: %s", conn.RemoteAddr())

	sliver := &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     "mtls",
		RemoteAddress: fmt.Sprintf("%s", conn.RemoteAddr()),
		Send:          make(chan *pb.Envelope),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *pb.Envelope{},
	}

	core.Hive.AddSliver(sliver)

	defer func() {
		log.Printf("Cleaning up for %s", sliver.Name)
		core.Hive.RemoveSliver(sliver)
		conn.Close()
		core.EventBroker.Publish(core.Event{
			EventType: consts.DisconnectedEvent,
			Sliver:    sliver,
		})
	}()

	go func() {
		handlers := serverHandlers.GetSliverHandlers()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				log.Printf("Socket read error %v", err)
				return
			}
			if envelope.ID != 0 {
				sliver.RespMutex.RLock()
				if resp, ok := sliver.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				sliver.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				go handler.(func(*core.Sliver, []byte))(sliver, envelope.Data)
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
func socketWriteEnvelope(connection net.Conn, envelope *pb.Envelope) error {
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
// returns messageType, message, and error
func socketReadEnvelope(connection net.Conn) (*pb.Envelope, error) {

	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		log.Printf("Socket error (read msg-length): %v", err)
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
			log.Printf("Read error: %s", err)
			break
		}
	}

	if err != nil {
		log.Printf("Socket error (read data): %v", err)
		return nil, err
	}
	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		log.Printf("unmarshaling envelope error: %v", err)
		return nil, err
	}
	return envelope, nil
}

// getServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getServerTLSConfig(caType string, host string) *tls.Config {

	rootDir := assets.GetRootAppDir()

	caCertPtr, _, err := certs.GetCertificateAuthority(rootDir, caType)
	if err != nil {
		log.Fatalf("Invalid ca type (%s): %v", caType, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	certPEM, keyPEM, _ := certs.GetServerCertificatePEM(rootDir, caType, host, true)
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
