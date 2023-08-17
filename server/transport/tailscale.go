package transport

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"tailscale.com/tsnet"

	"github.com/reeflective/team/server"

	"github.com/bishopfox/sliver/server/assets"
)

// tailscaleTeamserver is unexported since we only need it as
// a reeflective/team/server.Listener interface implementation.
type tailscaleTeamserver struct {
	*teamserver
}

// newTeamserverTailScale returns a Sliver teamserver backend using Tailscale.
func newTeamserverTailScale(opts ...grpc.ServerOption) server.Listener {
	core := newTeamserverTLS(opts...)

	return &tailscaleTeamserver{core}
}

// Name indicates the transport/rpc stack.
func (ts *tailscaleTeamserver) Name() string {
	return "gRPC/TSNet"
}

// Close implements team/server.Handler.Close().
// Instead of serving a classic TCP+TLS listener,
// we start a tailscale stack and create the listener out of it.
func (ts *tailscaleTeamserver) Listen(addr string) (ln net.Listener, err error) {
	tsNetLog := ts.NamedLogger("transport", "tailscale")

	url, err := url.Parse(fmt.Sprintf("ts://%s", addr))
	if err != nil {
		return nil, err
	}

	hostname := url.Hostname()
	port := url.Port()

	if hostname == "" {
		hostname = "sliver-server"
		machineName, _ := os.Hostname()
		if machineName != "" {
			hostname = fmt.Sprintf("%s-%s", hostname, machineName)
		}
	}

	tsNetLog.Infof("Starting gRPC/tsnet listener on %s:%s", hostname, port)

	authKey := os.Getenv("TS_AUTHKEY")
	if authKey == "" {
		tsNetLog.Errorf("TS_AUTHKEY not set")
		return nil, fmt.Errorf("TS_AUTHKEY not set")
	}

	tsnetDir := filepath.Join(assets.GetRootAppDir(), "tsnet")
	if err := os.MkdirAll(tsnetDir, 0o700); err != nil {
		return nil, err
	}

	tsNetServer := &tsnet.Server{
		Hostname: hostname,
		Dir:      tsnetDir,
		Logf:     tsNetLog.Debugf,
		AuthKey:  authKey,
	}

	ln, err = tsNetServer.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, err
	}

	ts.serve(ln)

	return ln, nil
}
