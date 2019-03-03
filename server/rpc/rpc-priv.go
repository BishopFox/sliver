package rpc

import (
	"fmt"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"
	"time"

	"github.com/golang/protobuf/proto"
)

func rpcImpersonate(req []byte, resp RPCResponse) {
	impersonateReq := &sliverpb.ImpersonateReq{}
	err := proto.Unmarshal(req, impersonateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[int(impersonateReq.SliverID)]
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ImpersonateReq{
		Process:  impersonateReq.Process,
		Username: impersonateReq.Username,
		Args:     impersonateReq.Args,
	})
	timeout := 30 * time.Second
	data, err = sliver.Request(sliverpb.MsgImpersonateReq, timeout, data)
	resp(data, err)
}

func rpcElevate(req []byte, resp RPCResponse) {
	elevateReq := &sliverpb.ElevateReq{}
	err := proto.Unmarshal(req, elevateReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[int(elevateReq.SliverID)]
	if sliver == nil {
		resp([]byte{}, fmt.Errorf("Could not find sliver"))
		return
	}
	data, _ := proto.Marshal(&sliverpb.ElevateReq{})
	timeout := 30 * time.Second
	data, err = sliver.Request(sliverpb.MsgElevateReq, timeout, data)
	resp(data, err)

}
