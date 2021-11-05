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
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/miekg/dns"
)

func NewGenericResolver(address string, port string, timeout time.Duration) DNSResolver {
	return &GenericResolver{
		address: address + ":" + port,
		resolver: &dns.Client{
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
		},
	}
}

type GenericResolver struct {
	address  string
	resolver *dns.Client
}

func (r *GenericResolver) Address() string {
	return r.address
}

func (r *GenericResolver) A(domain string) ([][]byte, time.Duration, error) {
	// {{if .Config.Debug}}
	log.Printf("[dns] %s->A record of %s ?", r.address, domain)
	// {{end}}
	resp, rtt, err := r.localQuery(r.address, domain, dns.TypeA)
	if err != nil {
		return nil, rtt, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		// {{if .Config.Debug}}
		log.Printf("[dns] error response status: %v", resp.Rcode)
		// {{end}}
		return nil, rtt, err
	}
	records := [][]byte{}
	for _, answer := range resp.Answer {
		switch answer := answer.(type) {
		case *dns.A:
			// {{if .Config.Debug}}
			log.Printf("[dns] answer: %v (%s)", answer.A, answer.A.String())
			// {{end}}
			records = append(records, []byte(answer.A))
		}
	}
	return records, rtt, err
}

func (r *GenericResolver) localQuery(resolver string, qName string, qType uint16) (*dns.Msg, time.Duration, error) {
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               headerID(),
			RecursionDesired: true,
			Opcode:           dns.OpcodeQuery,
		},
	}
	msg.SetQuestion(qName, qType)
	resp, rtt, err := r.resolver.Exchange(msg, resolver)
	// {{if .Config.Debug}}
	log.Printf("[dns] rtt->%s %s (err: %v)", resolver, rtt, err)
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
