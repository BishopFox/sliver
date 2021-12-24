package alias

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

var (
	// alias name -> manifest/command
	loadedAliases = map[string]*loadedAlias{}

	defaultHostProc = map[string]string{
		"windows": windowsDefaultHostProc,
		"linux":   windowsDefaultHostProc,
		"darwin":  macosDefaultHostProc,
	}
)

// Ties the manifest struct to the command struct
type loadedAlias struct {
	Manifest *AliasManifest
	Command  *grumble.Command
}

type binFiles struct {
	Ext64Path string `json:"x64"`
	Ext32Path string `json:"x86"`
}

type AliasFile struct {
	OS    string   `json:"os"`
	Files binFiles `json:"files"`
}

type AliasCommand struct {
	Name         string      `json:"name"`
	Entrypoint   string      `json:"entrypoint"`
	Help         string      `json:"help"`
	LongHelp     string      `json:"long_help"`
	AllowArgs    bool        `json:"allow_args"`
	DefaultArgs  string      `json:"default_args"`
	AliasFiles   []AliasFile `json:"files"`
	IsReflective bool        `json:"is_reflective"`
	IsAssembly   bool        `json:"is_assembly"`
}

func (ec *AliasCommand) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

type AliasManifest struct {
	Name     string       `json:"name"`
	Command  AliasCommand `json:"command"`
	RootPath string
}

func (a *AliasManifest) getFileForTarget(cmdName string, targetOS string, targetArch string) (string, error) {
	var filePath string
	var err error
	for _, ef := range a.Command.AliasFiles {
		if targetOS == ef.OS {
			switch targetArch {
			case "x86":
				filePath = filepath.Join(a.RootPath, filepath.Base(ef.Files.Ext32Path))
			case "x64":
				filePath = filepath.Join(a.RootPath, filepath.Base(ef.Files.Ext64Path))
			default:
				filePath = filepath.Join(a.RootPath, filepath.Base(ef.Files.Ext64Path))
			}
		}
	}
	if filePath == "" {
		err = fmt.Errorf("no alias file found for %s/%s", targetOS, targetArch)
	}
	return filePath, err
}

// LoadAliasCmd - Locally load a alias into the Sliver shell.
func LoadAliasCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	dirPath := ctx.Args.String("dir-path")
	alias, err := LoadAlias(dirPath, con)
	if err != nil {
		con.PrintErrorf("Failed to load alias: %s\n", err)
	} else {
		con.PrintInfof("%s alias has been loaded\n", alias.Name)
	}
}

// LoadAlias - Load an alias into the Sliver shell from a given directory
func LoadAlias(dirPath string, con *console.SliverConsoleClient) (*AliasManifest, error) {
	// retrieve alias manifest
	var err error
	dirPath, err = filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(dirPath, "manifest.json")
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}
	// parse it
	alias := &AliasManifest{}
	err = json.Unmarshal(jsonBytes, alias)
	if err != nil {
		return nil, err
	}
	alias.RootPath = dirPath
	// for each alias command, add a new app command

	// do not add if the command already exists
	if cmdExists(alias.Name, con.App) {
		return nil, fmt.Errorf("'%s' command already exists", alias.Name)
	}

	helpMsg := fmt.Sprintf("[%s] %s", alias.Name, alias.Command.Help)
	addAliasCmd := &grumble.Command{
		Name:     alias.Name,
		Help:     helpMsg,
		LongHelp: help.FormatHelpTmpl(alias.Command.LongHelp),
		Run: func(extCtx *grumble.Context) error {
			con.Println()
			runAliasCommand(extCtx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			if alias.Command.IsAssembly {
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
		HelpGroup: consts.AliasHelpGroup,
	}
	con.App.AddCommand(addAliasCmd)

	// Have to use a global map here, as passing the aliasCmd
	// either by value or by ref fucks things up
	loadedAliases[alias.Name] = &loadedAlias{
		Manifest: alias,
		Command:  addAliasCmd,
	}

	return alias, nil
}

func runAliasCommand(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	loadedAlias, ok := loadedAliases[ctx.Command.Name]
	if !ok {
		con.PrintErrorf("No alias found for `%s` command\n", ctx.Command.Name)
		return
	}
	alias := loadedAlias.Manifest
	binPath, err := alias.getFileForTarget(ctx.Command.Name, session.GetOS(), session.GetArch())
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	args := ctx.Args.StringList("arguments")
	var extArgs string
	if len(alias.Command.DefaultArgs) != 0 && len(args) == 0 {
		extArgs = alias.Command.DefaultArgs
	} else {
		extArgs = strings.Join(args, " ")
	}
	entryPoint := alias.Command.Entrypoint
	processName := ctx.Flags.String("process")
	if processName == "" {
		processName, err = alias.Command.getDefaultProcess(session.GetOS())
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
	if alias.Command.IsAssembly {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		executeAssemblyResp, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
			Request:   con.ActiveTarget.Request(ctx),
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
	} else if alias.Command.IsReflective {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		spawnDllResp, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
			Request:     con.ActiveTarget.Request(ctx),
			Args:        strings.Trim(extArgs, " "),
			Data:        binData,
			ProcessName: processName,
			EntryPoint:  alias.Command.Entrypoint,
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
			Request:     con.ActiveTarget.Request(ctx),
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
