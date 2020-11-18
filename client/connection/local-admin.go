package connection

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/server/transport"
)

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

// ConnectLocal - We have started a Sliver server, and we connect a command locally.
func ConnectLocal() (conn *grpc.ClientConn, err error) {

	_, ln, _ := transport.LocalListener()
	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return ln.Dial()
	})

	options := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	conn, err = grpc.DialContext(context.Background(), "bufnet", options...)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to dial bufnet: %s", err)
		return
	}

	return
}
