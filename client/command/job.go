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
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func jobs(ctx *grumble.Context, rpc RPCServer) {
	if ctx.Flags.Int("kill") != -1 {
		killJob(int32(ctx.Flags.Int("kill")), rpc)
	} else if ctx.Flags.Bool("kill-all") {
		killAllJobs(rpc)
	} else {
		jobs := getJobs(rpc)
		if jobs == nil {
			return
		}
		activeJobs := map[int32]*clientpb.Job{}
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

func killAllJobs(rpc RPCServer) {
	jobs := getJobs(rpc)
	if jobs == nil {
		return
	}
	for _, job := range jobs.Active {
		killJob(job.ID, rpc)
	}
}

func killJob(jobID int32, rpc RPCServer) {
	fmt.Printf(Info+"Killing job #%d ...\n", jobID)
	data, _ := proto.Marshal(&clientpb.JobKillReq{ID: jobID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgJobKill,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}
	jobKill := &clientpb.JobKill{}
	proto.Unmarshal(resp.Data, jobKill)

	if jobKill.Success {
		fmt.Printf(Info+"Successfully killed job #%d\n", jobKill.ID)
	} else {
		fmt.Printf(Warn+"Failed to kill job #%d, %s\n", jobKill.ID, jobKill.Err)
	}
}

func getJobs(rpc RPCServer) *clientpb.Jobs {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgJobs,
		Data: []byte{},
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return nil
	}
	jobs := &clientpb.Jobs{}
	proto.Unmarshal(resp.Data, jobs)
	return jobs
}

func printJobs(jobs map[int32]*clientpb.Job) {
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
		job := jobs[int32(k)]
		fmt.Fprintf(table, "%d\t%s\t%s\t%d\t\n", job.ID, job.Name, job.Protocol, job.Port)
	}
	table.Flush()
}

func startMTLSListener(ctx *grumble.Context, rpc RPCServer) {
	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info + "Starting mTLS listener ...\n")
	data, _ := proto.Marshal(&clientpb.MTLSReq{
		Server: server,
		LPort:  int32(lport),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgMtls,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Failed to start job %s\n", resp.Err)
		return
	}
	mtls := &clientpb.MTLS{}
	proto.Unmarshal(resp.Data, mtls)
	fmt.Printf(Info+"Successfully started job #%d\n", mtls.JobID)
}

func startDNSListener(ctx *grumble.Context, rpc RPCServer) {

	domains := strings.Split(ctx.Flags.String("domains"), ",")
	for _, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}
	}

	fmt.Printf(Info+"Starting DNS listener with parent domain(s) %v ...\n", domains)

	data, _ := proto.Marshal(&clientpb.DNSReq{
		Domains:  domains,
		Canaries: !ctx.Flags.Bool("no-canaries"),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgDns,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Failed to start job %s\n", resp.Err)
		return
	}
	dns := &clientpb.DNS{}
	proto.Unmarshal(resp.Data, dns)

	fmt.Printf(Info+"Successfully started job #%d\n", dns.JobID)
}

func startHTTPSListener(ctx *grumble.Context, rpc RPCServer) {
	domain := ctx.Flags.String("domain")
	website := ctx.Flags.String("website")
	lport := uint16(ctx.Flags.Int("lport"))

	cert, key, err := getLocalCertificatePair(ctx)
	if err != nil {
		fmt.Printf(Warn+"Failed to load local certificate %v", err)
		return
	}

	fmt.Printf(Info+"Starting HTTPS %s:%d listener ...\n", domain, lport)
	data, _ := proto.Marshal(&clientpb.HTTPReq{
		Domain:  domain,
		Website: website,
		LPort:   int32(lport),
		Secure:  true,
		Cert:    cert,
		Key:     key,
		ACME:    ctx.Flags.Bool("lets-encrypt"),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgHttps,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Failed to start job %s\n", resp.Err)
		return
	}
	httpJob := &clientpb.HTTP{}
	proto.Unmarshal(resp.Data, httpJob)
	fmt.Printf(Info+"Successfully started job #%d\n", httpJob.JobID)
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

func startHTTPListener(ctx *grumble.Context, rpc RPCServer) {
	domain := ctx.Flags.String("domain")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info+"Starting HTTP %s:%d listener ...\n", domain, lport)
	data, _ := proto.Marshal(&clientpb.HTTPReq{
		Domain:  domain,
		Website: ctx.Flags.String("website"),
		LPort:   int32(lport),
		Secure:  false,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgHttp,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Failed to start job %s\n", resp.Err)
		return
	}
	httpJob := &clientpb.HTTP{}
	proto.Unmarshal(resp.Data, httpJob)
	fmt.Printf(Info+"Successfully started job #%d\n", httpJob.JobID)
}
