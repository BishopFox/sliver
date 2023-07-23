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
	"sync"

	"github.com/reeflective/team/server"
	"google.golang.org/grpc"
	"tailscale.com/tsnet"

	"github.com/bishopfox/sliver/server/assets"
)

type TailScaleTeamserver struct {
	*Teamserver
}

func NewTailScaleListener(opts ...grpc.ServerOption) server.Listener {
	core := &Teamserver{
		mutex: &sync.RWMutex{},
	}

	core.options = append(core.options, opts...)

	return &TailScaleTeamserver{core}
}

func (ts *TailScaleTeamserver) Listen(addr string) (ln net.Listener, err error) {
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

	tsNetLog.Infof("Starting gRPC/tsnet listener on %s:%d", hostname, port)

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

	ln, err = tsNetServer.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	ts.serve(ln)

	return ln, nil
}

// It indicates the transport/rpc stack, in this case "gRPC".
func (ts *TailScaleTeamserver) Name() string {
	return "gRPC/TSNet"
}
