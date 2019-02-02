package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"
)

func jobsCmd(ctx *grumble.Context) {

	if len(*jobs) < 1 {
		fmt.Println("\n" + Info + "No jobs runnning\n")
		return
	}

	killID := ctx.Flags.Int("kill")
	if killID != -1 {
		if job, ok := (*jobs)[killID]; ok {
			fmt.Printf("\n"+Warn+"Killing job #%d\n\n", killID)
			job.JobCtrl <- true
		} else {
			fmt.Printf("\n"+Warn+"Job ID #%d does not exist\n\n", killID)
		}
	} else {
		printJobs()
	}

}

func printJobs() {
	fmt.Println()
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tName\tProtocol\tPort\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Protocol")),
		strings.Repeat("=", len("Port")))

	for ID, job := range *jobs {
		fmt.Fprintf(table, "%d\t%s\t%s\t%d\t\n", ID, job.Name, job.Protocol, job.Port)
	}
	table.Flush()
	fmt.Println()
}

func startMTLSListenerCmd(ctx *grumble.Context) {

	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))

	fmt.Printf("\n" + Info + "Starting mTLS listener ...\n")
	ID, err := jobStartMTLSListener(server, lport)
	if err == nil {
		fmt.Printf(Info+"Successfully started job #%d\n\n", ID)
	} else {
		fmt.Printf(Warn+"Failed to start job %v\n\n", err)
	}
}

func jobStartMTLSListener(bindIface string, port uint16) (int, error) {

	ln, err := startMutualTLSListener(bindIface, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &Job{
		ID:          getJobID(),
		Name:        "mTLS",
		Description: "mutual tls",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		log.Printf("Stopping mTLS listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		jobMutex.Lock()
		delete(*jobs, job.ID)
		jobMutex.Unlock()

		events <- Event{EventType: "stopped", Job: job}
	}()

	jobMutex.Lock()
	(*jobs)[job.ID] = job
	jobMutex.Unlock()

	return job.ID, nil
}

func startDNSListenerCmd(ctx *grumble.Context) {
	domain := ctx.Flags.String("domain")
	if domain == "" {
		fmt.Printf("\n" + Warn + "Missing parameter, see 'help dns'\n\n")
		return
	}
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}
	ID, _ := jobStartDNSListener(domain)
	fmt.Printf("\n"+Info+"Successfully started job #%d\n\n", ID)
}

func jobStartDNSListener(domain string) (int, error) {

	server := startDNSListener(domain)

	job := &Job{
		ID:          getJobID(),
		Name:        "dns",
		Description: domain,
		Protocol:    "udp",
		Port:        53,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		log.Printf("Stopping DNS listener (%d) ...", job.ID)
		server.Shutdown()

		jobMutex.Lock()
		delete(*jobs, job.ID)
		jobMutex.Unlock()

		events <- Event{EventType: "stopped", Job: job}
	}()

	jobMutex.Lock()
	(*jobs)[job.ID] = job
	jobMutex.Unlock()

	// There is no way to call ListenAndServe() without blocking
	// but we also need to check the error in the case the server
	// fails to start at all, so we setup all the Job mechanics
	// then kick off the server and if it fails we kill the job
	// ourselves.
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("DNS listener error %v", err)
			job.JobCtrl <- true
		}
	}()

	return job.ID, nil
}
