package main

import (
	"encoding/base64"
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
	Data  string
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

	fetchTxts = 200 // How many txts to request at a time
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
	insecureRand.Seed(time.Now().Unix())
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
	data := make([]string, reasm.Size)
	for index := 0; index < reasm.Size; index += fetchTxts {
		wg.Add(1)
		go fetchRecvBlock(parentDomain, reasm, index, index+fetchTxts, &wg)
	}
	go func() {
		for block := range reasm.Recv {
			data[block.Index] = block.Data
		}
	}()
	wg.Wait()
	close(reasm.Recv)

	return base64.RawStdEncoding.DecodeString(strings.Join(data, ""))
}

func fetchRecvBlock(parentDomain string, reasm *BlockReassembler, start int, stop int, wg *sync.WaitGroup) {
	defer wg.Done()
	nonce := DNSNonce(nonceStdSize)
	subdomain := fmt.Sprintf("_%s.%d.%d.%s._b.%s", nonce, start, stop, reasm.ID, parentDomain)
	txts, err := net.LookupTXT(subdomain)
	if err != nil {
		log.Printf("Failed to fetch blocks %v", err)
	}
	for _, txt := range txts {
		fields := strings.Split(txt, ".")
		if len(fields) == 2 {
			index, err := strconv.Atoi(fields[0])
			if err != nil {
				log.Printf("Invalid index in TXT record: %s", txt)
				continue
			}
			reasm.Recv <- RecvBlock{
				Index: index,
				Data:  fields[1],
			}
		} else {
			log.Printf("Invalid block TXT record: %s", txt)
		}

	}
}
