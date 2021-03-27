package transports

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
*/

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

const (
	defaultMTLSLPort    = 8888
	defaultHTTPLPort    = 80
	defaultHTTPSLPort   = 443
	defaultDNSLPort     = 53
	defaultTCPPort      = 4444
	defaultTCPPivotPort = 9898

	defaultReconnect = 60
	defaultMaxErrors = 1000

	defaultTimeout = 60
)

// MTLSListener - Start a mTLS listener
type MTLSListener struct {
	Options struct {
		LHost      string `long:"lhost" short:"l" description:"interface address to bind mTLS listener to" default:"localhost"`
		LPort      int    `long:"lport" short:"p" description:"listener TCP listen port" default:"1789"`
		Persistent bool   `long:"persistent" short:"P" description:"make listener persistent across server restarts"`
	} `group:"mTLS listener options"`
}

// Execute - Start a mTLS listener
func (m *MTLSListener) Execute(args []string) (err error) {
	server := m.Options.LHost
	lport := uint16(m.Options.LPort)

	if lport == 0 {
		lport = defaultMTLSLPort
	}

	fmt.Printf(util.Info+"Starting mTLS listener (%s:%d)...\n", m.Options.LHost, m.Options.LPort)
	mtls, err := transport.RPC.StartMTLSListener(context.Background(), &clientpb.MTLSListenerReq{
		Host:       server,
		Port:       uint32(lport),
		Persistent: m.Options.Persistent,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		// Temporary: increase jobs counter
		cctx.Context.Jobs++
		fmt.Printf(util.Info+"Successfully started job #%d\n", mtls.JobID)
	}

	return
}

// DNSListener - Start a DNS listener
type DNSListener struct {
	Options struct {
		Domains    []string `long:"domains" short:"d" description:"comma-separated list of DNS C2 domains to callback" env-delim:"," required:"true"`
		LPort      int      `long:"lport" short:"p" description:"listener UDP listen port"`
		NoCanaries bool     `long:"no-canaries" short:"c" description:"disable DNS canary detection for this listener"`
		Persistent bool     `long:"persistent" short:"P" description:"make listener persistent across server restarts"`
	} `group:"DNS listener options"`
}

// Execute - Start a DNS listener
func (m *DNSListener) Execute(args []string) (err error) {

	// domains := strings.Split(ctx.Flags.String("domains"), ",")
	domains := m.Options.Domains
	for _, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}
	}

	lport := uint16(m.Options.LPort)
	if lport == 0 {
		lport = defaultDNSLPort
	}

	fmt.Printf(util.Info+"Starting DNS listener with parent domain(s) %v ...\n", domains)
	dns, err := transport.RPC.StartDNSListener(context.Background(), &clientpb.DNSListenerReq{
		Domains:    domains,
		Port:       uint32(lport),
		Canaries:   !m.Options.NoCanaries,
		Persistent: m.Options.Persistent,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		// Temporary: increase jobs counter
		cctx.Context.Jobs++
		fmt.Printf(util.Info+"Successfully started job #%d\n", dns.JobID)
	}

	return
}

// HTTPSListener - Start a HTTP(S) listener
type HTTPSListener struct {
	Options struct {
		Domain      string `long:"domain" short:"d" description:"HTTPS C2 domain to callback (conversely, limit responses to specific domain)" required:"true"`
		LPort       int    `long:"lport" short:"p" description:"listener TCP listen port" default:"8443"`
		LetsEncrypt bool   `long:"lets-encrypt" description:"attempt to provision a let's encrypt certificate"`
		Website     string `long:"website" short:"w" description:"website name (see 'websites' command)"`
		Certificate string `long:"certificate" description:"PEM encoded certificate file"`
		PrivateKey  string `long:"key" description:"PEM encoded private key file"`
		Persistent  bool   `long:"persistent" short:"P" description:"make listener persistent across server restarts"`
	} `group:"HTTP(S) listener options"`
}

// Execute - Start a HTTP(S) listener
func (m *HTTPSListener) Execute(args []string) (err error) {
	domain := m.Options.Domain
	website := m.Options.Website
	lport := uint16(m.Options.LPort)
	if lport == 0 {
		lport = uint16(defaultHTTPSLPort)
	}

	cert, key, err := getLocalCertificatePair(m.Options.Certificate, m.Options.PrivateKey)
	if err != nil {
		fmt.Printf("\n"+util.Error+"Failed to load local certificate %v", err)
		return
	}

	fmt.Printf(util.Info+"Starting HTTPS %s:%d listener ...\n", domain, lport)
	https, err := transport.RPC.StartHTTPSListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    website,
		Port:       uint32(lport),
		Secure:     true,
		Cert:       cert,
		Key:        key,
		ACME:       m.Options.LetsEncrypt,
		Persistent: m.Options.Persistent,
	})
	if err != nil {
		fmt.Printf(util.Warn+"%s\n", err)
	} else {
		// Temporary: increase jobs counter
		cctx.Context.Jobs++
		fmt.Printf(util.Info+"Successfully started job #%d\n", https.JobID)
	}

	return
}

func getLocalCertificatePair(certPath, keyPath string) ([]byte, []byte, error) {
	if certPath == "" && keyPath == "" {
		return nil, nil, nil
	}
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// HTTPListener - Start a HTTP listener
type HTTPListener struct {
	Options struct {
		Domain     string `long:"domain" short:"d" description:"HTTP C2 domain to callback (conversely, limit responses to specific domain)" required:"true"`
		LPort      int    `long:"lport" short:"p" description:"listener TCP listen port" default:"8080"`
		Timeout    int    `long:"timeout" short:"t" description:"command timeout in seconds" default:"60"`
		Website    string `long:"website" short:"w" description:"website name (see 'websites' command)"`
		Persistent bool   `long:"persistent" short:"P" description:"make listener persistent across server restarts"`
	} `group:"HTTP listener options"`
}

// Execute - Start a HTTP listener
func (m *HTTPListener) Execute(args []string) (err error) {
	domain := m.Options.Domain
	lport := uint16(m.Options.LPort)
	if lport == 0 {
		lport = uint16(defaultHTTPSLPort)
	}

	fmt.Printf(util.Info+"Starting HTTP %s:%d listener ...\n", domain, lport)
	http, err := transport.RPC.StartHTTPListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    m.Options.Website,
		Port:       uint32(lport),
		Secure:     false,
		Persistent: m.Options.Persistent,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		// Temporary: increase jobs counter
		cctx.Context.Jobs++
		fmt.Printf(util.Info+"Successfully started job #%d\n", http.JobID)
	}

	return
}
