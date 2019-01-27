package main

import (
	"encoding/base64"
	"encoding/binary"
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
	nonceStdSize       = 4

	maxFetchTxts = 200 // How many txts to request at a time
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")
)

// LookupDomainKey - Attempt to get the server's RSA public key
func LookupDomainKey(selector string, parentDomain string) ([]byte, error) {
	selector = strings.ToLower(selector)
	nonce := DNSNonce(nonceStdSize)
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

// DNSNonce - Generate a nonce of a given size
func DNSNonce(size int) string {
	insecureRand.Seed(time.Now().UnixNano())
	nonce := []rune{}
	for i := 0; i < size; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		nonce = append(nonce, dnsCharSet[index])
	}
	return string(nonce)
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
	return msg, nil
}

func fetchRecvBlock(parentDomain string, reasm *BlockReassembler, start int, stop int, wg *sync.WaitGroup) {
	defer wg.Done()
	nonce := DNSNonce(nonceStdSize)
	subdomain := fmt.Sprintf("_%s.%d.%d.%s._b.%s", nonce, start, stop, reasm.ID, parentDomain)
	log.Printf("[dns] fetch -> %s", subdomain)
	txts, err := net.LookupTXT(subdomain)
	log.Printf("[dns] fetched %d txt record(s)", len(txts))
	if err != nil {
		log.Printf("Failed to fetch blocks %v", err)
	}
	for _, txt := range txts {
		log.Printf("Decoding TXT record: %s", txt)
		rawBlock, err := base64.RawStdEncoding.DecodeString(txt)
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
