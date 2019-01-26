package main

import (
	"log"
	"net"

	"github.com/miekg/dns"
)

func startDNSListener(domain string) *dns.Server {

	log.Printf("Starting DNS listener for '%s' ...", domain)

	dns.HandleFunc(".", func(writer dns.ResponseWriter, req *dns.Msg) {
		handleDNSRequest(domain, writer, req)
	})

	server := &dns.Server{Addr: ":53", Net: "udp"}
	return server
}

func handleDNSRequest(domain string, writer dns.ResponseWriter, req *dns.Msg) {

	log.Printf("Parsing incoming DNS request")

	if len(req.Question) < 1 {
		log.Printf("No questions in DNS request")
		return
	}

	if !dns.IsSubDomain(domain, req.Question[0].Name) {
		log.Printf("Ignoring DNS req, '%s' is not a child of '%s'", req.Question[0].Name, domain)
		return
	}

	resp := &dns.Msg{}
	switch req.Question[0].Qtype {
	case dns.TypeA:
		resp = handleA(req)
	case dns.TypeTXT:
		resp = handleTXT(req)
	default:
	}

	log.Printf("\n%v\n", resp.String())
	writer.WriteMsg(resp)
}

func handleTXT(req *dns.Msg) *dns.Msg {
	q := req.Question[0]
	resp := new(dns.Msg)
	resp.SetReply(req)

	txt := &dns.TXT{
		Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: []string{"this is a test 123"},
	}
	resp.Answer = append(resp.Answer, txt)

	return resp
}

func handleA(req *dns.Msg) *dns.Msg {
	q := req.Question[0]
	resp := new(dns.Msg)
	resp.SetReply(req)

	a := &dns.A{
		Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		A:   net.IP([]byte{0, 0, 0, 0}),
	}

	resp.Answer = append(resp.Answer, a)
	return resp
}
