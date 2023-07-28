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
	"sync"

	teamserver "github.com/reeflective/team/server"
	"github.com/reeflective/team/transports/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	clientTransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
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

// Teamserver is a vanilla TCP+MTLS gRPC server offering all Sliver services through it.
// This listener backend embeds a team/server.Server core driver and uses it for fetching
// server-side TLS configurations, use its loggers and access its database/users/list.
type Teamserver struct {
	*teamserver.Server

	options []grpc.ServerOption
	conn    *bufconn.Listener
	mutex   *sync.RWMutex
}

// NewListener returns a vanilla tcp+mtls gRPC teamserver listener backend.
// Developers: note that the teamserver type is already set with logging/
// auth/middleware/buffering gRPC options. You can still override them.
func NewListener(opts ...grpc.ServerOption) *Teamserver {
	listener := &Teamserver{
		mutex:   &sync.RWMutex{},
		options: bufferingOptions(),
	}

	listener.options = append(listener.options, opts...)

	return listener
}

// NewClientFrom requires an existing grpc Teamserver to create an in-memory
// connection bound to both the teamserver and the teamclient backends.
// It returns a teamclient meant to be ran in memory, with TLS credentials disabled.
func NewClientFrom(server *Teamserver, opts ...grpc.DialOption) *clientTransport.Teamclient {
	conn := bufconn.Listen(bufSize)

	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return conn.Dial()
	})

	opts = append(opts, []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(),
	}...)

	// The server will use this conn as a listener.
	// The reference is dropped after server start.
	server.conn = conn

	return clientTransport.NewTeamClient(opts...)
}

// Name immplements team/server.Handler.Name().
// It indicates the transport/rpc stack.
func (h *Teamserver) Name() string {
	return "gRPC/mTLS"
}

// Init implements team/server.Handler.Init().
// It is used to initialize the listener with the correct TLS credentials
// middleware (or absence of if about to serve an in-memory connection).
func (h *Teamserver) Init(serv *teamserver.Server) (err error) {
	h.Server = serv

	// Logging
	logOptions, err := logMiddlewareOptions(h.Server)
	if err != nil {
		return err
	}

	h.options = append(h.options, logOptions...)

	// Authentication/audit
	authOptions, err := h.initAuthMiddleware()
	if err != nil {
		return err
	}

	h.options = append(h.options, authOptions...)

	return nil
}

// Listen implements team/server.Handler.Listen().
// this teamserver uses a tcp+TLS (mutual) listener to serve remote clients.
func (h *Teamserver) Listen(addr string) (ln net.Listener, err error) {
	// In-memory connection are not authenticated.
	if h.conn == nil {
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}

		// Encryption.
		tlsOptions, err := tlsAuthMiddlewareOptions(h.Server)
		if err != nil {
			return nil, err
		}

		h.options = append(h.options, tlsOptions...)
	} else {
		h.mutex.Lock()
		ln = h.conn
		h.conn = nil
		h.mutex.Unlock()
	}

	h.serve(ln)

	return ln, nil
}

// Close implements team/server.Handler.Close().
// Original sliver never closes the gRPC HTTP server itself
// with server.Shutdown(), so here we don't close anything.
// Note that the listener itself is controled/closed by
// our core teamserver driver.
func (h *Teamserver) Close() error {
	return nil
}

// serve is the transport-agnostic routine to serve the gRPC server
// (and its implemented Sliver services) onto a generic listener.
// Both mTLS and Tailscale teamserver backends use this.
func (h *Teamserver) serve(ln net.Listener) {
	grpcServer := grpc.NewServer(h.options...)

	rpcLog := h.NamedLogger("transport", "gRPC")

	// Teamserver/Sliver services
	proto.RegisterTeamServer(grpcServer, newServer(h.Server))
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())

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
