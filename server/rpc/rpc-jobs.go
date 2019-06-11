package rpc

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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcJobs(_ []byte, timeout time.Duration, resp RPCResponse) {
	jobs := &clientpb.Jobs{
		Active: []*clientpb.Job{},
	}
	for _, job := range *core.Jobs.Active {
		jobs.Active = append(jobs.Active, &clientpb.Job{
			ID:          int32(job.ID),
			Name:        job.Name,
			Description: job.Description,
			Protocol:    job.Protocol,
			Port:        int32(job.Port),
		})
	}
	data, err := proto.Marshal(jobs)
	if err != nil {
		rpcLog.Errorf("Error encoding rpc response %v", err)
		resp([]byte{}, err)
		return
	}
	resp(data, err)
}

func rpcJobKill(data []byte, timeout time.Duration, resp RPCResponse) {
	jobKillReq := &clientpb.JobKillReq{}
	err := proto.Unmarshal(data, jobKillReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	job := core.Jobs.Job(int(jobKillReq.ID))
	jobKill := &clientpb.JobKill{ID: int32(job.ID)}
	if job != nil {
		job.JobCtrl <- true
		jobKill.Success = true
	} else {
		jobKill.Success = false
		jobKill.Err = "Invalid Job ID"
	}
	data, err = proto.Marshal(jobKill)
	resp(data, err)
}

func rpcStartMTLSListener(data []byte, timeout time.Duration, resp RPCResponse) {
	mtlsReq := &clientpb.MTLSReq{}
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
	data, err = proto.Marshal(&clientpb.MTLS{JobID: int32(jobID)})
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
		rpcLog.Infof("Stopping mTLS listener (%d) ...", job.ID)
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

func rpcStartDNSListener(data []byte, timeout time.Duration, resp RPCResponse) {
	dnsReq := &clientpb.DNSReq{}
	err := proto.Unmarshal(data, dnsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	jobID, err := jobStartDNSListener(dnsReq.Domains, dnsReq.Canaries)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, err = proto.Marshal(&clientpb.DNS{JobID: int32(jobID)})
	resp(data, err)
}

func jobStartDNSListener(domains []string, canaries bool) (int, error) {

	server := c2.StartDNSListener(domains, canaries)
	description := fmt.Sprintf("%s (canaries %v)", strings.Join(domains, " "), canaries)
	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        "dns",
		Description: description,
		Protocol:    "udp",
		Port:        53,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		rpcLog.Infof("Stopping DNS listener (%d) ...", job.ID)
		server.Shutdown()

		core.Jobs.RemoveJob(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.AddJob(job)

	// There is no way to call DNS's ListenAndServe() without blocking
	// but we also need to check the error in the case the server
	// fails to start at all, so we setup all the Job mechanics
	// then kick off the server and if it fails we kill the job
	// ourselves.
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			rpcLog.Errorf("DNS listener error %v", err)
			job.JobCtrl <- true
		}
	}()

	return job.ID, nil
}

func rpcStartHTTPSListener(data []byte, timeout time.Duration, resp RPCResponse) {
	httpReq := &clientpb.HTTPReq{}
	err := proto.Unmarshal(data, httpReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	conf := &c2.HTTPServerConfig{
		Addr:    fmt.Sprintf("%s:%d", httpReq.Iface, httpReq.LPort),
		LPort:   uint16(httpReq.LPort),
		Secure:  true,
		Domain:  httpReq.Domain,
		Website: httpReq.Website,
		Cert:    httpReq.Cert,
		Key:     httpReq.Key,
		ACME:    httpReq.ACME,
	}
	job := jobStartHTTPListener(conf)

	data, err = proto.Marshal(&clientpb.HTTP{
		JobID: int32(job.ID),
	})
	resp(data, err)
}

func rpcStartHTTPListener(data []byte, timeout time.Duration, resp RPCResponse) {
	httpReq := &clientpb.HTTPReq{}
	err := proto.Unmarshal(data, httpReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	conf := &c2.HTTPServerConfig{
		Addr:    fmt.Sprintf("%s:%d", httpReq.Iface, httpReq.LPort),
		LPort:   uint16(httpReq.LPort),
		Domain:  httpReq.Domain,
		Website: httpReq.Website,
		Secure:  false,
		ACME:    false,
	}
	job := jobStartHTTPListener(conf)
	if job == nil {
		data, _ = proto.Marshal(&clientpb.HTTP{JobID: int32(-1)})
		resp(data, errors.New("Failed to start job"))
	} else {
		data, err = proto.Marshal(&clientpb.HTTP{JobID: int32(job.ID)})
		resp(data, err)
	}

}

func jobStartHTTPListener(conf *c2.HTTPServerConfig) *core.Job {
	server := c2.StartHTTPSListener(conf)
	if server == nil {
		return nil
	}
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
		server.Cleanup()
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
		if server.Conf.Secure {
			if server.Conf.ACME {
				err = server.HTTPServer.ListenAndServeTLS("", "") // ACME manager pulls the certs under the hood
			} else {
				err = listenAndServeTLS(server.HTTPServer, conf.Cert, conf.Key)
			}
		} else {
			err = server.HTTPServer.ListenAndServe()
		}
		if err != nil {
			rpcLog.Errorf("%s listener error %v", name, err)
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

// Fuck'in Go - https://stackoverflow.com/questions/30815244/golang-https-server-passing-certfile-and-kyefile-in-terms-of-byte-array
// basically the same as server.ListenAndServerTLS() but we can pass in byte slices instead of file paths
func listenAndServeTLS(srv *http.Server, certPEMBlock, keyPEMBlock []byte) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return srv.Serve(tlsListener)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
