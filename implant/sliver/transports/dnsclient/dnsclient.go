package dnsclient

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

// {{if .Config.DNSc2Enabled}}

import (

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"errors"
	"strings"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	ErrTimeout = errors.New("{{if .Config.Debug}}DNS Timeout{{end}}")
)

// DNSConnect - Attempt to establish a connection to the DNS server of 'parent'
func DNSConnect(parent string, pollTimeout time.Duration) (*SliverDNSClient, error) {
	// {{if .Config.Debug}}
	log.Printf("DNS client connecting to '%s' ...", parent)
	// {{end}}
	client := &SliverDNSClient{
		parent:        strings.TrimPrefix(parent, "."),
		caseSensitive: false,
		pollTimeout:   pollTimeout,
	}
	return client, nil
}

// SliverDNSClient - The DNS client context
type SliverDNSClient struct {
	parent        string
	pollTimeout   time.Duration
	caseSensitive bool
}

// SessionInit - Initialize DNS session
func (s *SliverDNSClient) SessionInit() error {
	return nil
}

// WriteEnvelope - Send an envelope to the server
func (s *SliverDNSClient) WriteEnvelope(envelope *pb.Envelope) error {
	return nil
}

// ReadEnvelope - Recv an envelope from the server
func (s *SliverDNSClient) ReadEnvelope() (*pb.Envelope, error) {
	return nil, nil
}

// fingerprintResolver - Fingerprints resolve to determine if we can use a case sensitive encoding
func (s *SliverDNSClient) fingerprintResolver() {

}

// {{end}} -DNSc2Enabled
