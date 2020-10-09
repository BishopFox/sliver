package c2

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
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var (
	jobLog = log.NamedLogger("c2", "jobs")
)

func StartMTLSListenerJob(host string, listenPort uint16) (*core.Job, error) {
	bind := fmt.Sprintf("%s:%d", host, listenPort)
	ln, err := StartMutualTLSListener(host, listenPort)
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
		jobLog.Infof("Stopping mTLS listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in StartMutualTLSListener() but NOT connections
		core.Jobs.Remove(job)
	}()
	core.Jobs.Add(job)

	return job, nil
}

func StartDNSListenerJob(domains []string, canaries bool, listenPort uint16) (*core.Job, error) {
	server := StartDNSListener(domains, canaries)
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
		jobLog.Infof("Stopping DNS listener (%d) ...", job.ID)
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
			jobLog.Errorf("DNS listener error %v", err)
			job.JobCtrl <- true
		}
	}()

	return job, nil
}

func StartHTTPListenerJob(conf *HTTPServerConfig) (*core.Job, error) {
	server, err := StartHTTPSListener(conf)
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
			jobLog.Errorf("%s listener error %v", name, err)
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

// Start a TCP staging payload listener
func StartTCPStagerListenerJob(host string, port uint16, shellcode []byte) (*core.Job, error) {
	ln, err := StartTCPListener(host, port, shellcode)
	if err != nil {
		return nil, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "TCP",
		Description: "Raw TCP listener (stager only)",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		jobLog.Infof("Stopping TCP listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.Remove(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)

	return job, nil
}

// StartHTTPStagerListener - Start an HTTP(S) stager payload listener
func StartHTTPStagerListenerJob(conf *HTTPServerConfig, data []byte) (*core.Job, error) {
	server, err := StartHTTPSListener(conf)
	if err != nil {
		return nil, err
	}
	name := "http"
	if conf.Secure {
		name = "https"
	}
	server.SliverStage = data
	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        name,
		Description: fmt.Sprintf("Stager handler %s for domain %s", name, conf.Domain),
		Protocol:    "tcp",
		Port:        uint16(conf.LPort),
		JobCtrl:     make(chan bool),
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
			jobLog.Errorf("%s listener error %v", name, err)
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

// Start persistent jobs
func StartPersistentJobs(cfg *configs.ServerConfig) error {
	if cfg.Jobs == nil {
		return nil
	}

	for _, j := range cfg.Jobs.MTLS {
		job, err := StartMTLSListenerJob(j.Host, j.Port)
		if err != nil {
			return err
		}
		job.PersistentID = j.JobID
	}

	for _, j := range cfg.Jobs.DNS {
		job, err := StartDNSListenerJob(j.Domains, j.Canaries, j.Port)
		if err != nil {
			return err
		}
		job.PersistentID = j.JobID
	}

	for _, j := range cfg.Jobs.HTTP {
		cfg := &HTTPServerConfig{
			Addr:    fmt.Sprintf("%s:%d", j.Host, j.Port),
			LPort:   j.Port,
			Secure:  j.Secure,
			Domain:  j.Domain,
			Website: j.Website,
			Cert:    j.Cert,
			Key:     j.Key,
			ACME:    j.ACME,
		}
		job, err := StartHTTPListenerJob(cfg)
		if err != nil {
			return err
		}
		job.PersistentID = j.JobID
	}

	return nil
}

// checkInterface verifies if an IP address
// is attached to an existing network interface
func checkInterface(a string) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, i := range interfaces {
		addresses, err := i.Addrs()
		if err != nil {
			return false
		}
		for _, netAddr := range addresses {
			addr, err := net.ResolveTCPAddr("tcp", netAddr.String())
			if err != nil {
				return false
			}
			if addr.IP.String() == a {
				return true
			}
		}
	}
	return false
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
