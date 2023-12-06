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
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
)

const (
	defaultMTLSPort    = 4444
	defaultWGPort      = 53
	defaultWGNPort     = 8888
	defaultWGKeyExPort = 1337
	defaultDNSPort     = 53
	defaultHTTPPort    = 80
	defaultHTTPSPort   = 443
)

var (
	// ErrInvalidPort - Invalid TCP port number
	ErrInvalidPort = errors.New("invalid listener port")
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
			ProfileName: job.ProfileName,
		})
	}
	return jobs, nil
}

// Restart Jobs - Reload jobs
func (rpc *Server) RestartJobs(ctx context.Context, restartJobReq *clientpb.RestartJobReq) (*commonpb.Empty, error) {
	// reload jobs to include new profile
	for _, jobID := range restartJobReq.JobIDs {
		job := core.Jobs.Get(int(jobID))
		listenerJob, err := db.ListenerByJobID(jobID)
		if err != nil {
			return &commonpb.Empty{}, err
		}
		job.JobCtrl <- true
		job, err = c2.StartHTTPListenerJob(listenerJob.HTTPConf)
		if err != nil {
			return &commonpb.Empty{}, err
		}
		listenerJob.JobID = uint32(job.ID)
		db.UpdateHTTPC2Listener(listenerJob)
	}
	return &commonpb.Empty{}, nil
}

// KillJob - Kill a server-side job
func (rpc *Server) KillJob(ctx context.Context, kill *clientpb.KillJobReq) (*clientpb.KillJob, error) {
	job := core.Jobs.Get(int(kill.ID))
	killJob := &clientpb.KillJob{}
	var err error = nil
	if job != nil {
		job.JobCtrl <- true
		killJob.ID = uint32(job.ID)
		killJob.Success = true
		err = db.DeleteListener(killJob.ID)
		if err != nil {
			return nil, err
		}
	} else {
		killJob.Success = false
		err = errors.New("invalid Job ID")
	}
	return killJob, err
}

// StartMTLSListener - Start an MTLS listener
func (rpc *Server) StartMTLSListener(ctx context.Context, req *clientpb.MTLSListenerReq) (*clientpb.ListenerJob, error) {
	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	if req.Port == 0 {
		req.Port = defaultMTLSPort
	}

	err := PortInUse(req.Port)
	if err != nil {
		return nil, err
	}

	job, err := c2.StartMTLSListenerJob(req)
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:    uint32(job.ID),
		Type:     constants.MtlsStr,
		MTLSConf: req,
	}
	err = db.SaveHTTPC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.ListenerJob{JobID: uint32(job.ID)}, nil
}

// StartWGListener - Start a Wireguard listener
func (rpc *Server) StartWGListener(ctx context.Context, req *clientpb.WGListenerReq) (*clientpb.ListenerJob, error) {

	if 65535 <= req.Port || 65535 <= req.NPort || 65535 <= req.KeyPort {
		return nil, ErrInvalidPort
	}
	if req.Port == 0 {
		req.Port = defaultWGPort
	}

	if req.NPort == 0 {
		req.NPort = defaultWGNPort
	}

	if req.KeyPort == 0 {
		req.KeyPort = defaultWGKeyExPort
	}

	err := PortInUse(req.Port)
	if err != nil {
		return nil, err
	}

	err = PortInUse(req.NPort)
	if err != nil {
		return nil, err
	}

	err = PortInUse(req.KeyPort)
	if err != nil {
		return nil, err
	}

	job, err := c2.StartWGListenerJob(req)
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:  uint32(job.ID),
		Type:   constants.WGStr,
		WGConf: req,
	}
	err = db.SaveHTTPC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.ListenerJob{JobID: uint32(job.ID)}, nil
}

// StartDNSListener - Start a DNS listener TODO: respect request's Host specification
func (rpc *Server) StartDNSListener(ctx context.Context, req *clientpb.DNSListenerReq) (*clientpb.ListenerJob, error) {
	err := PortInUse(req.Port)
	if err != nil {
		return nil, err
	}

	job, err := c2.StartDNSListenerJob(req)
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:   uint32(job.ID),
		Type:    constants.DnsStr,
		DNSConf: req,
	}
	err = db.SaveHTTPC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.ListenerJob{JobID: uint32(job.ID)}, nil
}

// StartHTTPSListener - Start an HTTPS listener
func (rpc *Server) StartHTTPSListener(ctx context.Context, req *clientpb.HTTPListenerReq) (*clientpb.ListenerJob, error) {
	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	if req.Port == 0 {
		req.Port = defaultHTTPSPort
	}

	err := PortInUse(req.Port)
	if err != nil {
		return nil, err
	}

	job, err := c2.StartHTTPListenerJob(req)
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:    uint32(job.ID),
		Type:     constants.HttpsStr,
		HTTPConf: req,
	}
	err = db.SaveHTTPC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.ListenerJob{JobID: uint32(job.ID)}, nil
}

// StartHTTPListener - Start an HTTP listener
func (rpc *Server) StartHTTPListener(ctx context.Context, req *clientpb.HTTPListenerReq) (*clientpb.ListenerJob, error) {
	if 65535 <= req.Port {
		return nil, ErrInvalidPort
	}
	if req.Port == 0 {
		req.Port = defaultHTTPPort
	}

	err := PortInUse(req.Port)
	if err != nil {
		return nil, err
	}

	job, err := c2.StartHTTPListenerJob(req)
	if err != nil {
		return nil, err
	}

	listenerJob := &clientpb.ListenerJob{
		JobID:    uint32(job.ID),
		Type:     constants.HttpStr,
		HTTPConf: req,
	}
	err = db.SaveHTTPC2Listener(listenerJob)
	if err != nil {
		return nil, err
	}

	return &clientpb.ListenerJob{JobID: uint32(job.ID)}, nil
}

func PortInUse(newPort uint32) error {
	listenerJobs, err := db.ListenerJobs()
	if err != nil {
		return err
	}
	var port uint32
	for _, job := range listenerJobs {
		listener, err := db.ListenerByJobID(job.JobID)
		if err != nil {
			return err
		}
		switch job.Type {
		case "http":
			port = listener.HTTPConf.Port
		case "https":
			port = listener.HTTPConf.Port
		case "mtls":
			port = listener.MTLSConf.Port
		case "dns":
			port = listener.DNSConf.Port
		case "wg":
			port = listener.WGConf.Port
		case "multiplayer":
			port = listener.MultiConf.Port
		}

		if port == newPort {
			return errors.New(fmt.Sprintf("port %d is in use", port))
		}
	}
	return nil
}
