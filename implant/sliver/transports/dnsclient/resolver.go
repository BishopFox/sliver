package dnsclient

import (
	"crypto/rand"
	"encoding/binary"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/miekg/dns"
)

func NewGenericResolver(address string, timeout time.Duration) SystemResolver {
	return &GenericResolver{
		address: address,
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
	log.Printf("[dns] %s->A record of %s?", r.address, domain)
	// {{end}}
	resp, rtt, err := r.localQuery(r.address, domain, dns.TypeA)
	if err != nil {
		return nil, rtt, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		return nil, rtt, err
	}
	records := [][]byte{}
	for _, answer := range resp.Answer {
		if answer.Header().Rrtype == dns.TypeA {
			records = append(records, []byte(answer.String()))
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
