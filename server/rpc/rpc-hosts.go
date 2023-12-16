package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
)

// var (
// 	hostsRPCLog = log.NamedLogger("rpc", "hosts")
// )

// Hosts - List all hosts
func (rpc *Server) Hosts(ctx context.Context, _ *commonpb.Empty) (*clientpb.AllHosts, error) {
	dbHosts, err := db.ListHosts()
	if err != nil {
		return nil, err
	}
	hosts := []*clientpb.Host{}
	for _, dbHost := range dbHosts {
		hosts = append(hosts, dbHost)
	}
	return &clientpb.AllHosts{Hosts: hosts}, nil
}

// Host - Host by ID
func (rpc *Server) Host(ctx context.Context, req *clientpb.Host) (*clientpb.Host, error) {
	dbHost, err := db.HostByHostUUID(req.HostUUID)
	if err != nil {
		return nil, err
	}
	return dbHost.ToProtobuf(), nil
}

// HostRm - Remove a host from the database
func (rpc *Server) HostRm(ctx context.Context, req *clientpb.Host) (*commonpb.Empty, error) {
	dbHost, err := db.HostByHostUUID(req.HostUUID)
	if err != nil {
		return nil, err
	}
	err = db.Session().Delete(*dbHost).Error
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

// HostIOCRm - Remove a host from the database
func (rpc *Server) HostIOCRm(ctx context.Context, req *clientpb.IOC) (*commonpb.Empty, error) {
	dbIOC, err := db.IOCByID(req.ID)
	if err != nil {
		return nil, err
	}
	err = db.Session().Delete(dbIOC).Error
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}
