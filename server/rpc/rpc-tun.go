package rpc

import (
	"sliver/server/core"

	clientpb "sliver/protobuf/client"

	"github.com/golang/protobuf/proto"
)

func tunCreate(client *core.Client, req []byte, resp RPCResponse) {

	tunCreateReq := &clientpb.TunnelCreateReq{}
	proto.Unmarshal(req, tunCreateReq)

	tunnel := core.Tunnels.CreateTunnel(client, tunCreateReq.SliverID)

	data, err := proto.Marshal(&clientpb.TunnelCreate{
		SliverID: tunnel.Sliver.ID,
		TunnelID: tunnel.ID,
	})
	resp(data, err)
}

func tunData(client *core.Client, req []byte, _ RPCResponse) {

}

func tunClose(client *core.Client, req []byte, resp RPCResponse) {

}
