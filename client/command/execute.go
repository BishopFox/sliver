package command

import (
	"fmt"
	"strings"

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func execute(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "Please provide a path. See `help execute` for more info.\n")
		return
	}

	cmdPath := ctx.Args[0]
	args := ctx.Flags.String("args")
	if len(args) != 0 {
		args = cmdPath + " " + args
	}
	output := ctx.Flags.Bool("output")
	data, _ := proto.Marshal(&sliverpb.ExecuteReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     cmdPath,
		Args:     strings.Split(args, " "),
		Output:   output,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgExecuteReq,
		Data: data,
	}, defaultTimeout)

	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	execResp := &sliverpb.Execute{}
	err := proto.Unmarshal(resp.Data, execResp)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if execResp.Error != "" {
		fmt.Printf(Warn+"Error: %s\n", execResp.Error)
		return
	}
	if output {
		fmt.Printf(Info+"Output:\n%s", execResp.Result)
	}
}
