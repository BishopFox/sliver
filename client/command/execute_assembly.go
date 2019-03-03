package command

import (
	"fmt"
	"io/ioutil"

	"sliver/client/spin"
	sliverpb "sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func executeAssembly(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "Please provide valid arguments.\n")
		return
	}
	assemblyBytes, err := ioutil.ReadFile(ctx.Args[0])
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}

	assemblyArgs := ""
	if len(ctx.Args) == 2 {
		assemblyArgs = ctx.Args[1]
	}

	ctrl := make(chan bool)
	go spin.Until("Executing assembly ...", ctrl)
	data, _ := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		SliverID:   ActiveSliver.Sliver.ID,
		Timeout:    int32(5),
		Arguments:  assemblyArgs,
		Assembly:   assemblyBytes,
		HostingDll: []byte{},
	})

	resp := <-rpc(&sliverpb.Envelope{
		Data: data,
		Type: sliverpb.MsgExecuteAssembly,
	}, defaultTimeout)
	ctrl <- true
	execResp := &sliverpb.ExecuteAssembly{}
	proto.Unmarshal(resp.Data, execResp)
	if execResp.Error != "" {
		fmt.Printf(Warn+"%s", execResp.Error)
		return
	}
	fmt.Printf("\n"+Info+"Assembly output:\n%s", execResp.Output)
}
