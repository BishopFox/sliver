package handlers

import pb "sliver/protobuf/sliver"

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
	}
)

func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}
