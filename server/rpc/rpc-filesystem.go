package rpc

import (
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/core"
	"time"

	"github.com/golang/protobuf/proto"
)

func rpcLs(req []byte, timeout time.Duration, resp RPCResponse) {
	dirList := &sliverpb.LsReq{}
	err := proto.Unmarshal(req, dirList)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(dirList.SliverID)

	data, _ := proto.Marshal(&sliverpb.LsReq{
		Path: dirList.Path,
	})
	data, err = sliver.Request(sliverpb.MsgLsReq, timeout, data)
	resp(data, err)
}

func rpcRm(req []byte, timeout time.Duration, resp RPCResponse) {
	rmReq := &sliverpb.RmReq{}
	err := proto.Unmarshal(req, rmReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(rmReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.RmReq{
		Path: rmReq.Path,
	})
	data, err = sliver.Request(sliverpb.MsgRmReq, timeout, data)
	resp(data, err)
}

func rpcMkdir(req []byte, timeout time.Duration, resp RPCResponse) {
	mkdirReq := &sliverpb.MkdirReq{}
	err := proto.Unmarshal(req, mkdirReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(mkdirReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.MkdirReq{
		Path: mkdirReq.Path,
	})
	data, err = sliver.Request(sliverpb.MsgMkdirReq, timeout, data)
	resp(data, err)
}

func rpcCd(req []byte, timeout time.Duration, resp RPCResponse) {
	cdReq := &sliverpb.CdReq{}
	err := proto.Unmarshal(req, cdReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(cdReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.CdReq{
		Path: cdReq.Path,
	})
	data, err = sliver.Request(sliverpb.MsgCdReq, timeout, data)
	resp(data, err)
}

func rpcPwd(req []byte, timeout time.Duration, resp RPCResponse) {
	pwdReq := &sliverpb.PwdReq{}
	err := proto.Unmarshal(req, pwdReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[pwdReq.SliverID]

	data, _ := proto.Marshal(&sliverpb.PwdReq{})
	data, err = sliver.Request(sliverpb.MsgPwdReq, timeout, data)
	resp(data, err)
}

func rpcDownload(req []byte, timeout time.Duration, resp RPCResponse) {
	downloadReq := &sliverpb.DownloadReq{}
	err := proto.Unmarshal(req, downloadReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(downloadReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.DownloadReq{
		Path: downloadReq.Path,
	})
	data, err = sliver.Request(sliverpb.MsgDownloadReq, timeout, data)
	resp(data, err)
}

func rpcUpload(req []byte, timeout time.Duration, resp RPCResponse) {
	uploadReq := &sliverpb.UploadReq{}
	err := proto.Unmarshal(req, uploadReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := core.Hive.Sliver(uploadReq.SliverID)

	data, _ := proto.Marshal(&sliverpb.UploadReq{
		Encoder: uploadReq.Encoder,
		Path:    uploadReq.Path,
		Data:    uploadReq.Data,
	})
	data, err = sliver.Request(sliverpb.MsgUploadReq, timeout, data)
	resp(data, err)
}
