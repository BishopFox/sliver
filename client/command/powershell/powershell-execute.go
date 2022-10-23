package powershell

import (
	b64 "encoding/base64"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// PowerShellImportCmd - Import powershell script
func PowerShellExecuteCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	command := ctx.Args.StringList("command")
	etwBypass := ctx.Flags.Bool("etw-bypass")
	amsiBypass := ctx.Flags.Bool("amsi-bypass")
	timeout := ctx.Flags.Int("timeout")

	cmd := strings.Join(command[:], " ")
	sEnc := b64.StdEncoding.EncodeToString([]byte(cmd))
	sliverCommand := "execute-assembly -i"

	if etwBypass {
		sliverCommand += " -E"
	}
	if amsiBypass {
		sliverCommand += " -M"
	}

	con.App.RunCommand(strings.Split(fmt.Sprintf("%s -t %d %s %s", sliverCommand, timeout, PSpath, sEnc), " "))

}
