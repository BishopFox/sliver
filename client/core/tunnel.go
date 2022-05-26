package core

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

	stream, err := rpc.TunnelData(context.Background())
	if err != nil {
		return err
	}

	GetTunnels().SetStream(stream)

	for {
		// log.Printf("Waiting for TunnelData ...")
		incoming, err := stream.Recv()
		// log.Printf("Recv stream msg: %v", incoming)
		if err == io.EOF {
			log.Printf("EOF Error: Tunnel data stream closed")
			return nil
		}
		if err != nil {
			log.Printf("Tunnel data read error: %s", err)
			return err
		}
		// log.Printf("Received TunnelData for tunnel %d", incoming.TunnelID)
		tunnel := GetTunnels().Get(incoming.TunnelID)

		if tunnel != nil {
			if !incoming.Closed {
				log.Printf("Received data on tunnel %d", tunnel.ID)
				tunnel.Recv <- incoming.GetData()
			} else {
				log.Printf("Closing tunnel %d", tunnel.ID)
				GetTunnels().Close(tunnel.ID)
			}
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", incoming.TunnelID)
		}
	}
}
