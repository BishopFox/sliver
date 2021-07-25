package exec

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// SpawnDllCmd - Spawn execution of a DLL on the remote system
func SpawnDllCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.Get()
	if session == nil {
		return
	}
	dllArgs := strings.Join(ctx.Args.StringList("arguments"), " ")
	binPath := ctx.Args.String("filepath")
	processName := ctx.Flags.String("process")
	exportName := ctx.Flags.String("export")

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	spawndll, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
		Data:        binData,
		ProcessName: processName,
		Args:        dllArgs,
		EntryPoint:  exportName,
		Request:     con.ActiveSession.Request(ctx),
		Kill:        !ctx.Flags.Bool("keep-alive"),
	})

	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	ctrl <- true
	<-ctrl
	if spawndll.GetResponse().GetErr() != "" {
		con.PrintErrorf("Error: %s\n", spawndll.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	con.PrintInfof("Output:\n%s", spawndll.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(spawndll.GetResult()))
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}
