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

	DNS command and control implementation:

	1. Implant sends TOTP encoded message to DNS server, server checks validity
	2. DNS server responds with the "DNS Session ID" which is just some random value
	3. Requests with valid DNS session IDs enable the server to respond with CRC32 responses

*/

import (
	secureRand "crypto/rand"
	"errors"
	"net"
	"unicode"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"

	"encoding/binary"

	"fmt"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
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

	ErrInvalidMsg = errors.New("invalid dns message")
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

// AddData - Add data to the block
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

// GetData - Get a data block at index
func (b *Block) GetData(index int) ([]byte, error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if len(b.data) < index+1 {
		return nil, errors.New("Data index out of bounds")
	}
	return b.data[index], nil
}

// Reassemble - Reassemble a block of data
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
	Key     cryptography.CipherContext
	replay  *sync.Map // Sessions are mutex 'd
}

// --------------------------- DNS SERVER ---------------------------

// StartDNSListener - Start a DNS listener
func StartDNSListener(bindIface string, lport uint16, domains []string, canaries bool) *dns.Server {
	// StartPivotListener()
	dnsLog.Infof("Starting DNS listener for %v (canaries: %v) ...", domains, canaries)
	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		// req.Question[0].Name = strings.ToLower(req.Question[0].Name)
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
	dnsLog.Infof("'%s' is NOT subdomain of any c2 domain %v", reqDomain, domains)
	return false, ""
}

// The query is C2, pass to the appropriate record handler this is done
// so the record handler can encode the response based on the type of
// record that was requested
func handleC2(domain string, req *dns.Msg) *dns.Msg {
	subdomain := req.Question[0].Name[:len(req.Question[0].Name)-len(domain)]
	dnsLog.Debugf("processing req for subdomain = %s", subdomain)
	switch req.Question[0].Qtype {
	case dns.TypeTXT:
		return handleTXT(domain, subdomain, req)
	case dns.TypeA:
		// return handleA(domain, subdomain, req)
	default:
	}
	return nil
}

// Parse subdomain as data
func decodeSubdata(subdomain string) (*dnspb.DNSMessage, error) {
	subdata := strings.Join(strings.Split(subdomain, "."), "")
	dnsLog.Debugf("subdata = %s", subdata)
	encoders := determineLikelyEncoders(subdata)
	for _, encoder := range encoders {
		data, err := encoder.Decode([]byte(subdata))
		if err == nil {
			msg := &dnspb.DNSMessage{}
			err = proto.Unmarshal(data, msg)
			if err == nil {
				return msg, nil
			}
		}
		dnsLog.Debugf("failed to decode subdata with %#v (%s)", encoder, err)
	}
	return nil, ErrInvalidMsg
}

// Returns the most likely -> least likely encoders, if decoding fails fallback to
// the next encoder until we run out of options.
func determineLikelyEncoders(subdata string) []encoders.Encoder {
	// If the string contains i, l, o, s is must be base62 these
	// chars are missing from the base32 alphabet
	if strings.ContainsAny(subdata, "silo") {
		return []encoders.Encoder{encoders.Base62{}, encoders.Base32{}}
	}
	for _, char := range subdata {
		if unicode.IsUpper(char) {
			return []encoders.Encoder{encoders.Base62{}, encoders.Base32{}}
		}
	}
	return []encoders.Encoder{encoders.Base32{}, encoders.Base62{}}
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
	}

	return resp
}

// Ensure non-zeros with various bitmasks
func randomIP() net.IP {
	randBuf := make([]byte, 4)
	secureRand.Read(randBuf)
	return net.IPv4(randBuf[0]|0x10, randBuf[1]|0x10, randBuf[2]|0x1, randBuf[3]|0x10)
}

// ---------------------------
// C2 Record Handlers
// ---------------------------
func handleTXT(domain string, subdomain string, req *dns.Msg) *dns.Msg {
	q := req.Question[0]
	resp := new(dns.Msg)
	resp.SetReply(req)

	txtRecords := []string{}
	msg, err := decodeSubdata(subdomain)
	if err == nil {
		dnsLog.Infof("dns msg '%v'", msg)
	}
	txt := &dns.TXT{
		Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: txtRecords,
	}
	resp.Answer = append(resp.Answer, txt)
	return resp
}

// Returns an confirmation value (e.g. exit code 0 non-0) and error
func startDNSSession(msgID uint32, domain string) ([]string, error) {
	dnsLog.Infof("Complete session init message received, reassembling ...")

	return nil, nil
}

// DNSSessionIDs are public and identify a stream of DNS requests
// the lower 8 bits are the message ID so we chop them off
func dnsSessionID() uint32 {
	randBuf := make([]byte, 4)
	secureRand.Read(randBuf)
	dnsSessionID := binary.LittleEndian.Uint32(randBuf)
	return dnsSessionID & 0xffffff00
}
