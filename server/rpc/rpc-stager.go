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
	"fmt"
	"net"
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
)

// StartTCPStagerListener starts a TCP stager listener
func (rpc *Server) StartTCPStagerListener(ctx context.Context, req *clientpb.StagerListenerReq) (*clientpb.StagerListener, error) {
	host := req.GetHost()
	if !checkInterface(req.GetHost()) {
		host = "0.0.0.0"
	}
	jobID, err := jobStartTCPStagerListener(host, uint16(req.GetPort()), req.GetData())
	return &clientpb.StagerListener{JobID: uint32(jobID)}, err
}

// StartHTTPStagerListener starts a HTTP(S) stager listener
func (rpc *Server) StartHTTPStagerListener(ctx context.Context, req *clientpb.StagerListenerReq) (*clientpb.StagerListener, error) {
	var secure bool = false
	if req.GetProtocol() == clientpb.StageProtocol_HTTPS {
		secure = true
	}
	host := req.GetHost()
	if !checkInterface(req.GetHost()) {
		host = "0.0.0.0"
	}
	conf := &c2.HTTPServerConfig{
		Addr:   fmt.Sprintf("%s:%d", host, req.Port),
		LPort:  uint16(req.Port),
		Domain: req.Host,
		Secure: secure,
		ACME:   false,
	}
	job, err := jobStartHTTPStagerListener(conf, req.Data)
	return &clientpb.StagerListener{JobID: uint32(job.ID)}, err
}

// jobStartTCPStagerListener - Start a TCP staging payload listener
func jobStartTCPStagerListener(host string, port uint16, shellcode []byte) (int, error) {
	ln, err := c2.StartTCPListener(host, port, shellcode)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
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
		rpcLog.Infof("Stopping TCP listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.Remove(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
		})
	}()

	core.Jobs.Add(job)

	return job.ID, nil
}

// jobStartHTTPStagerListener - Start an HTTP(S) stager payload listener
func jobStartHTTPStagerListener(conf *c2.HTTPServerConfig, data []byte) (*core.Job, error) {
	server, err := c2.StartHTTPSListener(conf)
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
