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
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"
	"google.golang.org/grpc"
	"tailscale.com/tsnet"
)

var (
	tsLog = log.NamedLogger("transport", "ts")
)

// StartTsClientListener - Start a TSNet gRPC listener
func StartTsClientListener(hostname string, port uint16) (*grpc.Server, net.Listener, error) {
	tsLog.Infof("Starting gRPC/tsnet  listener on %s:%d", hostname, port)

	authKey := os.Getenv("TS_AUTHKEY")
	if authKey == "" {
		tsLog.Errorf("TS_AUTHKEY not set")
		return nil, nil, fmt.Errorf("TS_AUTHKEY not set")
	}

	appDir := assets.GetRootAppDir()
	tsnetDir := filepath.Join(appDir, "tsnet")
	if err := os.MkdirAll(tsnetDir, 0700); err != nil {
		return nil, nil, err
	}

	tsNetServer := &tsnet.Server{
		Hostname: hostname,
		Dir:      tsnetDir,
		Logf:     tsLog.Debugf,
		AuthKey:  authKey,
	}
	defer tsNetServer.Close()
	listenInterface := fmt.Sprintf(":%d", port)
	ln, err := tsNetServer.Listen("tcp", listenInterface)
	if err != nil {
		return nil, nil, err
	}

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, initMiddleware(true)...)
	grpcServer := grpc.NewServer(options...)
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				tsLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			tsLog.Warnf("gRPC/tsnet server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}
