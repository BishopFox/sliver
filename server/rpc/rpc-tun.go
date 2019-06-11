package rpc

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
	"time"

	"github.com/bishopfox/sliver/server/core"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

const (
	tunDefaultTimeout = 30 * time.Second
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
		tunnel.Sliver.Request(sliverpb.MsgTunnelData, tunDefaultTimeout, req)
	} else {
		rpcLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
}

func tunnelClose(client *core.Client, req []byte, resp RPCResponse) {
	tunCloseReq := &clientpb.TunnelCloseReq{}
	proto.Unmarshal(req, tunCloseReq)

	tunnel := core.Tunnels.Tunnel(tunCloseReq.TunnelID)

	if tunnel != nil && client.ID == tunnel.Client.ID {
		closed := core.Tunnels.CloseTunnel(tunCloseReq.TunnelID, "Client exit")
		closeResp := &sliverpb.TunnelClose{
			TunnelID: tunCloseReq.TunnelID,
		}
		if !closed {
			closeResp.Err = "Failed to close tunnel"
		}
		data, err := proto.Marshal(closeResp)
		resp(data, err)
	}
}
