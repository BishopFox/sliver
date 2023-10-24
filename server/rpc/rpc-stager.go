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
	"os"
	"path/filepath"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
)

// StartTCPStagerListener starts a TCP stager listener
func (rpc *Server) StartTCPStagerListener(ctx context.Context, req *clientpb.StagerListenerReq) (*clientpb.StagerListener, error) {
	host := req.GetHost()
	if !checkInterface(req.GetHost()) {
		host = "0.0.0.0"
	}
	job, err := c2.StartTCPStagerListenerJob(host, uint16(req.GetPort()), req.ProfileName, req.GetData())
	if err != nil {
		return nil, err
	}
	return &clientpb.StagerListener{JobID: uint32(job.ID)}, nil
}

// StartHTTPStagerListener starts a HTTP(S) stager listener
func (rpc *Server) StartHTTPStagerListener(ctx context.Context, req *clientpb.StagerListenerReq) (*clientpb.StagerListener, error) {
	host := req.GetHost()
	if !checkInterface(req.GetHost()) {
		host = "0.0.0.0"
	}

	conf := &clientpb.HTTPListenerReq{
		Host:   host,
		Port:   req.Port,
		Domain: req.Host,
		Secure: false,
	}
	if req.GetProtocol() == clientpb.StageProtocol_HTTPS {
		conf.Secure = true
		conf.Key = req.Key
		conf.Cert = req.Cert
		conf.ACME = req.ACME
	}
	job, err := c2.StartHTTPStagerListenerJob(conf, req.Data)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, fmt.Errorf("job is nil")
	}
	return &clientpb.StagerListener{JobID: uint32(job.ID)}, err
}

// SaveStager payload save an obfuscated sliver build to disk and database
func (rpc *Server) SaveStager(ctx context.Context, req *clientpb.SaveStagerReq) (*clientpb.SaveStagerResp, error) {

	// write implant to disk
	appDir := assets.GetRootAppDir()
	fPath := filepath.Join(appDir, "builds", filepath.Base(req.Name))
	err := os.WriteFile(fPath, req.Stage, 0600)
	if err != nil {
		return &clientpb.SaveStagerResp{}, err
	}

	// save implant build
	fileName := filepath.Base(fPath)
	err = generate.ImplantBuildSave(req.Name, req.Config, fPath)
	if err != nil {
		rpcLog.Errorf("Failed to save build: %s", err)
		return nil, err
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(fileName),
	})

	// Implant profile ?
	// display implant conf and resource id -> reuse display client side ?
	res := clientpb.SaveStagerResp{}

	return &res, nil
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
