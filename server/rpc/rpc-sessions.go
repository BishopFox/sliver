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

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcKill(data []byte, timeout time.Duration, resp RPCResponse) {
	killReq := &sliverpb.KillReq{}
	err := proto.Unmarshal(data, killReq)
	if err != nil {
		resp([]byte{}, err)
	}
	sliver := core.Hive.Sliver(killReq.SliverID)
	data, err = sliver.Request(sliverpb.MsgKill, timeout, data)
	core.Hive.RemoveSliver(sliver)
	resp(data, err)
}

func rpcSessions(_ []byte, timeout time.Duration, resp RPCResponse) {
	sessions := &clientpb.Sessions{}
	if 0 < len(*core.Hive.Slivers) {
		for _, sliver := range *core.Hive.Slivers {
			sessions.Slivers = append(sessions.Slivers, sliver.ToProtobuf())
		}
	}
	data, err := proto.Marshal(sessions)
	if err != nil {
		rpcLog.Errorf("Error encoding rpc response %v", err)
	}
	resp(data, err)
}
