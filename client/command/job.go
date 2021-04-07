package command

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
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	// "github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
	// "github.com/golang/protobuf/proto"
)

func jobs(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ctx.Flags.Int("kill") != -1 {
		killJob(uint32(ctx.Flags.Int("kill")), rpc)
	} else if ctx.Flags.Bool("kill-all") {
		killAllJobs(rpc)
	} else {
		jobs, err := rpc.GetJobs(context.Background(), &commonpb.Empty{})
		if err != nil {
			fmt.Printf(Warn+"%s", err)
			return
		}
		// Convert to a map
		activeJobs := map[uint32]*clientpb.Job{}
		for _, job := range jobs.Active {
			activeJobs[job.ID] = job
		}
		if 0 < len(activeJobs) {
			printJobs(activeJobs)
		} else {
			fmt.Printf(Info + "No active jobs\n")
		}
	}
}

func killAllJobs(rpc rpcpb.SliverRPCClient) {
	jobs, err := rpc.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	for _, job := range jobs.Active {
		killJob(job.ID, rpc)
	}
}

func killJob(jobID uint32, rpc rpcpb.SliverRPCClient) {
	fmt.Printf(Info+"Killing job #%d ...\n", jobID)
	jobKill, err := rpc.KillJob(context.Background(), &clientpb.KillJobReq{
		ID: jobID,
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"Successfully killed job #%d\n", jobKill.ID)
	}
}

func printJobs(jobs map[uint32]*clientpb.Job) {
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tName\tProtocol\tPort\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Protocol")),
		strings.Repeat("=", len("Port")))

	var keys []int
	for _, job := range jobs {
		keys = append(keys, int(job.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	for _, k := range keys {
		job := jobs[uint32(k)]
		fmt.Fprintf(table, "%d\t%s\t%s\t%d\t\n", job.ID, job.Name, job.Protocol, job.Port)
	}
	table.Flush()
}

func startMTLSListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info + "Starting mTLS listener ...\n")
	mtls, err := rpc.StartMTLSListener(context.Background(), &clientpb.MTLSListenerReq{
		Host:       server,
		Port:       uint32(lport),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		fmt.Printf("\n"+Warn+"%s\n", err)
	} else {
		fmt.Printf("\n"+Info+"Successfully started job #%d\n", mtls.JobID)
	}
}

func startWGListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	lport := uint16(ctx.Flags.Int("lport"))
	nport := uint16(ctx.Flags.Int("nport"))
	keyExchangePort := uint16(ctx.Flags.Int("key-port"))

	fmt.Printf(Info + "Starting Wireguard listener ...\n")
	wg, err := rpc.StartWGListener(context.Background(), &clientpb.WGListenerReq{
		Port:       uint32(lport),
		NPort:      uint32(nport),
		KeyPort:    uint32(keyExchangePort),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		fmt.Printf("\n"+Warn+"%s\n", err)
	} else {
		fmt.Printf("\n"+Info+"Successfully started job #%d\n", wg.JobID)
	}
}

func startDNSListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	domains := strings.Split(ctx.Flags.String("domains"), ",")
	for _, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}
	}

	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info+"Starting DNS listener with parent domain(s) %v ...\n", domains)
	dns, err := rpc.StartDNSListener(context.Background(), &clientpb.DNSListenerReq{
		Domains:    domains,
		Port:       uint32(lport),
		Canaries:   !ctx.Flags.Bool("no-canaries"),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		fmt.Printf("\n"+Warn+"%s\n", err)
	} else {
		fmt.Printf("\n"+Info+"Successfully started job #%d\n", dns.JobID)
	}
}

func startHTTPSListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	domain := ctx.Flags.String("domain")
	website := ctx.Flags.String("website")
	lport := uint16(ctx.Flags.Int("lport"))

	cert, key, err := getLocalCertificatePair(ctx)
	if err != nil {
		fmt.Printf("\n"+Warn+"Failed to load local certificate %v", err)
		return
	}

	fmt.Printf(Info+"Starting HTTPS %s:%d listener ...\n", domain, lport)
	https, err := rpc.StartHTTPSListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    website,
		Port:       uint32(lport),
		Secure:     true,
		Cert:       cert,
		Key:        key,
		ACME:       ctx.Flags.Bool("lets-encrypt"),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		fmt.Printf("\n"+Warn+"%s\n", err)
	} else {
		fmt.Printf("\n"+Info+"Successfully started job #%d\n", https.JobID)
	}
}

func getLocalCertificatePair(ctx *grumble.Context) ([]byte, []byte, error) {
	if ctx.Flags.String("cert") == "" && ctx.Flags.String("key") == "" {
		return nil, nil, nil
	}
	cert, err := ioutil.ReadFile(ctx.Flags.String("cert"))
	if err != nil {
		return nil, nil, err
	}
	key, err := ioutil.ReadFile(ctx.Flags.String("key"))
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func startHTTPListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	domain := ctx.Flags.String("domain")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info+"Starting HTTP %s:%d listener ...\n", domain, lport)
	http, err := rpc.StartHTTPListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    ctx.Flags.String("website"),
		Port:       uint32(lport),
		Secure:     false,
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"Successfully started job #%d\n", http.JobID)
	}
}
