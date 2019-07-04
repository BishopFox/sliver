package rpc

import (
	"time"

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"
	"github.com/golang/protobuf/proto"
)

func rpcExecute(req []byte, timeout time.Duration, resp RPCResponse) {
	execReq := &sliverpb.ExecuteReq{}

	err := proto.Unmarshal(req, execReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(execReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.ExecuteReq{
		Path:   execReq.Path,
		Args:   execReq.Args,
		Output: execReq.Output,
	})
	data, err = sliver.Request(sliverpb.MsgExecuteReq, timeout, data)
	resp(data, err)
}
