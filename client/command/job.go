package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func jobs(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgJobs,
		Data: []byte{},
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	jobs := &clientpb.Jobs{}
	proto.Unmarshal(resp.Data, jobs)

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
	domain := ctx.Flags.String("domain")
	if domain == "" {
		fmt.Printf(Warn + "Missing parameter, see 'help dns'\n")
		return
	}
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}

	fmt.Printf(Info+"Starting DNS listener with parent domain '%s' ...\n", domain)

	data, _ := proto.Marshal(&clientpb.DNSReq{Domain: domain})
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
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info+"Starting HTTPS %s:%d listener ...\n", domain, lport)
	data, _ := proto.Marshal(&clientpb.HTTPReq{
		Domain: domain,
		LPort:  int32(lport),
		Secure: true,
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

func startHTTPListener(ctx *grumble.Context, rpc RPCServer) {
	domain := ctx.Flags.String("domain")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf(Info+"Starting HTTP %s:%d listener ...\n", domain, lport)
	data, _ := proto.Marshal(&clientpb.HTTPReq{
		Domain: domain,
		LPort:  int32(lport),
		Secure: false,
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
