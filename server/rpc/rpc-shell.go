package rpc

import (
	clientpb "sliver/protobuf/client"
	"sliver/server/core"

	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcStartShell(req []byte, resp RPCResponse) {
	shellReq := &clientpb.ShellReq{}
	proto.Unmarshal(req, shellReq)
	sliver := (*core.Hive.Slivers)[int(shellReq.SliverID)]
	data, err := sliver.Request(sliverpb.MsgShellReq, defaultTimeout, []byte{})
	resp(data, err)
}

func rpcShellData(req []byte, resp RPCResponse) {

}
