package transport

import (
	"fmt"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"
	"sliver/server/rpc"
)

func localClientConnect(send, recv chan *sliverpb.Envelope) {
	client := core.GetClient("server")
	core.Clients.AddClient(client)

	go func() {
		rpcHandlers := rpc.GetRPCHandlers()
		tunHandlers := rpc.GetTunnelHandlers()
		for envelope := range recv {
			// RPC
			if rpcHandler, ok := (*rpcHandlers)[envelope.Type]; ok {
				go rpcHandler(envelope.Data, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:    envelope.ID,
						Data:  data,
						Error: errStr,
					}
				})
			}
			// TUN
			if tunHandler, ok := (*tunHandlers)[envelope.Type]; ok {
				go tunHandler(client, envelope.Data, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:    envelope.ID,
						Data:  data,
						Error: errStr,
					}
				})
			}
		}
	}()

	for envelope := range client.Send {
		send <- envelope
	}

}
