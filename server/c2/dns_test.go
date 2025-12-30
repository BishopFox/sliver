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
*/

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	insecureRand "math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/implant/sliver/transports/dnsclient"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
)

var (
	example1  = "1.example.com."
	example2  = "something-longer.example.com."
	example3  = "something-even-longer.example.computer."
	c2Domains = []string{example1, example2, example3}

	opts = &dnsclient.DNSOptions{
		QueryTimeout:       time.Duration(time.Second * 3),
		RetryWait:          time.Duration(time.Second * 3),
		RetryCount:         1,
		WorkersPerResolver: 1,
	}
)

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize-1)+1)
	rand.Read(buf)
	return buf
}

func shuffleDNSMsgs(a []*dnspb.DNSMessage) {
	for i := len(a) - 1; i > 0; i-- { // Fisher–Yates shuffle
		j := insecureRand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

type testDNSResolver struct {
	server *SliverDNSServer
	parent string
	base64 implantEncoders.Base64Encoder
}

func (r *testDNSResolver) Address() string {
	return "test-resolver"
}

func (r *testDNSResolver) A(domain string) ([]byte, time.Duration, error) {
	resp, err := r.exchange(domain, dns.TypeA)
	if err != nil {
		return nil, 0, err
	}
	records := []byte{}
	for _, answer := range resp.Answer {
		if a, ok := answer.(*dns.A); ok {
			records = append(records, []byte(a.A)...)
		}
	}
	return records, 0, nil
}

func (r *testDNSResolver) AAAA(domain string) ([]byte, time.Duration, error) {
	resp, err := r.exchange(domain, dns.TypeAAAA)
	if err != nil {
		return nil, 0, err
	}
	records := make([]byte, 512)
	dataSize := uint32(0)
	for _, answer := range resp.Answer {
		if aaaa, ok := answer.(*dns.AAAA); ok {
			chunkMeta := uint32(aaaa.Hdr.Ttl)
			chunkIdx := (chunkMeta & 0xff00) >> 8
			tempSize := chunkMeta & 0xff
			if dataSize != 0 && tempSize != dataSize {
				return nil, 0, dnsclient.ErrInvalidResponse
			}
			if dataSize == 0 {
				dataSize = tempSize
			}
			copy(records[chunkIdx*16:], []byte(aaaa.AAAA))
		}
	}
	if dataSize == 0 {
		return nil, 0, nil
	}
	data := make([]byte, dataSize)
	copy(data, records[:dataSize])
	return data, 0, nil
}

func (r *testDNSResolver) TXT(domain string) ([]byte, time.Duration, error) {
	resp, err := r.exchange(domain, dns.TypeTXT)
	if err != nil {
		return nil, 0, err
	}
	records := ""
	for _, answer := range resp.Answer {
		if txt, ok := answer.(*dns.TXT); ok {
			records += strings.Join(txt.Txt, "")
		}
	}
	if records == "" {
		return nil, 0, nil
	}
	data, err := r.base64.Decode([]byte(records))
	if err != nil {
		return nil, 0, err
	}
	return data, 0, nil
}

func (r *testDNSResolver) exchange(domain string, qtype uint16) (*dns.Msg, error) {
	req := new(dns.Msg)
	req.SetQuestion(domain, qtype)
	resp := r.server.handleC2(r.parent, req)
	if resp == nil {
		return nil, dnsclient.ErrInvalidResponse
	}
	if resp.Rcode != dns.RcodeSuccess {
		return resp, dnsclient.ErrInvalidResponse
	}
	return resp, nil
}

func randomDNSMsgs(t *testing.T, parent string, maxSize int, encoder encoders.Encoder, client *dnsclient.SliverDNSClient) ([]*dnspb.DNSMessage, []byte) {
	testData := randomDataRandomSize(maxSize)
	dnsMsgs := []*dnspb.DNSMessage{}
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_DATA_FROM_IMPLANT,
		Size: uint32(len(testData)),
	}
	domains, err := client.SplitBuffer(msg, encoder, testData)
	if err != nil {
		t.Fatalf("failed to encode sample: %s", err)
	}
	for _, domain := range domains {
		subdata := strings.TrimSuffix(domain, parent)
		subdata = strings.ReplaceAll(subdata, ".", "")
		data, err := encoder.Decode([]byte(subdata))
		if err != nil {
			t.Fatalf("Unexpected error decoding subdata: %s", err)
		}
		msg := &dnspb.DNSMessage{}
		err = proto.Unmarshal(data, msg)
		if err != nil {
			t.Fatalf("Unexpected error un-marshaling subdata: %s", err)
		}
		dnsMsgs = append(dnsMsgs, msg)
	}
	shuffleDNSMsgs(dnsMsgs)
	return dnsMsgs, testData
}

func TestPendingEnvelopes(t *testing.T) {
	// *** Small ***
	for i := 0; i < 100; i++ {
		reassemble(t, example1, 256, encoders.Base32{})
	}
	// *** Large ***
	for i := 0; i < 100; i++ {
		reassemble(t, example1, 30*1024, encoders.Base32{})
	}
}

func reassemble(t *testing.T, parent string, size int, encoder encoders.Encoder) {
	client := dnsclient.NewDNSClient(example1, opts)
	dnsMsgs, original := randomDNSMsgs(t, example1, size, encoder, client)
	dnsSession := DNSSession{
		ID:                dnsSessionID(),
		incomingEnvelopes: map[uint32]*PendingEnvelope{},
		incomingMutex:     &sync.Mutex{},
	}

	// Re-assemble original message
	t.Logf("Re-assembling %d messages", len(dnsMsgs))
	pending := dnsSession.IncomingPendingEnvelope(dnsMsgs[0].ID, dnsMsgs[0].Size)
	complete := pending.Insert(dnsMsgs[0])
	t.Logf("Inserted: %v", dnsMsgs[0])
	if !complete {
		for _, dnsMsg := range dnsMsgs[1:] {
			complete = pending.Insert(dnsMsg)
			t.Logf("Inserted: %v", dnsMsg)
			if complete {
				break
			}
		}
	}
	data, err := pending.Reassemble()
	if err != nil {
		t.Logf("Original (%d): %v", len(original), original)
		for k, v := range pending.messages {
			t.Logf("%d | %v", k, v)
		}
		t.Fatalf("Failed to reassemble pending envelope: %s", err)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("Reassembled data does not match original\nOriginal: %v\nData: %v", original, data)
	}
}

func TestDNSHelloRoundTrip(t *testing.T) {
	server := newTestDNSServer()
	resolver := &testDNSResolver{server: server, parent: example1}

	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_TOTP,
		ID:   0,
	}
	domain, err := encodeMessageDomain(example1, msg)
	if err != nil {
		t.Fatalf("encodeMessageDomain failed: %s", err)
	}
	data, _, err := resolver.A(domain)
	if err != nil {
		t.Fatalf("resolver.A failed: %s", err)
	}
	if len(data) != 4 {
		t.Fatalf("expected 4 bytes in response, got %d", len(data))
	}
	sessionID := binary.LittleEndian.Uint32(data) & sessionIDBitMask
	if sessionID == 0 {
		t.Fatal("expected non-zero session id")
	}
	if _, ok := server.sessions.Load(sessionID); !ok {
		t.Fatalf("expected session %d to be stored", sessionID)
	}
}

func TestDNSNOPRoundTripChecksum(t *testing.T) {
	server := newTestDNSServer()
	sessionID := uint32(0x123456)
	dnsSession := newTestDNSSession(sessionID)
	server.sessions.Store(sessionID, dnsSession)

	payload := make([]byte, 16)
	rand.Read(payload)
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_NOP,
		ID:   (1 << 24) | sessionID,
		Data: payload,
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}
	domain, err := encodeMessageDomain(example1, msg)
	if err != nil {
		t.Fatalf("encodeMessageDomain failed: %s", err)
	}
	resolver := &testDNSResolver{server: server, parent: example1}
	data, _, err := resolver.A(domain)
	if err != nil {
		t.Fatalf("resolver.A failed: %s", err)
	}
	if len(data) != 4 {
		t.Fatalf("expected 4 bytes in response, got %d", len(data))
	}
	got := binary.LittleEndian.Uint32(data)
	want := crc32.ChecksumIEEE(raw)
	if got != want {
		t.Fatalf("checksum mismatch: got %d want %d", got, want)
	}
}

func TestDNSPollManifestReuse(t *testing.T) {
	server := newTestDNSServer()
	sessionID := uint32(0x654321)
	dnsSession := newTestDNSSession(sessionID)
	server.sessions.Store(sessionID, dnsSession)
	resolver := &testDNSResolver{server: server, parent: example1}

	pollMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_POLL,
		ID:   sessionID,
		Data: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	pollDomain, err := encodeMessageDomain(example1, pollMsg)
	if err != nil {
		t.Fatalf("encodeMessageDomain failed: %s", err)
	}

	respData, _, err := resolver.TXT(pollDomain)
	if err != nil {
		t.Fatalf("resolver.TXT failed: %s", err)
	}
	manifest := &dnspb.DNSMessage{}
	if err := proto.Unmarshal(respData, manifest); err != nil {
		t.Fatalf("manifest unmarshal failed: %s", err)
	}
	if manifest.Type != dnspb.DNSMessageType_MANIFEST || manifest.Size != 0 || manifest.ID != 0 {
		t.Fatalf("unexpected empty manifest: %+v", manifest)
	}

	msgID := uint32(0x01000000 | sessionID)
	dnsSession.outgoingMsgIDs = append(dnsSession.outgoingMsgIDs, msgID)
	dnsSession.outgoingBuffers[msgID] = bytes.Repeat([]byte{0xAB}, 32)

	respData, _, err = resolver.TXT(pollDomain)
	if err != nil {
		t.Fatalf("resolver.TXT failed: %s", err)
	}
	manifest = &dnspb.DNSMessage{}
	if err := proto.Unmarshal(respData, manifest); err != nil {
		t.Fatalf("manifest unmarshal failed: %s", err)
	}
	if manifest.Type != dnspb.DNSMessageType_MANIFEST || manifest.ID != msgID || manifest.Size != 32 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if dnsSession.dnsIdMsgIdMap[pollMsg.ID] != msgID {
		t.Fatalf("expected dnsIdMsgIdMap to track msg id %d", msgID)
	}

	respData, _, err = resolver.TXT(pollDomain)
	if err != nil {
		t.Fatalf("resolver.TXT failed: %s", err)
	}
	manifest = &dnspb.DNSMessage{}
	if err := proto.Unmarshal(respData, manifest); err != nil {
		t.Fatalf("manifest unmarshal failed: %s", err)
	}
	if manifest.Type != dnspb.DNSMessageType_MANIFEST || manifest.ID != msgID || manifest.Size != 32 {
		t.Fatalf("unexpected reused manifest: %+v", manifest)
	}
}

func TestDNSReadEnvelopeRoundTripTXT(t *testing.T) {
	runDNSReadEnvelopeRoundTrip(t, false)
}

func TestDNSReadEnvelopeRoundTripAAAA(t *testing.T) {
	runDNSReadEnvelopeRoundTrip(t, true)
}

func TestDNSWriteEnvelopeRoundTrip(t *testing.T) {
	server := newTestDNSServer()
	sessionID := uint32(0x102030)
	key := implantCrypto.RandomSymmetricKey()
	dnsSession := newTestDNSSession(sessionID)
	dnsSession.CipherCtx = serverCrypto.NewCipherContext(key)
	dnsSession.ImplantConn = core.NewImplantConnection("dns", "n/a")
	respCh := make(chan *sliverpb.Envelope, 1)
	dnsSession.ImplantConn.Resp[77] = respCh
	server.sessions.Store(sessionID, dnsSession)

	resolver := &testDNSResolver{server: server, parent: example1}

	envelope := &sliverpb.Envelope{
		ID:   77,
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	}
	sendEnvelopeViaDNS(t, resolver, sessionID, key, envelope)

	select {
	case got := <-respCh:
		if !proto.Equal(got, envelope) {
			t.Fatalf("envelope mismatch: %#v != %#v", got, envelope)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for envelope")
	}
}

func TestDNSDataToImplantRejectsInvalidRead(t *testing.T) {
	server := newTestDNSServer()
	sessionID := uint32(0x121212)
	dnsSession := newTestDNSSession(sessionID)
	dnsSession.outgoingBuffers[0x01000000|sessionID] = bytes.Repeat([]byte{0xCD}, 16)
	server.sessions.Store(sessionID, dnsSession)

	msg := &dnspb.DNSMessage{
		Type:  dnspb.DNSMessageType_DATA_TO_IMPLANT,
		ID:    0x01000000 | sessionID,
		Start: 5,
		Stop:  4,
	}
	domain, err := encodeMessageDomain(example1, msg)
	if err != nil {
		t.Fatalf("encodeMessageDomain failed: %s", err)
	}
	req := new(dns.Msg)
	req.SetQuestion(domain, dns.TypeTXT)
	resp := server.handleC2(example1, req)
	if resp == nil || resp.Rcode != dns.RcodeRefused {
		t.Fatalf("expected refused response, got %+v", resp)
	}
}

func TestIsC2Domain(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false, true)
	isC2, domain := listener.isC2SubDomain(c2Domains, "asdf.1.example.com.")
	if !isC2 {
		t.Fatal("IsC2Domain expected true, got false")
	}
	if domain != example1 {
		t.Fatal("IsC2Domain expected example1, got", domain)
	}
	isC2, _ = listener.isC2SubDomain(c2Domains, "asdf.1.foobar.com.")
	if isC2 {
		t.Fatal("IsC2Domain expected false, got true (1)")
	}
	isC2, _ = listener.isC2SubDomain(c2Domains, "asdf.2.example.com.")
	if isC2 {
		t.Fatal("IsC2Domain expected false, got true (2)")
	}
	isC2, _ = listener.isC2SubDomain(c2Domains, fmt.Sprintf("asdf.asdf.asdf%s", example3))
	if isC2 {
		t.Fatal("IsC2Domain expected false, got true (3)")
	}
}

func TestDecodeSubdataBase32(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false, true)
	payload := make([]byte, 64)
	rand.Read(payload)
	original := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_NOP,
		ID:   42,
		Data: payload,
	}
	raw, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal message: %s", err)
	}
	encoded, _ := encoders.Base32{}.Encode(raw)
	if len(encoded) < 12 {
		t.Fatalf("encoded sample too short: %d", len(encoded))
	}
	subdomain := fmt.Sprintf("%s.%s", string(encoded[:12]), string(encoded[12:]))
	for _, sample := range []string{subdomain, strings.ToUpper(subdomain)} {
		msg, checksum, err := listener.decodeSubdata(sample)
		if err != nil {
			t.Fatalf("decodeSubdata failed: %s", err)
		}
		if !proto.Equal(msg, original) {
			t.Fatalf("decoded message mismatch: %#v != %#v", msg, original)
		}
		if checksum != crc32.ChecksumIEEE(raw) {
			t.Fatalf("unexpected checksum: %d", checksum)
		}
	}
}

func TestDecodeSubdataRejectsInvalid(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false, true)
	_, _, err := listener.decodeSubdata("invalid")
	if err != ErrInvalidMsg {
		t.Fatalf("expected ErrInvalidMsg, got %v", err)
	}
}

func TestDecodeSubdataRejectsInvalidProto(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false, true)
	encoded, err := implantEncoders.Base32Encoder{}.Encode([]byte{0x00})
	if err != nil {
		t.Fatalf("encode failed: %s", err)
	}
	_, _, err = listener.decodeSubdata(string(encoded))
	if err != ErrInvalidMsg {
		t.Fatalf("expected ErrInvalidMsg, got %v", err)
	}
}

func TestHandleC2RejectsUnknownSession(t *testing.T) {
	server := newTestDNSServer()
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_NOP,
		ID:   0x00123456,
		Data: []byte{1, 2, 3},
	}
	domain, err := encodeMessageDomain(example1, msg)
	if err != nil {
		t.Fatalf("encodeMessageDomain failed: %s", err)
	}
	req := new(dns.Msg)
	req.SetQuestion(domain, dns.TypeA)
	resp := server.handleC2(example1, req)
	if resp == nil || resp.Rcode != dns.RcodeNameError {
		t.Fatalf("expected name error response, got %+v", resp)
	}
}

func TestSplitToChunksPadding(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	chunks := splitToChunks(data, 4)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], []byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected first chunk: %v", chunks[0])
	}
	if !bytes.Equal(chunks[1], []byte{5, 6, 0, 0}) {
		t.Fatalf("unexpected second chunk: %v", chunks[1])
	}
	for _, chunk := range chunks {
		if len(chunk) != 4 {
			t.Fatalf("unexpected chunk length: %d", len(chunk))
		}
	}
}

func TestOutgoingReadBoundaries(t *testing.T) {
	session := &DNSSession{
		outgoingBuffers: map[uint32][]byte{
			1: {1, 2, 3, 4},
		},
		outgoingMutex: &sync.RWMutex{},
	}
	if _, err := session.OutgoingRead(1, 0, 0); err == nil {
		t.Fatal("expected error for stop <= start")
	}
	if _, err := session.OutgoingRead(1, 0, 5); err == nil {
		t.Fatal("expected error for stop > length")
	}
	if _, err := session.OutgoingRead(1, 5, 6); err == nil {
		t.Fatal("expected error for start > length")
	}
	if _, err := session.OutgoingRead(2, 0, 1); err == nil {
		t.Fatal("expected error for missing buffer")
	}
	read, err := session.OutgoingRead(1, 1, 3)
	if err != nil {
		t.Fatalf("unexpected error for valid read: %s", err)
	}
	if !bytes.Equal(read, []byte{2, 3}) {
		t.Fatalf("unexpected read data: %v", read)
	}
}

func runDNSReadEnvelopeRoundTrip(t *testing.T, noTXT bool) {
	t.Helper()
	server := newTestDNSServer()
	sessionID := uint32(0x2468AC)
	key := implantCrypto.RandomSymmetricKey()
	dnsSession := newTestDNSSession(sessionID)
	dnsSession.CipherCtx = serverCrypto.NewCipherContext(key)
	server.sessions.Store(sessionID, dnsSession)

	envelope := &sliverpb.Envelope{
		ID:   99,
		Type: sliverpb.MsgPing,
		Data: []byte("roundtrip"),
	}
	if err := dnsSession.StageOutgoingEnvelope(envelope); err != nil {
		t.Fatalf("StageOutgoingEnvelope failed: %s", err)
	}
	if len(dnsSession.outgoingMsgIDs) != 1 {
		t.Fatalf("expected 1 outgoing message id, got %d", len(dnsSession.outgoingMsgIDs))
	}
	msgID := dnsSession.outgoingMsgIDs[0]

	resolver := &testDNSResolver{server: server, parent: example1}
	got, err := readEnvelopeViaDNS(t, resolver, sessionID, key, noTXT)
	if err != nil {
		t.Fatalf("ReadEnvelope failed: %s", err)
	}
	if got == nil || !proto.Equal(got, envelope) {
		t.Fatalf("envelope mismatch: %#v != %#v", got, envelope)
	}
	if _, ok := dnsSession.outgoingBuffers[msgID]; ok {
		t.Fatalf("expected outgoing buffer %d to be cleared", msgID)
	}
}

func newTestDNSServer() *SliverDNSServer {
	return &SliverDNSServer{
		sessions:     &sync.Map{},
		messages:     &sync.Map{},
		TTL:          0,
		MaxTXTLength: defaultMaxTXTLength,
		EnforceOTP:   false,
	}
}

func newTestDNSSession(sessionID uint32) *DNSSession {
	return &DNSSession{
		ID:                sessionID,
		dnsIdMsgIdMap:     map[uint32]uint32{},
		outgoingMsgIDs:    []uint32{},
		outgoingBuffers:   map[uint32][]byte{},
		outgoingMutex:     &sync.RWMutex{},
		incomingEnvelopes: map[uint32]*PendingEnvelope{},
		incomingMutex:     &sync.Mutex{},
		msgCount:          0,
	}
}

func encodeMessageDomain(parent string, msg *dnspb.DNSMessage) (string, error) {
	raw, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	encoded, err := implantEncoders.Base32Encoder{}.Encode(raw)
	if err != nil {
		return "", err
	}
	return joinSubdataToParent(string(encoded), parent), nil
}

func joinSubdataToParent(subdata string, parent string) string {
	subdomains := []string{}
	for i := 0; i < len(subdata); i += 63 {
		stop := i + 63
		if len(subdata) < stop {
			stop = len(subdata)
		}
		subdomains = append(subdomains, subdata[i:stop])
	}
	return strings.Join(subdomains, ".") + parent
}

func sendEnvelopeViaDNS(t *testing.T, resolver *testDNSResolver, sessionID uint32, key [32]byte, envelope *sliverpb.Envelope) {
	t.Helper()
	plaintext, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope failed: %s", err)
	}
	cipherCtx := implantCrypto.NewCipherContext(key)
	ciphertext, err := cipherCtx.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt envelope failed: %s", err)
	}

	client := dnsclient.NewDNSClient(example1, opts)
	msgID := uint32(0x01000000 | sessionID)
	msg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_DATA_FROM_IMPLANT,
		ID:   msgID,
		Size: uint32(len(ciphertext)),
	}
	domains, err := client.SplitBuffer(msg, implantEncoders.Base32Encoder{}, ciphertext)
	if err != nil {
		t.Fatalf("SplitBuffer failed: %s", err)
	}

	subMsgs := make([]*dnspb.DNSMessage, 0, len(domains))
	for _, domain := range domains {
		subdata := strings.TrimSuffix(domain, example1)
		subdata = strings.ReplaceAll(subdata, ".", "")
		raw, err := implantEncoders.Base32Encoder{}.Decode([]byte(subdata))
		if err != nil {
			t.Fatalf("decode subdata failed: %s", err)
		}
		subMsg := &dnspb.DNSMessage{}
		if err := proto.Unmarshal(raw, subMsg); err != nil {
			t.Fatalf("unmarshal submsg failed: %s", err)
		}
		subMsgs = append(subMsgs, subMsg)
	}
	shuffleDNSMsgs(subMsgs)

	for _, subMsg := range subMsgs {
		raw, err := proto.Marshal(subMsg)
		if err != nil {
			t.Fatalf("marshal submsg failed: %s", err)
		}
		domain, err := encodeMessageDomain(example1, subMsg)
		if err != nil {
			t.Fatalf("encodeMessageDomain failed: %s", err)
		}
		data, _, err := resolver.A(domain)
		if err != nil {
			t.Fatalf("resolver.A failed: %s", err)
		}
		if len(data) != 4 {
			t.Fatalf("expected 4-byte checksum, got %d", len(data))
		}
		got := binary.LittleEndian.Uint32(data)
		want := crc32.ChecksumIEEE(raw)
		if got != want {
			t.Fatalf("checksum mismatch: got %d want %d", got, want)
		}
	}
}

func readEnvelopeViaDNS(t *testing.T, resolver *testDNSResolver, sessionID uint32, key [32]byte, noTXT bool) (*sliverpb.Envelope, error) {
	t.Helper()
	pollMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_POLL,
		ID:   sessionID,
		Data: randomBytes(8),
	}
	pollDomain, err := encodeMessageDomain(example1, pollMsg)
	if err != nil {
		return nil, err
	}

	var respData []byte
	if noTXT {
		respData, _, err = resolver.AAAA(pollDomain)
	} else {
		respData, _, err = resolver.TXT(pollDomain)
	}
	if err != nil {
		return nil, err
	}
	if len(respData) == 0 {
		return nil, nil
	}
	manifest := &dnspb.DNSMessage{}
	if err := proto.Unmarshal(respData, manifest); err != nil {
		return nil, err
	}
	if manifest.Type != dnspb.DNSMessageType_MANIFEST || manifest.Size == 0 {
		return nil, nil
	}

	bytesPerChunk := uint32(182)
	if noTXT {
		bytesPerChunk = 192
	}
	recvBuf := make([]byte, manifest.Size)
	for start := uint32(0); start < manifest.Size; start += bytesPerChunk {
		stop := start + bytesPerChunk
		if manifest.Size < stop {
			stop = manifest.Size
		}
		readMsg := &dnspb.DNSMessage{
			ID:    manifest.ID,
			Type:  dnspb.DNSMessageType_DATA_TO_IMPLANT,
			Start: start,
			Stop:  stop,
		}
		domain, err := encodeMessageDomain(example1, readMsg)
		if err != nil {
			return nil, err
		}
		var readData []byte
		if noTXT {
			readData, _, err = resolver.AAAA(domain)
		} else {
			readData, _, err = resolver.TXT(domain)
		}
		if err != nil {
			return nil, err
		}
		recvMsg := &dnspb.DNSMessage{}
		if err := proto.Unmarshal(readData, recvMsg); err != nil {
			return nil, err
		}
		copy(recvBuf[recvMsg.Start:], recvMsg.Data)
	}

	plain, err := implantCrypto.NewCipherContext(key).Decrypt(recvBuf)
	if err != nil && err != implantCrypto.ErrReplayAttack {
		return nil, err
	}
	envelope := &sliverpb.Envelope{}
	if err := proto.Unmarshal(plain, envelope); err != nil {
		return nil, err
	}

	clearMsg := &dnspb.DNSMessage{
		Type: dnspb.DNSMessageType_CLEAR,
		ID:   manifest.ID,
		Data: randomBytes(8),
	}
	clearDomain, err := encodeMessageDomain(example1, clearMsg)
	if err != nil {
		return nil, err
	}
	if _, _, err := resolver.A(clearDomain); err != nil {
		return nil, err
	}
	return envelope, nil
}

func randomBytes(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}
