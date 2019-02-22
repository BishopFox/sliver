package rpc

import (
	"log"
	"sliver/server/core"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func tunnelCreate(client *core.Client, req []byte, resp RPCResponse) {

	tunCreateReq := &clientpb.TunnelCreateReq{}
	proto.Unmarshal(req, tunCreateReq)

	tunnel := core.Tunnels.CreateTunnel(client, tunCreateReq.SliverID)

	data, err := proto.Marshal(&clientpb.TunnelCreate{
		SliverID: tunnel.Sliver.ID,
		TunnelID: tunnel.ID,
	})

	resp(data, err)
}

func tunnelData(client *core.Client, req []byte, _ RPCResponse) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(req, tunnelData)
	tunnel := core.Tunnels.Tunnel(tunnelData.TunnelID)
	if tunnel != nil && client.ID == tunnel.Client.ID {
		tunnel.Sliver.Request(sliverpb.MsgTunnelData, defaultTimeout, req)
	} else {
		log.Printf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
}

func tunnelClose(client *core.Client, req []byte, resp RPCResponse) {
	tunCloseReq := &clientpb.TunnelCloseReq{}
	proto.Unmarshal(req, tunCloseReq)

	closed := core.Tunnels.CloseTunnel(client, tunCloseReq.TunnelID)

	data, err := proto.Marshal(&clientpb.TunnelClose{
		TunnelID: tunCloseReq.TunnelID,
		Closed:   closed,
	})

	resp(data, err)
}
