package core

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"io"
	"log"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TunnelLoop - Parses incoming tunnel messages and distributes them
//
//	             to session/tunnel objects
//					Expected to be called only once during initialization
func TunnelLoop(ctx context.Context, rpc rpcpb.SliverRPCClient) error {
	return tunnelLoop(ctx, rpc, nil)
}

// TunnelLoopWithReady runs TunnelLoop and closes ready after the tunnel data
// stream has been installed. Callers can use that signal to avoid accepting
// local proxy connections before tunnel bind frames can be sent.
func TunnelLoopWithReady(ctx context.Context, rpc rpcpb.SliverRPCClient, ready chan<- struct{}) error {
	return tunnelLoop(ctx, rpc, ready)
}

//nolint:gocyclo // The shared stream loop keeps receive, ownership, close, and cancellation decisions together.
func tunnelLoop(ctx context.Context, rpc rpcpb.SliverRPCClient, ready chan<- struct{}) error {
	log.Println("Starting tunnel data loop ...")
	defer log.Printf("Warning: TunnelLoop exited")

	stream, err := rpc.TunnelData(ctx)

	if err != nil {
		return err
	}

	tunnels := GetTunnels()
	streamGeneration := tunnels.SetStream(stream)
	defer tunnels.CloseStream(streamGeneration)
	if ready != nil {
		close(ready)
	}

	for {
		log.Printf("Waiting for TunnelData ...")
		incoming, err := stream.Recv()
		if incoming != nil {
			log.Printf(
				"Recv stream msg: tunnel=%d session=%s bytes=%d closed=%t",
				incoming.TunnelID,
				incoming.SessionID,
				len(incoming.Data),
				incoming.Closed,
			)
		}
		if err != nil {
			log.Printf("Recv stream err: %s", err)
		}

		if err == io.EOF {
			log.Printf("EOF Error: Tunnel data stream closed")
			return nil
		}
		if err != nil {
			// A canceled status is only a clean shutdown when this caller
			// canceled the stream. A remote/server cancellation while ctx is
			// still live is a real tunnel-loop failure and must be surfaced.
			if ctx.Err() != nil && (errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled) {
				return nil
			}
			log.Printf("Tunnel data read error: %s", err)
			return err
		}
		log.Printf("Received TunnelData for tunnel %d", incoming.TunnelID)
		tunnel := tunnels.Get(incoming.TunnelID)

		if tunnel != nil {
			if !incoming.Closed && len(incoming.GetData()) == 0 {
				// The server uses an empty frame to acknowledge that the client
				// stream is bound to a newly created tunnel. It is control
				// traffic, not data for the tunnel reader.
				tunnel.markBound()
				log.Printf("Received bind acknowledgement for tunnel %d", incoming.TunnelID)
				continue
			}
			data := incoming.GetData()

			if !incoming.Closed {
				log.Printf("Received %d byte(s) on tunnel %d", len(data), tunnel.ID)
				err = tunnel.RecvData(data)

				if err != nil {
					log.Printf("Warning! Closing tunnel %d after receive admission failed: %v", tunnel.ID, err)
					tunnel.failReceive()
					tunnels.CloseIf(tunnel)
				}
			} else {
				log.Printf("Closing tunnel %d", tunnel.ID)
				tunnels.CloseIf(tunnel)
			}
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", incoming.TunnelID)
		}
	}
}
