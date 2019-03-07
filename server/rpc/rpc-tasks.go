package rpc

import (
	"sliver/server/core"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcLocalTask(req []byte, resp RPCResponse) {
	taskReq := &clientpb.TaskReq{}
	err := proto.Unmarshal(req, taskReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(taskReq.SliverID)
	data, _ := proto.Marshal(&sliverpb.Task{
		Encoder: "raw",
		Data:    taskReq.Data,
	})
	data, err = sliver.Request(sliverpb.MsgTask, defaultTimeout, data)
	resp(data, err)
}
