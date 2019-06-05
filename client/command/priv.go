package command

import (
	"fmt"

	"github.com/bishopfox/sliver/client/spin"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func impersonate(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	username := ctx.Flags.String("username")
	process := ctx.Flags.String("process")
	arguments := ctx.Flags.String("args")

	if username == "" {
		fmt.Printf(Warn + "please specify a username\n")
		return
	}

	if process == "" {
		fmt.Printf(Warn + "please specify a process path\n")
	}

	impersonate, err := runProcessAsUser(username, process, arguments, rpc)
	if err != nil {
		fmt.Printf(err.Error())
		return
	}
	if impersonate.Output != "" {
		fmt.Printf(Info+"Sucessfully ran %s %s on %s\n", process, arguments, ActiveSliver.Sliver.Name)
	}
}

func getsystem(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	go spin.Until("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)
	data, _ := proto.Marshal(&clientpb.GetSystemReq{
		SliverID: ActiveSliver.Sliver.ID,
		Config:   config,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgGetSystemReq,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}
	gsResp := &sliverpb.GetSystem{}
	err := proto.Unmarshal(resp.Data, gsResp)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if gsResp.Output != "" {
		fmt.Printf("\n"+Warn+"Error: %s\n", gsResp.Output)
		return
	}
	fmt.Printf("\n" + Info + "A new SYSTEM session should pop soon...\n")
}

func elevate(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	ctrl := make(chan bool)
	go spin.Until("Starting a new sliver session...", ctrl)
	data, _ := proto.Marshal(&sliverpb.ElevateReq{SliverID: ActiveSliver.Sliver.ID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgElevate,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}
	elevate := &sliverpb.Elevate{}
	err := proto.Unmarshal(resp.Data, elevate)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if !elevate.Success {
		fmt.Printf(Warn+"Elevation failed: %s\n", elevate.Err)
		return
	}
	fmt.Printf(Info + "Elevation successful, a new sliver session should pop soon.")
}

// Utility functions
func runProcessAsUser(username, process, arguments string, rpc RPCServer) (impersonate *sliverpb.Impersonate, err error) {
	data, _ := proto.Marshal(&sliverpb.ImpersonateReq{
		Username: username,
		Process:  process,
		Args:     arguments,
		SliverID: ActiveSliver.Sliver.ID,
	})

	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgImpersonate,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		err = fmt.Errorf(Warn+"Error: %s", resp.Err)
		return
	}
	impersonate = &sliverpb.Impersonate{}
	err = proto.Unmarshal(resp.Data, impersonate)
	if err != nil {
		err = fmt.Errorf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	return
}
