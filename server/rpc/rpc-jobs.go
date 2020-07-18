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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
)

const (
	defaultMTLSPort  = 4444
	defaultDNSPort   = 53
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
)

var (
	// ErrInvalidPort - Invalid TCP port number
	ErrInvalidPort = errors.New("Invalid listener port")
)

// GetJobs - List jobs
func (rpc *Server) GetJobs(ctx context.Context, _ *commonpb.Empty) (*clientpb.Jobs, error) {
	jobs := &clientpb.Jobs{
		Active: []*clientpb.Job{},
	}
	for _, job := range core.Jobs.All() {
		jobs.Active = append(jobs.Active, &clientpb.Job{
			ID:          uint32(job.ID),
			Name:        job.Name,
			Description: job.Description,
			Protocol:    job.Protocol,
			Port:        uint32(job.Port),
			Domains:     job.Domains,
		})
	}
	return jobs, nil
}

// KillJob - Kill a server-side job
func (rpc *Server) KillJob(ctx context.Context, kill *clientpb.KillJobReq) (*clientpb.KillJob, error) {
	job := core.Jobs.Get(int(kill.ID))
	// killJob := &clientpb.KillJob{ID: uint32(job.ID)}
	killJob := &clientpb.KillJob{}
	var err error = nil
	if job != nil {
		job.JobCtrl <- true
		killJob.ID = uint32(job.ID)
		killJob.Success = true
	} else {
		killJob.Success = false
		err = errors.New("Invalid Job ID")
	}
	return killJob, err
}

// StartMTLSListener - Start an MTLS listener
func (rpc *Server) StartMTLSListener(ctx context.Context, req *clientpb.MTLSListenerReq) (*clientpb.MTLSListener, error) {

	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	listenPort := uint16(defaultMTLSPort)
	if req.Port != 0 {
		listenPort = uint16(req.Port)
	}

	bind := fmt.Sprintf("%s:%d", req.Host, listenPort)
	ln, err := c2.StartMutualTLSListener(req.Host, listenPort)
	if err != nil {
		return nil, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "mtls",
		Description: fmt.Sprintf("mutual tls listener %s", bind),
		Protocol:    "tcp",
		Port:        listenPort,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		rpcLog.Infof("Stopping mTLS listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in StartMutualTLSListener() but NOT connections
		core.Jobs.Remove(job)
	}()
	core.Jobs.Add(job)
	return &clientpb.MTLSListener{JobID: uint32(job.ID)}, nil
}

// StartDNSListener - Start a DNS listener TODO: respect request's Host specification
func (rpc *Server) StartDNSListener(ctx context.Context, req *clientpb.DNSListenerReq) (*clientpb.DNSListener, error) {
	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	listenPort := uint16(defaultDNSPort)
	if req.Port != 0 {
		listenPort = uint16(req.Port)
	}
	jobID, err := jobStartDNSListener(req.Domains, req.Canaries, listenPort)
	if err != nil {
		return nil, err
	}
	return &clientpb.DNSListener{JobID: uint32(jobID)}, nil
}

func jobStartDNSListener(domains []string, canaries bool, listenPort uint16) (int, error) {

	server := c2.StartDNSListener(domains, canaries)
	description := fmt.Sprintf("%s (canaries %v)", strings.Join(domains, " "), canaries)
	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "dns",
		Description: description,
		Protocol:    "udp",
		Port:        listenPort,
		JobCtrl:     make(chan bool),
		Domains:     domains,
	}

	go func() {
		<-job.JobCtrl
		rpcLog.Infof("Stopping DNS listener (%d) ...", job.ID)
		server.Shutdown()
		core.Jobs.Remove(job)
		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)

	// There is no way to call DNS' ListenAndServe() without blocking
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

// StartHTTPSListener - Start an HTTPS listener
func (rpc *Server) StartHTTPSListener(ctx context.Context, req *clientpb.HTTPListenerReq) (*clientpb.HTTPListener, error) {

	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	listenPort := uint16(defaultHTTPSPort)
	if req.Port != 0 {
		listenPort = uint16(req.Port)
	}

	conf := &c2.HTTPServerConfig{
		Addr:    fmt.Sprintf("%s:%d", req.Host, listenPort),
		LPort:   listenPort,
		Secure:  true,
		Domain:  req.Domain,
		Website: req.Website,
		Cert:    req.Cert,
		Key:     req.Key,
		ACME:    req.ACME,
	}
	job, err := jobStartHTTPListener(conf)
	if err != nil {
		return nil, err
	}
	return &clientpb.HTTPListener{JobID: uint32(job.ID)}, nil
}

// StartHTTPListener - Start an HTTP listener
func (rpc *Server) StartHTTPListener(ctx context.Context, req *clientpb.HTTPListenerReq) (*clientpb.HTTPListener, error) {
	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	listenPort := uint16(defaultHTTPPort)
	if req.Port != 0 {
		listenPort = uint16(req.Port)
	}

	conf := &c2.HTTPServerConfig{
		Addr:    fmt.Sprintf("%s:%d", req.Host, listenPort),
		LPort:   listenPort,
		Domain:  req.Domain,
		Website: req.Website,
		Secure:  false,
		ACME:    false,
	}
	job, err := jobStartHTTPListener(conf)
	if err != nil {
		return nil, err
	}
	return &clientpb.HTTPListener{JobID: uint32(job.ID)}, nil
}

func jobStartHTTPListener(conf *c2.HTTPServerConfig) (*core.Job, error) {
	server, err := c2.StartHTTPSListener(conf)
	if err != nil {
		return nil, err
	}
	name := "http"
	if conf.Secure {
		name = "https"
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        name,
		Description: fmt.Sprintf("%s for domain %s", name, conf.Domain),
		Protocol:    "tcp",
		Port:        uint16(conf.LPort),
		JobCtrl:     make(chan bool),
		Domains:     []string{conf.Domain},
	}
	core.Jobs.Add(job)

	cleanup := func(err error) {
		server.Cleanup()
		core.Jobs.Remove(job)
		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
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

	return job, nil
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