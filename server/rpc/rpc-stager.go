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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/c2"
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
	conf := &c2.HTTPServerConfig{
		Addr:   fmt.Sprintf("%s:%d", host, req.Port),
		LPort:  uint16(req.Port),
		Domain: req.Host,
		Secure: false,
	}
	if req.GetProtocol() == clientpb.StageProtocol_HTTPS {
		conf.Secure = true
		conf.Key = req.Key
		conf.Cert = req.Cert
		conf.ACME = req.ACME
	}
	job, err := c2.StartHTTPStagerListenerJob(conf, req.ProfileName, req.Data)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, fmt.Errorf("job is nil")
	}
	return &clientpb.StagerListener{JobID: uint32(job.ID)}, err
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
