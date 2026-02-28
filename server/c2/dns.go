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
	protobuf and contain a 24-bit "dns session ID" (16777216 possible values), and a 8 bit
	"message ID." 16,777,216 can probably be bruteforced but it'll at least be slow.

	DNS command and control outline:
		1. Implant generates a random DNS Session ID and sends an INIT (Age key exchange)
		2. DNS server validates INIT and allocates session state
		3. Requests with valid DNS session IDs enable the server to respond with CRC32 responses
		4. Implant establishes encrypted session

*/

import (
	"bytes"
	secureRand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	sliverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
)

const (
	// Little endian
	sessionIDBitMask = 0x00ffffff // Bitwise mask to get the dns session ID
	messageIDBitMask = 0xff000000 // Bitwise mask to get the message ID

	defaultMaxTXTLength = 254

	// Upper bound on unauthenticated init reassembly state. INIT is split across multiple
	// DNS queries, so we cap the number of pending init messages to avoid unbounded memory use.
	defaultMaxPendingDNSInits = 1024

	// Pending INIT messages are expected to complete quickly. If they don't, expire them.
	defaultPendingDNSInitTTL        = 2 * time.Minute
	defaultPendingDNSInitGCInterval = 30 * time.Second

	// INIT carries only key-exchange material and should remain small.
	defaultMaxDNSInitSize = 16 * 1024
)

var (
	dnsLog                = log.NamedLogger("c2", "dns")
	implantBase64         = encoders.Base64{} // Implant's version of base64 with custom alphabet
	ErrInvalidMsg         = errors.New("invalid dns message")
	ErrNoOutgoingMessages = errors.New("no outgoing messages")
	ErrMsgTooLong         = errors.New("too much data to encode")
	ErrMsgTooShort        = errors.New("too little data to encode")
)

// StartDNSListener - Start a DNS listener
func StartDNSListener(bindIface string, lport uint16, domains []string, canaries bool) *SliverDNSServer {
	// StartPivotListener()
	server := &SliverDNSServer{
		server:       &dns.Server{Addr: fmt.Sprintf("%s:%d", bindIface, lport), Net: "udp"},
		sessions:     &sync.Map{}, // DNS Session ID -> DNSSession
		messages:     &sync.Map{}, // In progress message streams
		TTL:          0,
		MaxTXTLength: defaultMaxTXTLength,
	}
	server.startInitGC()
	dnsLog.Infof("Starting DNS listener for %v (canaries: %v) ...", domains, canaries)
	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		defer recoverAndLogPanic(dnsLog.Errorf, "dns request callback")

		started := time.Now()
		server.HandleDNSRequest(domains, canaries, writer, req)
		dnsLog.Debugf("DNS server took %s", time.Since(started))
	})
	return server
}

// DNSSession - Holds DNS session information
type DNSSession struct {
	ID          uint32
	ImplantConn *core.ImplantConnection
	CipherCtx   *cryptography.CipherContext

	Created  time.Time
	lastSeen atomic.Int64 // unix nanos

	dnsIdMsgIdMap   map[uint32]uint32
	outgoingMsgIDs  []uint32
	outgoingBuffers map[uint32][]byte
	outgoingMutex   *sync.RWMutex

	incomingEnvelopes map[uint32]*PendingEnvelope
	incomingMutex     *sync.Mutex
	msgCount          uint32
}

func (s *DNSSession) touch(now time.Time) {
	s.lastSeen.Store(now.UnixNano())
}

func (s *DNSSession) msgID(id uint32) uint32 {
	return uint32(id<<24) | uint32(s.ID)
}

func (s *DNSSession) nextMsgID() uint32 {
	s.msgCount++
	return s.msgID(s.msgCount % 255)
}

// StageOutgoingEnvelope - Stage an outgoing envelope
func (s *DNSSession) StageOutgoingEnvelope(envelope *sliverpb.Envelope) error {
	dnsLog.Debugf("Staging outgoing envelope %v", envelope)
	plaintext, err := proto.Marshal(envelope)
	if err != nil {
		dnsLog.Errorf("[dns] failed to marshal outgoing message %s", err)
		return err
	}
	ciphertext, err := s.CipherCtx.Encrypt(plaintext)
	if err != nil {
		dnsLog.Errorf("[dns] failed to encrypt outgoing message %s", err)
		return err
	}

	s.outgoingMutex.Lock()
	defer s.outgoingMutex.Unlock()
	msgID := s.nextMsgID()
	s.outgoingMsgIDs = append(s.outgoingMsgIDs, msgID)
	s.outgoingBuffers[msgID] = ciphertext
	dnsLog.Debugf("Staged outgoing envelope successfully (%d bytes)", len(ciphertext))
	return nil
}

// PopOutgoingMsgID - Pop the next outgoing message ID, FIFO
// returns msgID, len, err
func (s *DNSSession) PopOutgoingMsgID(msg *dnspb.DNSMessage) (uint32, uint32, error) {
	s.outgoingMutex.Lock()
	defer s.outgoingMutex.Unlock()
	if len(s.outgoingMsgIDs) == 0 {
		return 0, 0, ErrNoOutgoingMessages
	}
	msgID := s.outgoingMsgIDs[0]
	s.outgoingMsgIDs = s.outgoingMsgIDs[1:]
	ciphertext, ok := s.outgoingBuffers[msgID]
	if !ok {
		return 0, 0, errors.New("no buffer for msg id")
	}
	//Necessary for any race conditions for resolvers that send out multiple identical requests
	s.dnsIdMsgIdMap[msg.ID] = msgID
	return msgID, uint32(len(ciphertext)), nil
}

// OutgoingRead - Read request from implant
func (s *DNSSession) OutgoingRead(msgID uint32, start uint32, stop uint32) ([]byte, error) {
	s.outgoingMutex.RLock()
	defer s.outgoingMutex.RUnlock()

	outgoingBuf, ok := s.outgoingBuffers[msgID]
	if !ok {
		return nil, ErrInvalidMsg
	}
	if uint32(len(outgoingBuf)) < start || uint32(len(outgoingBuf)) < stop || stop <= start {
		return nil, errors.New("invalid read boundaries")
	}
	readBuf := make([]byte, stop-start)
	copy(readBuf, outgoingBuf[start:stop])
	return readBuf, nil
}

// ClearOutgoingEnvelope - Clear an outgoing envelope this will generally, but not always,
// be the first value in the list
func (s *DNSSession) ClearOutgoingEnvelope(msgID uint32) {
	s.outgoingMutex.Lock()
	defer s.outgoingMutex.Unlock()
	delete(s.outgoingBuffers, msgID)
}

// IncomingPendingEnvelope - Get a pending message linked list, creates one if it doesn't exist
func (s *DNSSession) IncomingPendingEnvelope(msgID uint32, size uint32) *PendingEnvelope {
	s.incomingMutex.Lock()
	defer s.incomingMutex.Unlock()
	if pendingMsg, ok := s.incomingEnvelopes[msgID]; ok {
		return pendingMsg
	}
	pendingMsg := &PendingEnvelope{
		Size:     size,
		received: uint32(0),
		messages: map[uint32][]byte{},
		mutex:    &sync.Mutex{},
		complete: false,
	}
	s.incomingEnvelopes[msgID] = pendingMsg
	return pendingMsg
}

// ForwardCompletedEnvelope - Reassembles and forwards envelopes to core
func (s *DNSSession) ForwardCompletedEnvelope(msgID uint32, pending *PendingEnvelope) {
	defer recoverAndLogPanic(dnsLog.Errorf, "dns ForwardCompletedEnvelope")

	dnsLog.Debugf("[dns] dns session id: %d, msg id: %d completed message", s.ID, msgID)
	s.incomingMutex.Lock()
	delete(s.incomingEnvelopes, msgID) // Remove pending message
	s.incomingMutex.Unlock()
	data, err := pending.Reassemble()
	if err != nil {
		dnsLog.Errorf("Failed to reassemble message %d: %s", msgID, err)
		return
	}
	// dnsLog.Debugf("[dns] decrypt: %v", data)
	plaintext, err := s.CipherCtx.Decrypt(data)
	if err != nil {
		dnsLog.Errorf("Failed to decrypt message %d: %s", msgID, err)
		return
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(plaintext, envelope)
	if err != nil {
		dnsLog.Errorf("Failed to unmarshal message %d: %s", msgID, err)
		return
	}

	s.ImplantConn.UpdateLastMessage()
	handlers := sliverHandlers.GetHandlers()
	if envelope.ID != 0 {
		s.ImplantConn.RespMutex.RLock()
		defer s.ImplantConn.RespMutex.RUnlock()
		if resp, ok := s.ImplantConn.Resp[envelope.ID]; ok {
			resp <- envelope
		}
	} else if handler, ok := handlers[envelope.Type]; ok {
		respEnvelope := handler(s.ImplantConn, envelope.Data)
		if respEnvelope != nil {
			s.ImplantConn.Send <- respEnvelope
		}
	}
}

// PendingEnvelope - Holds data related to a pending incoming message
type PendingEnvelope struct {
	Size     uint32
	received uint32
	messages map[uint32][]byte
	mutex    *sync.Mutex
	complete bool
}

// pendingInit tracks the reassembly state for an INIT message (which may arrive
// split across multiple DNS queries).
type pendingInit struct {
	Created  time.Time
	Envelope *PendingEnvelope
}

// Reassemble - Reassemble a completed message
func (p *PendingEnvelope) Reassemble() ([]byte, error) {
	// There's some weird race here with a nil pointer
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.complete {
		return nil, fmt.Errorf("pending message not complete %d of %d", p.received, p.Size)
	}
	buffer := []byte{}
	keys := []uint32{}
	for k := range p.messages {
		keys = append(keys, k)
	}
	if 1 < len(keys) {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	}
	// dnsLog.Debugf("[dns] reassemble from: %v", keys)
	for _, k := range keys {
		// dnsLog.Debugf("[dns] reassemble %d (%d->%d): %d bytes",
		// 	index, len(buffer), len(buffer)+len(p.messages[k]), len(p.messages[k]))
		buffer = append(buffer, p.messages[k]...)
	}
	if len(buffer) != int(p.Size) {
		return nil, fmt.Errorf("invalid data size %d expected %d", len(buffer), p.Size)
	}
	return buffer, nil
}

// Insert - Pending message, returns true if message is complete
func (p *PendingEnvelope) Insert(dnsMsg *dnspb.DNSMessage) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.complete {
		return false // Already complete
	}
	p.messages[dnsMsg.Start] = bytes.NewBuffer(dnsMsg.Data).Bytes()
	// dnsLog.Debugf("[dns] msg id: %d, %d->%d, recv: %d of %d",
	// 	dnsMsg.ID, dnsMsg.Start, int(dnsMsg.Start)+len(dnsMsg.Data), p.received, p.Size)

	total := uint32(0)
	for k := range p.messages {
		total += uint32(len(p.messages[k]))
	}
	p.received = total
	p.complete = p.received >= p.Size
	// if p.complete {
	// 	dnsLog.Debugf("[dns] message complete %d of %d", p.received, p.Size)
	// }
	return p.complete
}

// --------------------------- DNS SERVER ---------------------------

// SliverDNSServer - DNS server implementation
type SliverDNSServer struct {
	server       *dns.Server
	sessions     *sync.Map
	messages     *sync.Map
	TTL          uint32
	MaxTXTLength int

	pendingInits atomic.Int64
	gcStopOnce   sync.Once
	gcStop       chan struct{}
	gcDone       chan struct{}
}

// Shutdown - Shutdown the DNS server
func (s *SliverDNSServer) Shutdown() error {
	s.gcStopOnce.Do(func() {
		if s.gcStop != nil {
			close(s.gcStop)
			if s.gcDone != nil {
				<-s.gcDone
			}
		}
	})
	return s.server.Shutdown()
}

// ListenAndServe - Listen for DNS requests and respond
func (s *SliverDNSServer) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *SliverDNSServer) startInitGC() {
	// Tests construct SliverDNSServer directly; only start GC for real listeners.
	if s.gcStop != nil {
		return
	}
	s.gcStop = make(chan struct{})
	s.gcDone = make(chan struct{})
	go func() {
		defer recoverAndLogPanic(dnsLog.Errorf, "dns init gc")

		ticker := time.NewTicker(defaultPendingDNSInitGCInterval)
		defer ticker.Stop()
		defer close(s.gcDone)

		for {
			select {
			case <-ticker.C:
				s.gcPendingInits()
			case <-s.gcStop:
				return
			}
		}
	}()
}

func (s *SliverDNSServer) gcPendingInits() {
	now := time.Now()
	var deleted int64
	s.messages.Range(func(key, value any) bool {
		p, ok := value.(*pendingInit)
		if !ok || p == nil {
			if _, loaded := s.messages.LoadAndDelete(key); loaded {
				deleted++
			}
			return true
		}
		created := p.Created
		if created.IsZero() {
			created = now
		}
		if now.Sub(created) > defaultPendingDNSInitTTL {
			if _, loaded := s.messages.LoadAndDelete(key); loaded {
				deleted++
			}
		}
		return true
	})
	if deleted > 0 {
		s.pendingInits.Add(-deleted)
	}
}

// ---------------------------
// DNS Handler
// ---------------------------
// Handles all DNS queries, first we determine if the query is C2 or a canary
func (s *SliverDNSServer) HandleDNSRequest(domains []string, canaries bool, writer dns.ResponseWriter, req *dns.Msg) {
	defer recoverAndLogPanic(dnsLog.Errorf, "dns HandleDNSRequest")

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
		// These responses often have near-max-length QNAMEs. Enable compression to
		// keep UDP responses under the minimum DNS message size limits.
		resp.Compress = true
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
	dnsLog.Debugf("[dns] processing req for subdomain = %s", subdomain)
	msg, checksum, err := s.decodeSubdata(subdomain)
	if err != nil {
		dnsLog.Errorf("[dns] error decoding subdata: %v", err)
		return s.nameErrorResp(req)
	}

	// INIT can be called without an existing server-side session. The server only
	// stores session state after validating the key exchange.
	if msg.Type == dnspb.DNSMessageType_INIT {
		return s.handleDNSSessionInit(domain, msg, checksum, req)
	}

	// All other handlers require a valid dns session ID
	sessionValue, ok := s.sessions.Load(msg.ID & sessionIDBitMask)
	if !ok {
		dnsLog.Warnf("[dns] session not found for id %v (%v)", msg.ID, msg.ID&sessionIDBitMask)
		return s.nameErrorResp(req)
	}
	if session, ok := sessionValue.(*DNSSession); ok && session != nil {
		session.touch(time.Now())
	}

	// Msg Type -> Handler
	switch msg.Type {
	case dnspb.DNSMessageType_NOP:
		return s.handleNOP(domain, msg, checksum, req)
	case dnspb.DNSMessageType_POLL:
		return s.handlePoll(domain, msg, checksum, req)
	case dnspb.DNSMessageType_DATA_FROM_IMPLANT:
		return s.handleDataFromImplant(domain, msg, checksum, req)
	case dnspb.DNSMessageType_DATA_TO_IMPLANT:
		return s.handleDataToImplant(domain, msg, checksum, req)
	case dnspb.DNSMessageType_CLEAR:
		return s.handleClear(domain, msg, checksum, req)
	}
	return nil
}

// Parse subdomain as data and calculate the CRC32 checksum, I decided to add the
// checksum calculation here to ensure that no one accidentally calculates the crc32
// of the plaintext data (that would be very bad).
func (s *SliverDNSServer) decodeSubdata(subdomain string) (*dnspb.DNSMessage, uint32, error) {
	subdata := strings.ToLower(strings.Join(strings.Split(subdomain, "."), ""))
	dnsLog.Debugf("subdata = %s", subdata)
	data, err := encoders.Base32{}.Decode([]byte(subdata))
	if err != nil {
		return nil, 0, ErrInvalidMsg
	}
	msg := &dnspb.DNSMessage{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, 0, ErrInvalidMsg
	}
	return msg, crc32.ChecksumIEEE(data), nil
}

func (s *SliverDNSServer) nameErrorResp(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetRcode(req, dns.RcodeNameError)
	resp.Authoritative = true
	return resp
}

func (s *SliverDNSServer) emptySuccessResp(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	return resp
}

func (s *SliverDNSServer) refusedErrorResp(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetRcode(req, dns.RcodeRefused)
	resp.Authoritative = true
	return resp
}

func (s *SliverDNSServer) accumulateInitData(msg *dnspb.DNSMessage) ([]byte, bool, error) {
	if msg.Size == 0 {
		return nil, false, ErrInvalidMsg
	}
	if msg.Size > defaultMaxDNSInitSize {
		return nil, false, ErrInvalidMsg
	}
	pendingValue, loaded := s.messages.Load(msg.ID)
	if !loaded {
		// Enforce a hard cap on pending INIT messages to prevent unbounded allocation.
		next := s.pendingInits.Add(1)
		if int64(defaultMaxPendingDNSInits) > 0 && next > int64(defaultMaxPendingDNSInits) {
			s.pendingInits.Add(-1)
			return nil, false, ErrInvalidMsg
		}

		pendingValue = &pendingInit{
			Created: time.Now(),
			Envelope: &PendingEnvelope{
				Size:     msg.Size,
				received: uint32(0),
				messages: map[uint32][]byte{},
				mutex:    &sync.Mutex{},
				complete: false,
			},
		}
		actual, loaded := s.messages.LoadOrStore(msg.ID, pendingValue)
		if loaded {
			// Another goroutine won the race. Roll back our pending counter.
			s.pendingInits.Add(-1)
			pendingValue = actual
		}
	}
	p, ok := pendingValue.(*pendingInit)
	if !ok || p == nil || p.Envelope == nil {
		s.deletePendingInit(msg.ID)
		return nil, false, ErrInvalidMsg
	}
	pending := p.Envelope
	if pending.Size != msg.Size && msg.Size != 0 {
		s.deletePendingInit(msg.ID)
		return nil, false, ErrInvalidMsg
	}
	if !pending.Insert(msg) {
		return nil, false, nil
	}

	if pending.received > pending.Size {
		s.deletePendingInit(msg.ID)
		return nil, false, ErrInvalidMsg
	}

	data, err := pending.Reassemble()
	s.deletePendingInit(msg.ID)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s *SliverDNSServer) deletePendingInit(id uint32) {
	if _, loaded := s.messages.LoadAndDelete(id); loaded {
		s.pendingInits.Add(-1)
	}
}

func (s *SliverDNSServer) handleDNSSessionInit(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	sessionID := msg.ID & sessionIDBitMask
	dnsLog.Debugf("[session init] with dns session id %d", sessionID)
	if sessionID == 0 {
		dnsLog.Warnf("[session init] invalid dns session id")
		return s.refusedErrorResp(req)
	}
	if msg.Size <= 32 {
		dnsLog.Warnf("[session init] invalid msg data length")
		return s.refusedErrorResp(req)
	}
	if existing, ok := s.sessions.Load(sessionID); ok {
		// Defensive: INIT should only be called once per session ID.
		if dnsSession, ok := existing.(*DNSSession); ok && dnsSession != nil && dnsSession.CipherCtx != nil {
			dnsLog.Warnf("[session init] session is already initialized")
			return s.refusedErrorResp(req)
		}
		dnsLog.Warnf("[session init] session already exists")
		return s.refusedErrorResp(req)
	}

	initData, complete, err := s.accumulateInitData(msg)
	if err != nil {
		dnsLog.Errorf("[session init] error reassembling init data: %s", err)
		return s.refusedErrorResp(req)
	}
	if !complete {
		return s.emptySuccessResp(req)
	}
	msg.Data = initData

	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], msg.Data[:32])
	implantBuild, err := db.ImplantBuildByPublicKeyDigest(publicKeyDigest)
	if err != nil || implantBuild == nil {
		dnsLog.Errorf("[session init] error implant public key not found")
		return s.refusedErrorResp(req)
	}

	serverKeyPair := cryptography.AgeServerKeyPair()
	sessionInit, err := cryptography.AgeKeyExFromImplant(serverKeyPair.Private, implantBuild.PeerPrivateKey, msg.Data[32:])
	if err != nil {
		dnsLog.Errorf("[session init] error decrypting session init data: %s", err)
		return s.refusedErrorResp(req)
	}
	sessionKey, err := cryptography.KeyFromBytes(sessionInit)
	if err != nil {
		dnsLog.Errorf("[session init] invalid session key: %s", err)
		return s.refusedErrorResp(req)
	}

	now := time.Now()
	dnsSession := &DNSSession{
		ID:                sessionID,
		Created:           now,
		dnsIdMsgIdMap:     map[uint32]uint32{},
		outgoingMsgIDs:    []uint32{},
		outgoingBuffers:   map[uint32][]byte{},
		outgoingMutex:     &sync.RWMutex{},
		incomingEnvelopes: map[uint32]*PendingEnvelope{},
		incomingMutex:     &sync.Mutex{},
		msgCount:          uint32(0),
	}
	dnsSession.touch(now)
	dnsSession.ImplantConn = core.NewImplantConnection("dns", "n/a")
	dnsSession.CipherCtx = cryptography.NewCipherContext(sessionKey)

	if _, loaded := s.sessions.LoadOrStore(sessionID, dnsSession); loaded {
		dnsLog.Warnf("[session init] session id collision for %d", sessionID)
		return s.refusedErrorResp(req)
	}

	go func() {
		defer recoverAndLogPanic(dnsLog.Errorf, "dns session send loop")

		dnsLog.Debugf("[dns] starting implant conn send loop")
		for envelope := range dnsSession.ImplantConn.Send {
			dnsSession.StageOutgoingEnvelope(envelope)
		}
		dnsLog.Debugf("[dns] closing implant conn send loop")
	}()
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

		case dns.TypeAAAA:

			chunks := splitToChunks(respData, 16)
			msg_len := len(respData)
			dnsLog.Infof("[dns] msg length: %d)", msg_len)
			for i, chunk := range chunks {
				ttl := uint32(msg_len)
				chunkIdx := uint32(i) << 8
				ttl = ttl ^ chunkIdx
				a_record := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: chunk,
				}
				resp.Answer = append(resp.Answer, a_record)
			}

		case dns.TypeTXT:
			rawTxt, _ := implantBase64.Encode(respData)
			respTxt := string(rawTxt)
			txts := []string{}
			for start, stop := 0, 0; stop < len(respTxt); start = stop {
				stop += s.MaxTXTLength
				if len(respTxt) < stop {
					stop = len(respTxt)
				}
				txts = append(txts, respTxt[start:stop])
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: s.TTL},
				Txt: txts,
			}
			resp.Answer = append(resp.Answer, txt)
		}
	}
	return resp
}

func (s *SliverDNSServer) handlePoll(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[poll] with dns session id %d", msg.ID&sessionIDBitMask)
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)
	dnsSession.touch(time.Now())

	msgID, msgLen, err := dnsSession.PopOutgoingMsgID(msg)
	if err != nil {
		if err != ErrNoOutgoingMessages {
			dnsLog.Errorf("[poll] error popping outgoing msg id: %s", err)
			return s.refusedErrorResp(req)
		} else {
			msgLen = 0
			msgID = 0
			dnsLog.Debugf("[poll] error: %s", err)
			oldID, ok := dnsSession.dnsIdMsgIdMap[msg.ID]
			if !ok {
				dnsLog.Debugf("[poll] no msg id for given request")
			} else {
				ciphertext, ok := dnsSession.outgoingBuffers[oldID]
				if !ok {
					dnsLog.Debugf("[poll] no msg for given id")
				} else {
					msgLen = uint32(len(ciphertext))
					msgID = oldID
				}
			}

		}
	}

	respData := []byte{}
	dnsLog.Debugf("[poll] manifest %d (%d bytes)", msgID, msgLen)
	respData, _ = proto.Marshal(&dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_MANIFEST,
		ID:   msgID,
		Size: msgLen,
	})

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeAAAA:

			chunks := splitToChunks(respData, 16)
			msg_len := len(respData)
			dnsLog.Infof("[dns] msg length: %d)", msg_len)
			for i, chunk := range chunks {
				ttl := uint32(msg_len)
				chunkIdx := uint32(i) << 8
				ttl = ttl ^ chunkIdx
				a_record := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: chunk,
				}
				resp.Answer = append(resp.Answer, a_record)
			}

		case dns.TypeTXT:
			rawTxt, _ := implantBase64.Encode(respData)
			respTxt := string(rawTxt)
			txts := []string{}
			for start, stop := 0, 0; stop < len(respTxt); start = stop {
				stop += s.MaxTXTLength
				if len(respTxt) < stop {
					stop = len(respTxt)
				}
				txts = append(txts, respTxt[start:stop])
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: s.TTL},
				Txt: txts,
			}
			resp.Answer = append(resp.Answer, txt)
		}
	}

	return resp
}

func (s *SliverDNSServer) handleDataFromImplant(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[from implant] dns session id %d", msg.ID&sessionIDBitMask)
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)
	dnsSession.touch(time.Now())
	dnsLog.Debugf("[from implant] msg id: %d, size: %d", msg.ID, msg.Size)
	pending := dnsSession.IncomingPendingEnvelope(msg.ID, msg.Size)
	complete := pending.Insert(msg)
	if complete {
		go func() {
			defer recoverAndLogPanic(dnsLog.Errorf, "dns completed envelope dispatch")
			dnsSession.ForwardCompletedEnvelope(msg.ID, pending)
		}()
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			resp.Authoritative = true
			// resp.RecursionAvailable = complete
			respBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(respBuf, checksum)
			a := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: s.TTL},
				A:   respBuf,
			}
			resp.Answer = append(resp.Answer, a)
		}
	}
	return resp
}

func (s *SliverDNSServer) handleDataToImplant(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[to implant] dns session id %d", msg.ID&sessionIDBitMask)
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)
	dnsSession.touch(time.Now())

	data, err := dnsSession.OutgoingRead(msg.ID, msg.Start, msg.Stop)
	if err != nil {
		dnsLog.Errorf("[to implant] read failed: %s", err)
		return s.refusedErrorResp(req)
	}

	respData, _ := proto.Marshal(&dnspb.DNSMessage{
		// Type:  dnspb.DNSMessageType_DATA_TO_IMPLANT,
		Start: msg.Start,
		Data:  data,
	})

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeAAAA:

			chunks := splitToChunks(respData, 16)
			msg_len := len(respData)
			dnsLog.Infof("[dns] msg length: %d)", msg_len)
			for i, chunk := range chunks {
				ttl := uint32(msg_len)
				chunkIdx := uint32(i) << 8
				ttl = ttl ^ chunkIdx
				a_record := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: chunk,
				}
				resp.Answer = append(resp.Answer, a_record)
			}

		case dns.TypeTXT:
			rawTxt, _ := implantBase64.Encode(respData)
			respTxt := string(rawTxt)
			txts := []string{}
			for start, stop := 0, 0; stop < len(respTxt); start = stop {
				stop += s.MaxTXTLength
				if len(respTxt) < stop {
					stop = len(respTxt)
				}
				txts = append(txts, respTxt[start:stop])
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: s.TTL},
				Txt: txts,
			}
			resp.Answer = append(resp.Answer, txt)
		}
	}
	return resp
}

func (s *SliverDNSServer) handleClear(domain string, msg *dnspb.DNSMessage, checksum uint32, req *dns.Msg) *dns.Msg {
	dnsLog.Debugf("[clear] dns session id %d", msg.ID&sessionIDBitMask)
	loadSession, _ := s.sessions.Load(msg.ID & sessionIDBitMask)
	dnsSession := loadSession.(*DNSSession)
	dnsSession.touch(time.Now())
	dnsSession.ClearOutgoingEnvelope(msg.ID)

	respBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(respBuf, checksum)

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			a := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: s.TTL},
				A:   respBuf,
			}
			resp.Answer = append(resp.Answer, a)
		case dns.TypeTXT:
			rawTxt, _ := implantBase64.Encode(respBuf)
			respTxt := string(rawTxt)
			txts := []string{}
			for start, stop := 0, 0; stop < len(respTxt); start = stop {
				stop += s.MaxTXTLength
				if len(respTxt) < stop {
					stop = len(respTxt)
				}
				txts = append(txts, respTxt[start:stop])
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: s.TTL},
				Txt: txts,
			}
			resp.Answer = append(resp.Answer, txt)
		}
	}
	return resp
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
		defer recoverAndLogPanic(dnsLog.Errorf, "dns handleCanary")

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
				canary.FirstTriggered = time.Now().Format(time.RFC1123)
			}
			canary.LatestTrigger = time.Now().Format(time.RFC1123)
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

func splitToChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize

		// If end is greater than the length of the data,
		// adjust it to be the length of data to avoid slicing beyond.
		if end > len(data) {
			end = len(data)
		}

		chunk := make([]byte, chunkSize)
		copy(chunk, data[i:end])

		chunks = append(chunks, chunk)
	}

	return chunks
}
