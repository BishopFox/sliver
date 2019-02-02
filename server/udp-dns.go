package main

/*

DNS Tunnel Implementation


*/

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"sort"

	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"

	"errors"
	"fmt"
	"log"
	insecureRand "math/rand"
	pb "sliver/protobuf"
	"sliver/server/cryptography"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/miekg/dns"
)

const (
	sessionIDSize = 8

	domainKeyMsg  = "_domainkey"
	blockReqMsg   = "b"
	clearBlockMsg = "cb"

	sessionInitMsg    = "si"
	sessionPollingMsg = "sp"

	// Max TXT record is 255, so (n*8 + 5) / 6 = ~250 (250 bytes per block + 4 byte sequence number)
	byteBlockSize = 185 // Can be as high as n = 187, but we'll leave some slop

	blockIDSize = 6
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")

	sendBlocksMutex = &sync.RWMutex{}
	sendBlocks      = &map[string]*SendBlock{}

	dnsSessionsMutex = &sync.RWMutex{}
	dnsSessions      = &map[string]*DNSSession{}

	blockReassemblerMutex = &sync.RWMutex{}
	blockReassembler      = &map[string][][]byte{}

	initReassemblerMutex = &sync.RWMutex{}
	initReassembler      = &map[string](*map[int][]string){}
)

// SendBlock - Data is encoded and split into `Blocks`
type SendBlock struct {
	ID   string
	Data []string
}

// DNSSession - Holds DNS session information
type DNSSession struct {
	ID            string
	SessionInitID string
	SliverName    string
	Sliver        *Sliver
	Key           cryptography.AESKey
	LastCheckin   time.Time
	replay        map[string]bool // Sessions are mutex'd
}

func (s *DNSSession) isReplayAttack(ciphertext []byte) bool {
	sha := sha256.New()
	sha.Write(ciphertext)
	digest := base64.StdEncoding.EncodeToString(sha.Sum(nil))
	if _, ok := s.replay[digest]; ok {
		return true
	}
	s.replay[digest] = true
	return false
}

// --------------------------- DNS SERVER ---------------------------

func startDNSListener(domain string) *dns.Server {

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
		blockID, size := getDomainKeyFor(domain)
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: []string{fmt.Sprintf("%s.%d", blockID, size)},
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

	case "_" + sessionInitMsg:
		fallthrough
	case sessionInitMsg: // Session init: (data)...(seq).(nonce).(_)si.example.com
		result, err := startDNSSession(domain, fields)
		if err != nil {
			log.Printf("Error during session init: %v", err)
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
			Txt: []string{result},
		}
		resp.Answer = append(resp.Answer, txt)

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

func getFieldNonce(fields []string) (string, error) {
	if len(fields) < 2 {
		return "", errors.New("Invalid number of fields in session init message (nounce)")
	}
	return fields[len(fields)-2], nil
}

func getFieldSeq(fields []string) (int, error) {
	if len(fields) < 3 {
		return -1, errors.New("Invalid number of fields in session init message (seq)")
	}
	rawSeq := fields[len(fields)-3]
	data, err := dnsDecodeString(rawSeq)
	if err != nil {
		log.Printf("Failed to decode seq field: %#v", rawSeq)
		return 0, err
	}
	index := int(binary.LittleEndian.Uint32(data))

	return index, nil
}

func getFieldSubdata(fields []string) ([]string, error) {
	if len(fields) < 4 {
		return []string{}, errors.New("Invalid number of fields in session init message (subdata)")
	}
	subdataFields := len(fields) - 3
	log.Printf("Domain contains %d subdata fields", subdataFields)
	return fields[:subdataFields], nil
}

// --------------------------- DNS SESSION START ---------------------------

// Returns an confirmation value (e.g. exit code 0 non-0) and error
func startDNSSession(domain string, fields []string) (string, error) {
	log.Printf("[start session] fields = %#v", fields)

	msgType, err := getFieldMsgType(fields)
	if err != nil {
		return "1", err
	}

	nonce, err := getFieldNonce(fields)
	if err != nil {
		return "1", err
	}

	if !strings.HasPrefix(msgType, "_") {
		return dnsSessionInitSegment(fields)
	}
	log.Printf("Complete session init message received, reassembling ...")

	// TODO: We don't have replay protection against the RSA-encrypt
	// sessionInit messages, but I don't think it's an issue ...
	encryptedSessionInit, err := dnsSessionInitReassemble(nonce)
	if err != nil {
		return "1", err
	}

	publicKeyPEM, privateKeyPEM, err := GetServerRSACertificatePEM("slivers", domain)
	if err != nil {
		log.Printf("Failed to fetch rsa private key")
		return "1", err
	}
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	log.Printf("RSA Fingerprint: %s", fingerprintSHA256(publicKeyBlock))
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	sessionInitData, err := cryptography.RSADecrypt(encryptedSessionInit, privateKey)
	if err != nil {
		log.Printf("Failed to decrypt session init msg")
		return "1", err
	}

	sessionInit := &pb.DNSSessionInit{}
	proto.Unmarshal(sessionInitData, sessionInit)

	sliver := &Sliver{
		ID:        getHiveID(),
		Send:      make(chan pb.Envelope),
		RespMutex: &sync.RWMutex{},
		Resp:      map[string]chan *pb.Envelope{},
	}

	aesKey, _ := cryptography.AESKeyFromBytes(sessionInit.Key)
	sessionID := dnsSessionID()
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
	encodedSessionID := base64.RawStdEncoding.EncodeToString(encryptedSessionID)
	return encodedSessionID, nil
}

// The domain is only a segment of the startDNSSession message, so we just store the data
func dnsSessionInitSegment(fields []string) (string, error) {
	initReassemblerMutex.Lock()
	defer initReassemblerMutex.Unlock()

	nonce, _ := getFieldNonce(fields)
	index, err := getFieldSeq(fields)
	if err != nil {
		return "1", err
	}
	subdata, err := getFieldSubdata(fields)
	if err != nil {
		return "1", err
	}
	if _, ok := (*initReassembler)[nonce]; !ok {
		(*initReassembler)[nonce] = &map[int][]string{}
	}
	if reasm, ok := (*initReassembler)[nonce]; ok {
		(*reasm)[index] = subdata
		return "0", nil
	}
	log.Printf("Invalid nonce (session init segment): %#v", nonce)
	return "1", errors.New("Invalid nonce (session init segment)")
}

// Client should have sent all of the data, attempt to reassemble segments
func dnsSessionInitReassemble(nonce string) ([]byte, error) {
	initReassemblerMutex.Lock()
	defer initReassemblerMutex.Unlock()
	if reasm, ok := (*initReassembler)[nonce]; ok {
		var keys []int
		for k := range *reasm {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		orderedSubdata := []string{}
		for _, k := range keys {
			orderedSubdata = append(orderedSubdata, (*reasm)[k]...)
		}
		log.Printf("Full session init: %#v", strings.Join(orderedSubdata, ""))
		data, err := dnsDecodeString(strings.Join(orderedSubdata, ""))
		if err != nil {
			log.Printf("Failed to decode session init: %v", err)
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("Invalid nonce '%#v' (session init reassembler)", nonce)
}

// --------------------------- DNS SESSION RECV ---------------------------

func getDomainKeyFor(domain string) (string, int) {
	certPEM, _, _ := GetServerRSACertificatePEM("slivers", domain)
	blockID, blockSize := storeSendBlocks(certPEM)
	log.Printf("Encoded cert into %d blocks with ID = %s", blockSize, blockID)
	return blockID, blockSize
}

// --------------------------- DNS SESSION SEND ---------------------------

// Send blocks of data via DNS TXT responses
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
	sequenceNumber := 0
	for index := 0; index < len(data); index += byteBlockSize {
		start := index
		stop := index + byteBlockSize
		if len(data) <= stop {
			stop = len(data) - 1
		}
		seqBuf := new(bytes.Buffer)
		binary.Write(seqBuf, binary.LittleEndian, uint32(sequenceNumber))
		blockBytes := append(seqBuf.Bytes(), data[start:stop]...)
		encoded := "." + base64.RawStdEncoding.EncodeToString(blockBytes)
		log.Printf("Encoded block is %d bytes", len(encoded))
		sendBlock.Data = append(sendBlock.Data, encoded)
		sequenceNumber++
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

// Wrapper around GCMEncrypt & Base32 encode
func sessionDecrypt(sessionKey cryptography.AESKey, data string) ([]byte, error) {
	encryptedData, err := base32.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return cryptography.GCMDecrypt(sessionKey, encryptedData)
}

func fingerprintSHA256(block *pem.Block) string {
	hash := sha256.Sum256(block.Bytes)
	b64hash := base64.StdEncoding.EncodeToString(hash[:])
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
