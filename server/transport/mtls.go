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
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team/server"
)

// teamserver is a vanilla TCP+MTLS gRPC server offering all Sliver services through it.
// This listener backend embeds a team/server.Server core driver and uses it for fetching
// server-side TLS configurations, use its loggers and access its database/users/list.
type teamserver struct {
	*server.Server

	options []grpc.ServerOption
	conn    *bufconn.Listener
	mutex   *sync.RWMutex
}

// newTeamserverTLS returns a vanilla tcp+mtls gRPC teamserver listener backend.
// Developers: note that the teamserver type is already set with logging/
// auth/middleware/buffering gRPC options. You can still override them.
func newTeamserverTLS(opts ...grpc.ServerOption) *teamserver {
	listener := &teamserver{
		mutex:   &sync.RWMutex{},
		options: bufferingOptions(),
	}

	listener.options = append(listener.options, opts...)

	return listener
}

// Name immplements team/server.Handler.Name().
// It indicates the transport/rpc stack.
func (h *teamserver) Name() string {
	return "gRPC/mTLS"
}

// Init implements team/server.Handler.Init().
// It is used to initialize the listener with the correct TLS credentials
// middleware (or absence of if about to serve an in-memory connection).
func (h *teamserver) Init(team *server.Server) (err error) {
	h.Server = team

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
func (h *teamserver) Listen(addr string) (ln net.Listener, err error) {
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
func (h *teamserver) Close() error {
	return nil
}
