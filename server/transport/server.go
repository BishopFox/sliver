package transport

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
	"net"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team/server"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	bufSize = 2 * mb

	// ServerMaxMessageSize - Server-side max GRPC message size.
	ServerMaxMessageSize = 2*gb - 1
)

// NewTeamserver returns a Sliver teamserver ready to run and serve
// itself over either TCP+MTLS/gRPC, or Tailscale+MTLS/gRPC channels.
// The client options returned should be passed to an in-memory teamclient.
// All errors returned by this function are critical: the server can't work.
func NewTeamserver() (team *server.Server, clientOpts []grpc.DialOption, err error) {
	tlsListener := newTeamserverTLS()
	tailscaleListener := newTeamserverTailScale()

	// Here is an import step, where we are given a change to setup
	// the reeflective/teamserver with everything we want: our own
	// database, the application daemon default port, loggers or files,
	// directories, and much more.
	var serverOpts []server.Options
	serverOpts = append(serverOpts,
		// Core directories/loggers.
		server.WithHomeDirectory(assets.GetRootAppDir()), // ~/.sliver/
		server.WithLogger(log.RootLogger),                // Logs to ~/.sliver/logs/sliver.{log,json} and audit.json
		server.WithDatabase(db.Client),                   // Uses our traditional ~/.sliver/sliver.db for storing users.

		// Network options/stacks
		server.WithDefaultPort(31337),          // Our now famous port.
		server.WithListener(tlsListener),       // Our legacy TCP+MTLS gRPC stack.
		server.WithListener(tailscaleListener), // And our new Tailscale variant.
	)

	// Create the application teamserver.
	// Any error is critical, and means we can't work correctly.
	teamserver, err := server.New("sliver", serverOpts...)
	if err != nil {
		return nil, nil, err
	}

	// The gRPC teamserver backend is hooked to produce a single
	// in-memory teamclient RPC/dialer backend. Not encrypted.
	return teamserver, clientOptionsFor(tlsListener), nil
}

// clientOptionsFor requires an existing grpc Teamserver to create an in-memory connection.
// Those options are passed to the SliverClient constructor for setting up its own dialer.
// It returns a teamclient meant to be ran in memory, with TLS credentials disabled.
func clientOptionsFor(server *teamserver, opts ...grpc.DialOption) []grpc.DialOption {
	conn := bufconn.Listen(bufSize)
	insecureCreds := insecure.NewCredentials()

	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return conn.Dial()
	})

	opts = append(opts, []grpc.DialOption{
		ctxDialer,
		grpc.WithTransportCredentials(insecureCreds),
	}...)

	// The server will use this conn as a listener.
	// The reference is dropped after server start.
	server.conn = conn

	return opts
}

// serve is the transport-agnostic routine to serve the gRPC server
// (and its implemented Sliver services) onto a generic listener.
// Both mTLS and Tailscale teamserver backends use this.
func (h *teamserver) serve(ln net.Listener) {
	grpcServer := grpc.NewServer(h.options...)

	rpcLog := h.NamedLogger("transport", "gRPC")

	// Teamserver/Sliver services
	sliverServer := rpc.NewServer(h.Server)
	rpcpb.RegisterSliverRPCServer(grpcServer, sliverServer)

	rpcLog.Infof("Serving gRPC teamserver on %s", ln.Addr())

	// Start serving the listener
	go func() {
		panicked := true
		defer func() {
			if panicked {
				rpcLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()

		if err := grpcServer.Serve(ln); err != nil {
			rpcLog.Errorf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
}
