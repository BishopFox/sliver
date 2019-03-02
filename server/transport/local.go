package transport

import (
	"fmt"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"
	"sliver/server/rpc"
)

// LocalClientConnect - Handles local connections to the server console
// keep in mind the arguments to this function are in the context of the client
// so send = "send to server" and recv = "recv from server"
func LocalClientConnect(send, recv chan *sliverpb.Envelope) {
	client := core.GetClient("server")
	client.Send = recv // Client's recv channel
	core.Clients.AddClient(client)

	go func() {
		rpcHandlers := rpc.GetRPCHandlers()
		tunHandlers := rpc.GetTunnelHandlers()
		for envelope := range send {
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
}
