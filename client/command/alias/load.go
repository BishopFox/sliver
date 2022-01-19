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
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

const (
	defaultTimeout = 60

	ManifestFileName = "alias.json"

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

type AliasFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type AliasManifest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	CommandName    string `json:"command_name"`
	OriginalAuthor string `json:"original_author"`
	RepoURL        string `json:"repo_url"`
	Help           string `json:"help"`
	LongHelp       string `json:"long_help"`

	Entrypoint   string       `json:"entrypoint"`
	AllowArgs    bool         `json:"allow_args"`
	DefaultArgs  string       `json:"default_args"`
	Files        []*AliasFile `json:"files"`
	IsReflective bool         `json:"is_reflective"`
	IsAssembly   bool         `json:"is_assembly"`

	RootPath string `json:"-"`
}

func (ec *AliasManifest) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

func (a *AliasManifest) getFileForTarget(cmdName string, targetOS string, targetArch string) (string, error) {
	var filePath string
	var err error
	for _, ef := range a.Files {
		if targetOS == ef.OS && targetArch == ef.Arch {
			filePath = ef.Path
			break
		}
	}
	if filePath == "" {
		err = fmt.Errorf("no alias file found for %s/%s", targetOS, targetArch)
	}
	return filePath, err
}

// AliasesLoadCmd - Locally load a alias into the Sliver shell.
func AliasesLoadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

	// parse it
	manifestPath := filepath.Join(dirPath, ManifestFileName)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	alias, err := ParseAliasManifest(data)
	if err != nil {
		return nil, err
	}
	alias.RootPath = filepath.Dir(manifestPath)
	// for each alias command, add a new app command

	// do not add if the command already exists
	if cmdExists(alias.CommandName, con.App) {
		return nil, fmt.Errorf("'%s' command already exists", alias.CommandName)
	}

	helpMsg := fmt.Sprintf("[%s] %s", alias.Name, alias.Help)
	addAliasCmd := &grumble.Command{
		Name:     alias.Name,
		Help:     helpMsg,
		LongHelp: help.FormatHelpTmpl(alias.LongHelp),
		Run: func(extCtx *grumble.Context) error {
			con.Println()
			runAliasCommand(extCtx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			if alias.IsAssembly {
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

func ParseAliasManifest(data []byte) (*AliasManifest, error) {
	// parse it
	alias := &AliasManifest{}
	err := json.Unmarshal(data, alias)
	if err != nil {
		return nil, err
	}
	if alias.Name == "" {
		return nil, fmt.Errorf("missing alias name in manifest")
	}
	if alias.CommandName == "" {
		return nil, fmt.Errorf("missing command.name in alias manifest")
	}
	if alias.Help == "" {
		return nil, fmt.Errorf("missing command.help in alias manifest")
	}

	for _, aliasFile := range alias.Files {
		if aliasFile.OS == "" {
			return nil, fmt.Errorf("missing command.files.os in alias manifest")
		}
		aliasFile.OS = strings.ToLower(aliasFile.OS)
		if aliasFile.Arch == "" {
			return nil, fmt.Errorf("missing command.files.arch in alias manifest")
		}
		aliasFile.Arch = strings.ToLower(aliasFile.Arch)
		aliasFile.Path = util.ResolvePath(aliasFile.Path)
		if aliasFile.Path == "" || aliasFile.Path == "/" {
			return nil, fmt.Errorf("missing command.files.path in alias manifest")
		}
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
	if len(alias.DefaultArgs) != 0 && len(args) == 0 {
		extArgs = alias.DefaultArgs
	} else {
		extArgs = strings.Join(args, " ")
	}
	entryPoint := alias.Entrypoint
	processName := ctx.Flags.String("process")
	if processName == "" {
		processName, err = alias.getDefaultProcess(session.GetOS())
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
	if alias.IsAssembly {
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
	} else if alias.IsReflective {
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		con.SpinUntil(msg, ctrl)
		spawnDllResp, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
			Request:     con.ActiveTarget.Request(ctx),
			Args:        strings.Trim(extArgs, " "),
			Data:        binData,
			ProcessName: processName,
			EntryPoint:  alias.Entrypoint,
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
