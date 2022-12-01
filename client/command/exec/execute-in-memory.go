package exec

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

// ExecuteInMemoryCmd - Function to execute an ELF file in memory using the ExecuteInMemory RPC
func ExecuteInMemoryCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var (
		remotePath string
		elfBytes   []byte
		err        error
	)
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	// check beacon or session OS
	if session != nil {
		if session.OS != "linux" {
			con.PrintErrorf("This command is only available on Linux")
			return
		}
	} else if beacon != nil {
		if beacon.OS != "linux" {
			con.PrintErrorf("This command is only available on Linux")
			return
		}
	}

	elfPath := ctx.Args.String("filepath")
	elfArgs := ctx.Args.StringList("arguments")
	elfRemote := ctx.Flags.Bool("remote")
	hostname := getHostname(session, beacon)

	if elfRemote {
		remotePath = elfPath
		elfArgs = append([]string{elfPath}, elfArgs...)
	} else {
		elfBytes, err = os.ReadFile(elfPath)
		if err != nil {
			con.PrintErrorf("%s", err.Error())
			return
		}
		// Don't leak client info to the Implant
		cmdPath := filepath.Base(elfPath)
		elfArgs = append([]string{cmdPath}, elfArgs...)
	}
	ctrl := make(chan bool)
	con.SpinUntil("Executing ELF ...", ctrl)
	execInMem, err := con.Rpc.ExecuteInMemory(context.Background(), &sliverpb.ExecuteInMemoryReq{
		Data:    elfBytes,
		Args:    elfArgs,
		Path:    remotePath,
		Request: con.ActiveTarget.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}
	if execInMem.Response != nil && execInMem.Response.Async {
		con.AddBeaconCallback(execInMem.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execInMem)
			if err != nil {
				con.PrintErrorf("%s", err.Error())
				return
			}
			HandleExecuteInMemoryResponse(execInMem, elfPath, hostname, ctx, con)
		})
		if execInMem.Response != nil && execInMem.Response.Err != "" {
			con.PrintErrorf("%s", execInMem.Response.Err)
			return
		}
	}
	if execInMem.Response != nil && execInMem.Response.Err != "" {
		con.PrintErrorf("Error: %s", execInMem.Response.Err)
		return
	}
	HandleExecuteInMemoryResponse(execInMem, elfPath, hostname, ctx, con)
}

func HandleExecuteInMemoryResponse(exec *sliverpb.Execute, cmdPath string, hostName string, ctx *grumble.Context, con *console.SliverConsoleClient) {
	var lootedOutput []byte
	saveLoot := ctx.Flags.Bool("loot")
	saveOutput := ctx.Flags.Bool("save")
	lootName := ctx.Flags.String("name")

	if saveLoot || saveOutput {
		// stdout and stderr are combined implant side
		lootedOutput = combineCommandOutput(exec, true, false)
	}
	if saveLoot {
		LootExecute(lootedOutput, lootName, ctx.Command.Name, cmdPath, hostName, con)
	}

	if saveOutput {
		SaveExecutionOutput(string(lootedOutput), ctx.Command.Name, hostName, con)
	}
	PrintExecute(exec, ctx, con)
}
