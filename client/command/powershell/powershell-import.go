package powershell

import (
	b64 "encoding/base64"
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
	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}
	sEnc := b64.StdEncoding.EncodeToString([]byte(scriptBytes))
	con.App.RunCommand(strings.Split("execute-assembly -i /home/kali/Scaricati/PS.exe loadmodule"+sEnc, " "))

}
