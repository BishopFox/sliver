package console

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
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	cliTransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"
	"github.com/bishopfox/sliver/server/transport"
)

const bufSize = 2 * mb

var (
	pipeLog = log.NamedLogger("transport", "local")
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = 2 * gb

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * gb
)

// connectLocal - We have started a Sliver server, and we connect a command locally.
func connectLocal() (*grpc.ClientConn, error) {

	_, ln, _ := localListener()
	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return ln.Dial()
	})

	options := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	var err error
	conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to dial bufnet: %s", err)
		return nil, err
	}

	// The client keeps a reference of this connection.
	cliTransport.SetClientConnGRPC(conn)

	return conn, nil
}

// localListener - Bind gRPC server to an in-memory listener, which is
//                 typically used for unit testing, but ... it should be fine
func localListener() (*grpc.Server, *bufconn.Listener, error) {
	pipeLog.Infof("Binding gRPC to listener ...")
	ln := bufconn.Listen(bufSize)
	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, transport.InitLoggerMiddleware()...)
	grpcServer := grpc.NewServer(options...)

	// Register both normal and admin server.
	rpcServer := rpc.NewServer()
	rpcpb.RegisterSliverRPCServer(grpcServer, rpcServer)
	rpcpb.RegisterSliverAdminRPCServer(grpcServer, rpcServer)

	// Monitoring the coming connection in the background.
	go func() {
		if err := grpcServer.Serve(ln); err != nil {
			pipeLog.Fatalf("gRPC local listener error: %v", err)
		}
	}()
	return grpcServer, ln, nil
}
