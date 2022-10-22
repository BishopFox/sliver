package powershell

import (
	b64 "encoding/base64"
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
	cmd := strings.Join(command[:], " ")
	sEnc := b64.StdEncoding.EncodeToString([]byte(cmd))

	con.App.RunCommand(strings.Split("execute-assembly -i /home/kali/Scaricati/PS.exe "+sEnc, " "))

}
