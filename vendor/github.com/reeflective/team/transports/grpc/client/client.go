package client

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
	"errors"
	"fmt"
	"time"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/transports/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb.
	ClientMaxReceiveMessageSize = (2 * gb) - 1 // 2Gb - 1 byte

	defaultTimeout = 10 * time.Second
)

var (
	// ErrNoRPC indicates that no gRPC generated proto.Teamclient bound to a client
	// connection is available. The error is raised when the handler hasn't connected.
	ErrNoRPC = errors.New("no working grpc.Teamclient available")

	// ErrNoTLSCredentials is an error raised if the teamclient was asked to setup, or try
	// connecting with, TLS credentials. If such an error is raised, make sure your team
	// client has correctly fetched -using client.Config()- a remote teamserver config.
	ErrNoTLSCredentials = errors.New("the grpc Teamclient has no TLS credentials to use")
)

// Teamclient is a simple example gRPC teamclient and dialer backend.
// It comes correctly configured with Mutual TLS authentication and
// RPC connection/registration/use when created with NewTeamClient().
//
// This teamclient embeds a team/client.Client core driver and uses
// it for fetching/setting up the transport credentials, dialers, etc...
// It also has a few internal types (clientConns, options) for working.
//
// Note that this teamclient is not able to be used as an in-memory dialer.
// See the counterpart `team/transports/grpc/server` package for creating one.
// Also note that this example transport has been made for a single use-case,
// and that your program might require more elaborated behavior.
// In this case, please use this simple code as a reference for what/not to do.
type Teamclient struct {
	*client.Client
	conn    *grpc.ClientConn
	rpc     proto.TeamClient
	options []grpc.DialOption
}

// NewTeamClient creates a new gRPC-based RPC teamclient and dialer backend.
// This client has by default only a few options, like max message buffer size.
// All options passed to this call are stored as is and will be used later.
func NewTeamClient(opts ...grpc.DialOption) *Teamclient {
	client := &Teamclient{
		options: opts,
	}

	client.options = append(client.options,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	)

	return client
}

// Init implements team/client.Dialer.Init(c).
// This implementation asks the teamclient core for its remote server
// configuration, and uses it to load a set of Mutual TLS dialing options.
func (h *Teamclient) Init(cli *client.Client) error {
	h.Client = cli
	config := cli.Config()

	options := LogMiddlewareOptions(cli)

	// If the configuration has no credentials, we are most probably
	// an in-memory dialer, don't authenticate and encrypt the conn.
	if config.PrivateKey != "" {
		tlsOpts, err := tlsAuthMiddleware(cli)
		if err != nil {
			return err
		}

		h.options = append(h.options, tlsOpts...)
	}

	h.options = append(h.options, options...)

	return nil
}

// Dial implements team/client.Dialer.Dial().
// It uses the teamclient remote server configuration as a target of a dial call.
// If the connection is successful, the teamclient registers a proto.Teamclient
// RPC around its client connection, to provide the core teamclient functionality.
func (h *Teamclient) Dial() (rpcClient any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	host := fmt.Sprintf("%s:%d", h.Config().Host, h.Config().Port)

	h.conn, err = grpc.DialContext(ctx, host, h.options...)
	if err != nil {
		return nil, err
	}

	h.rpc = proto.NewTeamClient(h.conn)

	return h.conn, nil
}

// Close implements team/client.Dialer.Close(), and closes the gRPC client connection.
func (h *Teamclient) Close() error {
	return h.conn.Close()
}

// Users returns a list of all users registered with the app teamserver.
// If the gRPC teamclient is not connected or does not have an RPC client,
// an ErrNoRPC is returned.
func (h *Teamclient) Users() (users []team.User, err error) {
	if h.rpc == nil {
		return nil, ErrNoRPC
	}

	res, err := h.rpc.GetUsers(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, err
	}

	for _, user := range res.GetUsers() {
		users = append(users, team.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: time.Unix(user.LastSeen, 0),
		})
	}

	return
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
// If the gRPC teamclient is not connected or does not have an RPC client,
// an ErrNoRPC is returned.
func (h *Teamclient) Version() (version team.Version, err error) {
	if h.rpc == nil {
		return version, ErrNoRPC
	}

	ver, err := h.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return version, errors.New(status.Convert(err).Message())
	}

	return team.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}
