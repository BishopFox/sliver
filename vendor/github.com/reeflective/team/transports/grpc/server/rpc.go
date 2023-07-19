package server

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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

	"github.com/reeflective/team/server"
	"github.com/reeflective/team/transports/grpc/proto"
)

// rpcServer is the gRPC server implementation.
// It makes uses of the teamserver core to query users and version information.
type rpcServer struct {
	server *server.Server
	*proto.UnimplementedTeamServer
}

func newServer(server *server.Server) *rpcServer {
	return &rpcServer{
		server:                  server,
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}
}

// GetVersion returns the teamserver version.
func (ts *rpcServer) GetVersion(context.Context, *proto.Empty) (*proto.Version, error) {
	ver, err := ts.server.Version()

	return &proto.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, err
}

// GetUsers returns the list of teamserver users and their status.
func (ts *rpcServer) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	users, err := ts.server.Users()

	userspb := make([]*proto.User, len(users))
	for i, user := range users {
		userspb[i] = &proto.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: user.LastSeen.Unix(),
			Clients:  int32(user.Clients),
		}
	}

	return &proto.Users{Users: userspb}, err
}
