package rpc

import (
	"context"
	"fmt"
	"log"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	"sliver/server/assets"
	"sliver/server/c2"
	"sliver/server/certs"
	"sliver/server/core"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

func rpcJobs(_ []byte, resp RPCResponse) {
	jobs := &pb.Jobs{
		Active: []*pb.Job{},
	}
	for _, job := range *core.Jobs.Active {
		jobs.Active = append(jobs.Active, &pb.Job{
			ID:          int32(job.ID),
			Name:        job.Name,
			Description: job.Description,
			Protocol:    job.Protocol,
			Port:        int32(job.Port),
		})
	}
	data, err := proto.Marshal(jobs)
	if err != nil {
		log.Printf("Error encoding rpc response %v", err)
		resp([]byte{}, err)
		return
	}
	resp(data, err)
}

func rpcStartMTLSListener(data []byte, resp RPCResponse) {
	mtlsReq := &pb.MTLSReq{}
	err := proto.Unmarshal(data, mtlsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	jobID, err := jobStartMTLSListener(mtlsReq.Server, uint16(mtlsReq.LPort))
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, err = proto.Marshal(&pb.MTLS{JobID: int32(jobID)})
	resp(data, err)
}

func jobStartMTLSListener(bindIface string, port uint16) (int, error) {

	ln, err := c2.StartMutualTLSListener(bindIface, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.GetJobID(),
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

		core.Jobs.RemoveJob(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.AddJob(job)

	return job.ID, nil
}

func rpcStartDNSListener(data []byte, resp RPCResponse) {
	dnsReq := &pb.DNSReq{}
	err := proto.Unmarshal(data, dnsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	jobID, err := jobStartDNSListener(dnsReq.Domain)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, err = proto.Marshal(&pb.DNS{JobID: int32(jobID)})
	resp(data, err)
}

func jobStartDNSListener(domain string) (int, error) {
	rootDir := assets.GetRootAppDir()
	certs.GetServerRSACertificatePEM(rootDir, "slivers", domain, true)
	server := c2.StartDNSListener(domain)

	job := &core.Job{
		ID:          core.GetJobID(),
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

		core.Jobs.RemoveJob(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.AddJob(job)

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

func rpcStartHTTPSListener(data []byte, resp RPCResponse) {
	httpReq := &pb.HTTPReq{}
	err := proto.Unmarshal(data, httpReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	conf := &c2.HTTPServerConfig{
		Addr:     fmt.Sprintf("%s:%d", httpReq.Iface, httpReq.LPort),
		LPort:    uint16(httpReq.LPort),
		Secure:   true,
		Domain:   httpReq.Domain,
		CertPath: "", // TODO: Get certs
		KeyPath:  "",
	}
	job := jobStartHTTPListener(conf)

	data, err = proto.Marshal(&pb.HTTP{JobID: int32(job.ID)})
	resp(data, err)
}

func rpcStartHTTPListener(data []byte, resp RPCResponse) {
	httpReq := &pb.HTTPReq{}
	err := proto.Unmarshal(data, httpReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	conf := &c2.HTTPServerConfig{
		Addr:   fmt.Sprintf("%s:%d", httpReq.Iface, httpReq.LPort),
		LPort:  uint16(httpReq.LPort),
		Domain: httpReq.Domain,
		Secure: false,
	}
	job := jobStartHTTPListener(conf)

	data, err = proto.Marshal(&pb.HTTP{JobID: int32(job.ID)})
	resp(data, err)
}

func jobStartHTTPListener(conf *c2.HTTPServerConfig) *core.Job {
	server := c2.StartHTTPSListener(conf)
	name := "http"
	if conf.Secure {
		name = "https"
	}

	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        name,
		Description: fmt.Sprintf("%s for domain %s", name, conf.Domain),
		Protocol:    "tcp",
		Port:        uint16(conf.LPort),
		JobCtrl:     make(chan bool),
	}
	core.Jobs.AddJob(job)

	cleanup := func(err error) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		core.Jobs.RemoveJob(job)
		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
			Err:       err,
		})
	}
	once := &sync.Once{}

	go func() {
		var err error
		if conf.Secure {
			err = server.ListenAndServeTLS(conf.CertPath, conf.KeyPath)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil {
			log.Printf("%s listener error %v", name, err)
			once.Do(func() { cleanup(err) })
			job.JobCtrl <- true // Cleanup other goroutine
		}
	}()

	go func() {
		<-job.JobCtrl
		once.Do(func() { cleanup(nil) })
	}()

	return job
}
