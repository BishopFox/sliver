package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	pb "sliver/protobuf"
)

const (
	// randomIDSize - Size of the TunnelID in bytes
	randomIDSize = 8

	logFileName = "sliver.log"
	timeout     = 30 * time.Second
	readBufSize = 1024
)

// Sliver implant
type Sliver struct {
	ID            string
	Name          string
	Hostname      string
	Username      string
	Uid           string
	Gid           string
	Os            string
	Arch          string
	RemoteAddress string
	Send          chan pb.Envelope
	Recv          chan pb.Envelope
}

// ConsoleMsg -
type ConsoleMsg struct {
	Level   string
	Message string
}

var (
	server      *string
	serverLPort *int

	// Yea I'm lazy, it'd be better not to use mutex
	hiveMutex = &sync.RWMutex{}
	hive      = &map[string]*Sliver{}
)

func main() {
	server = flag.String("server", "", "bind server address")
	serverLPort = flag.Int("server-lport", 8888, "bind listen port")
	flag.Parse()

	appDir := GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()
	if _, err := os.Stat(path.Join(appDir, goDirName)); os.IsNotExist(err) {
		log.Println("First time setup, unpacking assets please wait ... ")
		SetupAssets()
	}

	events := make(chan *Sliver)

	log.Println("Starting listeners ...")
	// fmt.Printf(Info+"Binding to %s:%d\n", *server, *serverLPort)
	ln, err := startSliverListener(*server, uint16(*serverLPort), events)
	if err != nil {
		log.Printf("Failed to start server")
		fmt.Printf("\rFailed to start server %v", err)
		return
	}

	defer func() {
		ln.Close()
		hiveMutex.Lock()
		defer hiveMutex.Unlock()
		for _, sliver := range *hive {
			close(sliver.Send)
		}
	}()

	startConsole(events)
}

// Initialize logging
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}

func startSliverListener(bindIface string, port uint16, events chan *Sliver) (net.Listener, error) {
	log.Printf("Starting listener on %s:%d", bindIface, port)

	tlsConfig := getServerTLSConfig(SliversDir, bindIface)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	go acceptConnections(ln, events)
	return ln, nil
}

func acceptConnections(ln net.Listener, events chan *Sliver) {
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

func handleSliverConnection(conn net.Conn, events chan *Sliver) {
	log.Printf("Accepted incoming connection: %s", conn.RemoteAddr())

	envelope, err := socketReadEnvelope(conn)
	if err != nil {
		log.Printf("Socket read error: %v", err)
		return
	}
	registerSliver := &pb.RegisterSliver{}
	proto.Unmarshal(envelope.Data, registerSliver)
	send := make(chan pb.Envelope)
	recv := make(chan pb.Envelope)

	sliver := &Sliver{
		ID:            randomID(),
		Name:          registerSliver.Name,
		Hostname:      registerSliver.Hostname,
		Username:      registerSliver.Username,
		Uid:           registerSliver.Uid,
		Gid:           registerSliver.Gid,
		Os:            registerSliver.Os,
		Arch:          registerSliver.Arch,
		RemoteAddress: fmt.Sprintf("%s", conn.RemoteAddr()),
		Send:          send,
		Recv:          recv,
	}

	hiveMutex.Lock()
	(*hive)[sliver.ID] = sliver
	hiveMutex.Unlock()

	defer func() {
		hiveMutex.Lock()
		delete(*hive, sliver.ID)
		hiveMutex.Unlock()
		conn.Close()
	}()

	select {
	case events <- sliver: // Non-blocking channel send
	default:
	}

	go func() {
		defer close(sliver.Recv)
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				log.Printf("Socket read error %v", err)
				break
			}
			sliver.Recv <- envelope
		}
	}()

	for envelope := range sliver.Send {
		err := socketWriteEnvelope(conn, envelope)
		if err != nil {
			return
		}
	}

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

// GetRootAppDir - Get the Sliver app dir ~/.sliver/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, ".sliver")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// log.Printf("Creating app directory: %s", dir)
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// randomID - Generate random ID of randomIDSize bytes
func randomID() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:randomIDSize])
}
