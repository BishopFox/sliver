package rpc

import (
	"sliver/server/core"
	"time"

	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcPs(req []byte, resp RPCResponse) {
	psReq := &sliverpb.PsReq{}
	err := proto.Unmarshal(req, psReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[psReq.SliverID]
	if sliver == nil {
		resp([]byte{}, err)
		return
	}

	data, _ := proto.Marshal(&sliverpb.PsReq{})
	data, err = sliver.Request(sliverpb.MsgPsListReq, defaultTimeout, data)
	resp(data, err)
}

func rpcProcdump(req []byte, resp RPCResponse) {
	procdumpReq := &sliverpb.ProcessDumpReq{}
	err := proto.Unmarshal(req, procdumpReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[procdumpReq.SliverID]
	if sliver == nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.ProcessDumpReq{
		Pid: procdumpReq.Pid,
	})
	timeout := time.Duration(procdumpReq.Timeout) * time.Second
	data, err = sliver.Request(sliverpb.MsgProcessDumpReq, timeout, data)
	resp(data, err)
}
