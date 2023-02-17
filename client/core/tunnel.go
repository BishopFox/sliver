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
	"io"
	"log"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// TunnelLoop - Parses incoming tunnel messages and distributes them
//              to session/tunnel objects
// 				Expected to be called only once during initialization
func TunnelLoop(rpc rpcpb.SliverRPCClient) error {
	log.Println("Starting tunnel data loop ...")
	defer log.Printf("Warning: TunnelLoop exited")

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := rpc.TunnelData(ctx)
	defer cancel()

	if err != nil {
		return err
	}

	GetTunnels().SetStream(stream)

	for {
		log.Printf("Waiting for TunnelData ...")
		incoming, err := stream.Recv()
		log.Printf("Recv stream msg: %v", incoming)
		log.Printf("Recv stream err: %s", err)

		if err == io.EOF {
			log.Printf("EOF Error: Tunnel data stream closed")
			return nil
		}
		if err != nil {
			log.Printf("Tunnel data read error: %s", err)
			return err
		}
		log.Printf("Received TunnelData for tunnel %d", incoming.TunnelID)
		tunnel := GetTunnels().Get(incoming.TunnelID)

		if tunnel != nil {
			data := incoming.GetData()

			log.Printf("This is data on tunnel %d: %s", tunnel.ID, data)

			if !incoming.Closed {
				log.Printf("Received data on tunnel %d", tunnel.ID)
				err = tunnel.RecvData(data)

				if err != nil {
					log.Printf("Warning! Error sending data to tunnel %d, %v", tunnel.ID, err)
				}
			} else {
				log.Printf("Closing tunnel %d", tunnel.ID)
				GetTunnels().Close(tunnel.ID)
			}
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", incoming.TunnelID)
		}
	}
}
