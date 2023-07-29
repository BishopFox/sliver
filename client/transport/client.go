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
	"fmt"

	"github.com/reeflective/team/client"
	"google.golang.org/grpc"
)

// TeamClient is a type implementing the reeflective/team/client.Dialer
// interface, and can thus be used to communicate with any remote or
// in-memory Sliver teamserver.
// When used to connect remotely, this type can safely
// be instantiated with `new(transport.Teamclient)`.
type TeamClient struct {
	team    *client.Client
	options []grpc.DialOption
	Conn    *grpc.ClientConn
}

// NewClient creates a teamclient transport with specific gRPC options.
// It can also be used for in-memory clients, which specify their dialer.
func NewClient(opts ...grpc.DialOption) *TeamClient {
	tc := new(TeamClient)
	tc.options = append(tc.options, opts...)

	return tc
}

// Init implements team/client.Dialer.Init(c).
// It uses teamclient core driver for a remote server configuration.
// It also includes all pre-existing Sliver-specific log/middleware.
func (h *TeamClient) Init(cli *client.Client) error {
	h.team = cli
	config := cli.Config()

	// Buffering
	h.options = append(h.options,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize),
		),
	)

	// Logging/audit
	options := LogMiddlewareOptions(cli)
	h.options = append(h.options, options...)

	// If the configuration has no credentials, we are an
	// in-memory dialer, don't authenticate/encrypt the conn.
	if config.PrivateKey != "" {
		tlsOpts, err := TLSAuthMiddleware(cli)
		if err != nil {
			return err
		}

		h.options = append(h.options, tlsOpts...)
	}

	return nil
}

// Dial implements team/client.Dialer.Dial().
// It uses the teamclient remote server configuration as a target of a dial call.
// If the connection is successful, the teamclient registers a Sliver RPC client.
func (h *TeamClient) Dial() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cfg := h.team.Config()

	host := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	h.Conn, err = grpc.DialContext(ctx, host, h.options...)
	if err != nil {
		return err
	}

	return nil
}

// Close implements team/client.Dialer.Close().
// It closes the gRPC client connection if any.
func (h *TeamClient) Close() error {
	if h.Conn == nil {
		return nil
	}

	return h.Conn.Close()
}
