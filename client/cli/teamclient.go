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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/reeflective/team/client"
	teamGrpc "github.com/reeflective/team/transports/grpc/client"
	"google.golang.org/grpc"
)

func newSliverTeam(con *console.SliverClient) *client.Client {
	gTeamclient := teamGrpc.NewTeamClient()

	bindClient := func(clientConn any) error {
		grpcClient, ok := clientConn.(*grpc.ClientConn)
		if !ok || grpcClient == nil {
			return errors.New("No gRPC client to use for service")
		}

		con.Rpc = rpcpb.NewSliverRPCClient(grpcClient)

		return nil
	}

	var clientOpts []client.Options
	clientOpts = append(clientOpts,
		client.WithDialer(gTeamclient, bindClient),
	)

	teamclient, err := client.New("sliver", gTeamclient, clientOpts...)
	if err != nil {
		panic(err)
	}

	return teamclient
}
