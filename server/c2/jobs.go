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
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/constants"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"golang.zx2c4.com/wireguard/device"
)

var (
	jobLog = log.NamedLogger("c2", "jobs")
)

// StartMTLSListenerJob - Start an mTLS listener as a job
func StartMTLSListenerJob(mtlsListener *clientpb.MTLSListenerReq) (*core.Job, error) {
	bind := fmt.Sprintf("%s:%d", mtlsListener.Host, mtlsListener.Port)
	ln, err := StartMutualTLSListener(mtlsListener.Host, uint16(mtlsListener.Port))
	if err != nil {
		return nil, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        constants.MtlsStr,
		Description: fmt.Sprintf("mutual tls listener %s", bind),
		Protocol:    constants.TCPListenerStr,
		Port:        uint16(mtlsListener.Port),
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

// StartWGListenerJob - Start a WireGuard listener as a job
func StartWGListenerJob(wgListener *clientpb.WGListenerReq) (*core.Job, error) {
	ln, dev, _, err := StartWGListener(uint16(wgListener.Port), uint16(wgListener.NPort), uint16(wgListener.KeyPort))
	if err != nil {
		return nil, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        constants.WGStr,
		Description: fmt.Sprintf("wg listener port: %d", wgListener.Port),
		Protocol:    constants.UDPListenerStr,
		Port:        uint16(wgListener.Port),
		JobCtrl:     make(chan bool),
	}

	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)

	// Listener for new keys
	go func(dev *device.Device) {
		for event := range core.EventBroker.Subscribe() {
			switch event.EventType {

			case consts.WireGuardNewPeer:
				buf := bytes.NewBuffer(nil)
				fmt.Fprintf(buf, "%s", event.Data)
				if err := dev.IpcSetOperation(bufio.NewReader(buf)); err != nil {
					jobLog.Errorf("Failed to update Wireguard Config %s", err)
					continue
				}
			}
		}
	}(dev)

	go func() {
		<-job.JobCtrl
		jobLog.Infof("Stopping wg listener (%d) ...", job.ID)
		ticker.Stop()

		err = ln.Close() // Kills listener GoRoutines in StartWGListener()
		if err != nil {
			jobLog.Fatal("Error closing listener", err)
		}
		err = dev.Down() // Kill wg tunnel
		if err != nil {
			jobLog.Fatal("Error closing wg tunnel", err)
		}
		core.Jobs.Remove(job)
		done <- true
	}()
	core.Jobs.Add(job)

	return job, nil
}

// StartDNSListenerJob - Start a DNS listener as a job
func StartDNSListenerJob(dnsListener *clientpb.DNSListenerReq) (*core.Job, error) {
	server := StartDNSListener(dnsListener.Host, uint16(dnsListener.Port), dnsListener.Domains, dnsListener.Canaries, dnsListener.EnforceOTP)
	description := fmt.Sprintf("%s (canaries %v)", strings.Join(dnsListener.Domains, " "), dnsListener.Canaries)
	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        constants.DnsStr,
		Description: description,
		Protocol:    constants.UDPListenerStr,
		Port:        uint16(dnsListener.Port),
		JobCtrl:     make(chan bool),
		Domains:     dnsListener.Domains,
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

// StartHTTPListenerJob - Start a HTTP listener as a job
func StartHTTPListenerJob(req *clientpb.HTTPListenerReq) (*core.Job, error) {
	server, err := StartHTTPListener(req)
	if err != nil {
		return nil, err
	}
	name := constants.HttpStr
	if req.Secure {
		name = constants.HttpsStr
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        name,
		Description: fmt.Sprintf("%s for domain %s", name, req.Domain),
		Protocol:    constants.TCPListenerStr,
		Port:        uint16(req.Port),
		JobCtrl:     make(chan bool),
		Domains:     []string{req.Domain},
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
		if server.ServerConf.Secure {
			if server.ServerConf.ACME {
				err = server.HTTPServer.ListenAndServeTLS("", "") // ACME manager pulls the certs under the hood
			} else {
				err = listenAndServeTLS(server.HTTPServer, req.Cert, req.Key)
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

// StartTCPStagerListenerJob - Start a TCP staging payload listener
func StartTCPStagerListenerJob(host string, port uint16, profileName string, shellcode []byte) (*core.Job, error) {
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
		ProfileName: profileName,
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

// Fuck'in Go - https://stackoverflow.com/questions/30815244/golang-https-server-passing-certfile-and-kyefile-in-terms-of-byte-array
// basically the same as server.ListenAndServerTLS() but we can pass in byte slices instead of file paths
func listenAndServeTLS(srv *http.Server, certPEMBlock, keyPEMBlock []byte) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		config = srv.TLSConfig
	}
	if certs.TLSKeyLogger != nil {
		config.KeyLogWriter = certs.TLSKeyLogger
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
