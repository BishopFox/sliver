package c2

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
	------------------------------------------------------------------------

	DNS command and control implementation

*/

import (
	secureRand "crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"hash/crc32"
	insecureRand "math/rand"
	"net"

	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"

	"encoding/base64"
	"encoding/binary"
	"encoding/pem"

	"fmt"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/log"

	"github.com/miekg/dns"
)

const (
	sessionIDSize = 12

	// Max TXT record is 255, records are b64 so (n*8 + 5) / 6 = ~250
	byteBlockSize = 185 // Can be as high as n = 187, but we'll leave some slop
	blockIDSize   = 6
)

var (
	dnsLog = log.NamedLogger("c2", "dns")

	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789_")

	dnsSessionsMutex = &sync.RWMutex{}
	dnsSessions      = map[string]*DNSSession{}

	sendBlocksMutex = &sync.RWMutex{}
	sendBlocks      = map[uint32]*Block{}

	recvBlocksMutex = &sync.RWMutex{}
	recvBlocks      = map[uint32]*Block{}
)

// Block - A blob of data that we're sending or receiving, blocks of data
// are split up into arrays of bytes (chunks) that are encoded per-query
// the amount of data that can be encoded into a single request or response
// varies depending on the type of query and the length of the parent domain.
type Block struct {
	ID      uint32
	data    [][]byte
	Size    int
	Started time.Time
	Mutex   sync.RWMutex
}

func (b *Block) AddData(index int, data []byte) (bool, error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if len(b.data) < index+1 {
		return false, errors.New("Data index out of bounds")
	}
	b.data[index] = data
	sum := 0
	for _, data := range b.data {
		sum += len(data)
	}
	return sum == b.Size, nil
}

func (b *Block) GetData(index int) ([]byte, error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if len(b.data) < index+1 {
		return nil, errors.New("Data index out of bounds")
	}
	return b.data[index], nil
}

func (b *Block) Reassemble() []byte {
	b.Mutex.RLock()
	defer b.Mutex.RUnlock()
	data := []byte{}
	for _, block := range b.data {
		data = append(data, block...)
	}
	return data
}

// DNSSession - Holds DNS session information
type DNSSession struct {
	ID      string
	Session *core.Session
	Key     cryptography.AESKey
	replay  map[string]bool // Sessions are mutex 'd
}

func (s *DNSSession) isReplayAttack(ciphertext []byte) bool {
	if len(ciphertext) < 1 {
		return false
	}
	sha := sha256.New()
	sha.Write(ciphertext)
	digest := base64.RawStdEncoding.EncodeToString(sha.Sum(nil))
	if _, ok := s.replay[digest]; ok {
		return true
	}
	s.replay[digest] = true
	return false
}

// --------------------------- DNS SERVER ---------------------------

// StartDNSListener - Start a DNS listener
func StartDNSListener(bindIface string, lport uint16, domains []string, canaries bool) *dns.Server {
	StartPivotListener()
	dnsLog.Infof("Starting DNS listener for %v (canaries: %v) ...", domains, canaries)

	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		req.Question[0].Name = strings.ToLower(req.Question[0].Name)
		handleDNSRequest(domains, canaries, writer, req)
	})

	server := &dns.Server{Addr: fmt.Sprintf("%s:%d", bindIface, lport), Net: "udp"}
	return server
}

// ---------------------------
// DNS Handler
// ---------------------------
// Handles all DNS queries, first we determine if the query is C2 or a canary
func handleDNSRequest(domains []string, canaries bool, writer dns.ResponseWriter, req *dns.Msg) {
	if req == nil {
		dnsLog.Info("req can not be nil")
		return
	}

	if len(req.Question) < 1 {
		dnsLog.Info("No questions in DNS request")
		return
	}

	var resp *dns.Msg
	isC2, domain := isC2SubDomain(domains, req.Question[0].Name)
	if isC2 {
		dnsLog.Debugf("'%s' is subdomain of c2 parent '%s'", req.Question[0].Name, domain)
		resp = handleC2(domain, req)
	} else if canaries {
		dnsLog.Debugf("checking '%s' for DNS canary matches", req.Question[0].Name)
		resp = handleCanary(req)
	}

	if resp != nil {
		writer.WriteMsg(resp)
	} else {
		dnsLog.Infof("Invalid query, no DNS response")
	}
}

// Returns true if the requested domain is a c2 subdomain, and the domain it matched with
func isC2SubDomain(domains []string, reqDomain string) (bool, string) {
	for _, parentDomain := range domains {
		if dns.IsSubDomain(parentDomain, reqDomain) {
			dnsLog.Infof("'%s' is subdomain of '%s'", reqDomain, parentDomain)
			return true, parentDomain
		}
	}
	dnsLog.Infof("'%s' is NOT subdomain of any %v", reqDomain, domains)
	return false, ""
}

// The query is C2, pass to the appropriate record handler this is done
// so the record handler can encode the response based on the type of
// record that was requested
func handleC2(domain string, req *dns.Msg) *dns.Msg {
	subdomain := req.Question[0].Name[:len(req.Question[0].Name)-len(domain)]
	if strings.HasSuffix(subdomain, ".") {
		subdomain = subdomain[:len(subdomain)-1]
	}
	dnsLog.Infof("processing req for subdomain = %s", subdomain)
	switch req.Question[0].Qtype {
	case dns.TypeTXT:
		return handleTXT(domain, subdomain, req)
	case dns.TypeA:
		return handleA(domain, subdomain, req)
	default:
	}
	return nil
}

// ---------------------------
// Canary Record Handler
// ---------------------------
// Canary -> valid? -> trigger alert event
func handleCanary(req *dns.Msg) *dns.Msg {
	reqDomain := strings.ToLower(req.Question[0].Name)
	if !strings.HasSuffix(reqDomain, ".") {
		reqDomain += "." // Ensure we have the FQDN
	}
	canary, err := db.CanaryByDomain(reqDomain)
	if err != nil {
		return nil
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
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

	// Respond with random IPs
	switch req.Question[0].Qtype {
	case dns.TypeA:
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   randomIP(),
		})
	default:
	}

	return resp
}

func randomIP() net.IP {
	randBuf := make([]byte, 4)
	secureRand.Read(randBuf)

	// Ensure non-zeros with various bitmasks
	return net.IPv4(randBuf[0]|0x10, randBuf[1]|0x10, randBuf[2]|0x1, randBuf[3]|0x10)
}

// ---------------------------
// C2 Record Handlers
// ---------------------------
func handleTXT(domain string, subdomain string, req *dns.Msg) *dns.Msg {
	q := req.Question[0]
	resp := new(dns.Msg)
	resp.SetReply(req)

	checksum, dnsMsg := parseC2Query(subdomain)
	if dnsMsg == nil {
		dnsLog.Errorf("Failed to parse TXT query")
		return nil
	}

	// Execute action based on message type

	txtRecords := []string{}

	switch dnsMsg.Type {
	case dnspb.DNSMessageType_DOMAIN_KEY:
		blockID := handleDomainKeyQuery(domain, dnsMsg)
		data, _ := proto.Marshal(&dnspb.DNSMessage{
			N:    checksum,
			Data: blockID,
		})
		record := string(new(encoders.Base64).Encode(data))
		txtRecords = append(txtRecords, record)
	}

	txt := &dns.TXT{
		Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: txtRecords,
	}
	resp.Answer = append(resp.Answer, txt)
	return resp
}

func handleA(domain string, subdomain string, req *dns.Msg) *dns.Msg {
	q := req.Question[0]
	resp := new(dns.Msg)
	resp.SetReply(req)

	checksum, dnsMsg := parseC2Query(subdomain)
	if dnsMsg == nil {
		dnsLog.Errorf("Failed to parse A query")
		return nil
	}

	switch dnsMsg.Type {
	case dnspb.DNSMessageType_NOP:
		break
	}

	checksumBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(checksumBuf, checksum)
	resp.Answer = append(resp.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   checksumBuf,
	})
	return resp
}

// ---------------------------
// C2 Handler
// ---------------------------
// This parses the C2 message and performs actions based on the request
// the response is a CRC32 of the encoded data in the message
func parseC2Query(subdomain string) (uint32, *dnspb.DNSMessage) {
	subdomainData := strings.Join(strings.Split(subdomain, "."), "")
	dnsLog.Debugf("Subdomain = %v, Data = %v", subdomain, subdomainData)
	dnsMsg := decodeMessage(subdomainData)
	if dnsMsg == nil {
		return 0, nil
	}
	checksum := crc32.Checksum(dnsMsg.Data, crc32.IEEETable)
	return checksum, dnsMsg
}

func decodeMessage(subdomainData string) *dnspb.DNSMessage {
	encoders := detectEncoding(subdomainData)
	query := &dnspb.DNSMessage{}
	for _, encoder := range encoders {
		dnsLog.Debugf("Attempting to decode subdomain data with %v", encoder)
		data, err := encoder.Decode([]byte(subdomainData))
		if err != nil {
			dnsLog.Debugf("Decode attempt failed %s, continue ...", err)
			continue
		}
		err = proto.Unmarshal(data, query)
		if err != nil {
			dnsLog.Debugf("Failed to parse pb %s, continue ...", err)
			continue
		}
		return query
	}
	return nil
}

// This function attempts to detect the most likely encoder used by the implant
// we return a list of encoders in order of most likely to least likely
func detectEncoding(subdomainData string) []encoders.Encoder {

	// Our version of base32 is missing chars: i, l, o, s
	// if any of these appear the message must be base62
	base62Chars := strings.ContainsAny(subdomainData, "silo") // "silo" simply appeases my spell checker
	if base62Chars {
		return []encoders.Encoder{new(encoders.Base62)}
	}

	// Okay now we detect if the message only contains lower case letters
	// our base32 encoder only uses lower case but a resolver could have
	// messed with the domain letter cases on the way over to us. So we
	// say it's likely base62 if it contains a mix but fallback to base32
	if strings.ContainsAny(subdomainData, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return []encoders.Encoder{new(encoders.Base62), new(encoders.Base32)}
	} else {
		return []encoders.Encoder{new(encoders.Base32), new(encoders.Base64)}
	}
}

// --------------------------- DNS HANDLERS ---------------------------
func encodeSendBlock(domain string, data []byte) *Block {
	encodedData := new(encoders.Base64).Encode(data)
	blockData := [][]byte{}
	for index := 0; index < len(encodedData); {
		end := index + 254
		if len(encodedData) < end {
			end = len(encodedData)
		}
		blockData = append(blockData, encodedData[index:end])
		index += 254
	}
	return &Block{
		ID:      blockID(),
		Size:    len(data),
		data:    blockData,
		Started: time.Now(),
		Mutex:   sync.RWMutex{},
	}
}

func handleDomainKeyQuery(domain string, query *dnspb.DNSMessage) []byte {
	domainKey, err := getDomainKeyFor(domain)
	if err != nil {
		dnsLog.Errorf("Failed to find domain key for '%s' %s", domain, err)
		return nil
	}
	block := encodeSendBlock(domain, domainKey)
	sendBlocksMutex.Lock()
	defer sendBlocksMutex.Unlock()
	sendBlocks[block.ID] = block
	return []byte{}
}

// Returns an confirmation value (e.g. exit code 0 non-0) and error
func startDNSSession(msgID uint32, domain string) ([]string, error) {

	dnsLog.Infof("Complete session init message received, reassembling ...")

	publicKeyPEM, privateKeyPEM, err := certs.GetCertificate(certs.C2ServerCA, certs.RSAKey, domain)
	if err != nil {
		dnsLog.Infof("Failed to fetch RSA private key")
		return []string{"1"}, err
	}
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	dnsLog.Infof("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		dnsLog.Infof("Failed to decode RSA private key")
		return []string{"1"}, err
	}

	dnsLog.Debugf("Session Init: %v", encryptedSessionInit)
	sessionInitData, err := cryptography.RSADecrypt(encryptedSessionInit, privateKey)
	if err != nil {
		dnsLog.Infof("Failed to decrypt session init msg")
		return []string{"1"}, err
	}

	sessionInit := &sliverpb.DNSSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	dnsLog.Infof("Received new session in request")

	session := &core.Session{
		ID:            core.NextSessionID(),
		Transport:     "dns",
		RemoteAddress: "n/a",
		Send:          make(chan *sliverpb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
	}
	session.UpdateCheckin()

	aesKey, _ := cryptography.AESKeyFromBytes(sessionInit.Key)
	sessionID := dnsSessionID()
	dnsLog.Infof("Starting new DNS session with id = %s", sessionID)
	dnsSessionsMutex.Lock()
	dnsSessions[sessionID] = &DNSSession{
		ID:      sessionID,
		Session: session,
		Key:     aesKey,
		replay:  map[string]bool{},
	}
	dnsSessionsMutex.Unlock()

	encryptedSessionID, _ := cryptography.GCMEncrypt(aesKey, []byte(sessionID))

	return result, nil
}

func fingerprintSHA256(block *pem.Block) string {
	hash := sha256.Sum256(block.Bytes)
	b64hash := base64.RawStdEncoding.EncodeToString(hash[:])
	return strings.TrimRight(b64hash, "=")
}

// SessionIDs are public parameters in this use case
// so it's only important that they're unique
func dnsSessionID() string {
	insecureRand.Seed(time.Now().UnixNano())
	sessionID := []rune{}
	for i := 0; i < sessionIDSize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		sessionID = append(sessionID, dnsCharSet[index])
	}
	return string(sessionID)
}

func blockID() uint32 {
	randBuf := make([]byte, 4)
	secureRand.Read(randBuf)
	return binary.LittleEndian.Uint32(randBuf)
}

func getDomainKeyFor(domain string) ([]byte, error) {
	_, _, err := certs.GetCertificate(certs.C2ServerCA, certs.RSAKey, domain)
	if err != nil {
		certs.C2ServerGenerateRSACertificate(domain)
	}
	certPEM, _, err := certs.GetCertificate(certs.C2ServerCA, certs.RSAKey, domain)
	if err != nil {
		return nil, err
	}
	return certPEM, nil
}
