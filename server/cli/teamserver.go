package cli

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
	"errors"
	"log"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/rpc"
	"github.com/reeflective/team/server"
	teamGrpc "github.com/reeflective/team/transports/grpc/server"
	"google.golang.org/grpc"
)

func newSliverTeam() (*server.Server, *console.SliverClient) {
	// Teamserver
	gTeamserver := teamGrpc.NewListener()

	bindServer := func(grpcServer *grpc.Server) error {
		if grpcServer == nil {
			return errors.New("No gRPC server to use for service")
		}

		rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())

		return nil
	}

	var serverOpts []server.Options
	serverOpts = append(serverOpts,
		server.WithDefaultPort(31337),
		server.WithListener(gTeamserver),
	)

	gTeamserver.PostServe(bindServer)

	teamserver, err := server.New("sliver", serverOpts...)
	if err != nil {
		log.Fatal(err)
	}

	// Teamclient
	gTeamclient := teamGrpc.NewClientFrom(gTeamserver)

	sliver, opts := console.NewSliverClient(gTeamclient)

	sliver.Teamclient = teamserver.Self(opts...)

	return teamserver, sliver
}
