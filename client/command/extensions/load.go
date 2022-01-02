package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

const (
	defaultTimeout = 60

	ManifestFileName = "extension.json"
)

var loadedExtensions = map[string]*ExtensionManifest{}

type ExtensionManifest struct {
	Name            string               `json:"name"`
	CommandName     string               `json:"command_name"`
	Version         string               `json:"version"`
	ExtensionAuthor string               `json:"extension_author"`
	OriginalAuthor  string               `json:"original_author"`
	RepoURL         string               `json:"repo_url"`
	Help            string               `json:"help"`
	Files           []*extensionFile     `json:"files"`
	Arguments       []*extensionArgument `json:"arguments"`
	Entrypoint      string               `json:"entrypoint"`
	DependsOn       string               `json:"depends_on"`
	Init            string               `json:"init"`

	RootPath string `json:"-"`
}

type extensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type extensionArgument struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"desc"`
	Optional bool   `json:"optional"`
}

func (e *ExtensionManifest) getFileForTarget(cmdName string, targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range e.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = path.Join(assets.GetExtensionsDir(), e.CommandName, extFile.Path)
			break
		}
	}
	if filePath == "" {
		err := fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
		return "", err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = fmt.Errorf("extension file not found: %s", filePath)
		return "", err
	}
	return filePath, nil
}

// ExtensionLoadCmd - Load extension command
func ExtensionLoadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	dirPath := ctx.Args.String("dir-path")
	extCmd, err := LoadExtensionManifest(filepath.Join(dirPath, ManifestFileName))
	if err != nil {
		return
	}
	// do not add if the command already exists
	if CmdExists(extCmd.CommandName, con.App) {
		con.PrintErrorf("%s command already exists\n", extCmd.CommandName)
		return
	}
	ExtensionRegisterCommand(extCmd, con)
	con.PrintInfof("Added %s command: %s\n", extCmd.CommandName, extCmd.Help)
}

// ParseExtensions - Parse extension files
func LoadExtensionManifest(manifestPath string) (*ExtensionManifest, error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	extManifest, err := ParseExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	extManifest.RootPath = filepath.Dir(manifestPath)
	loadedExtensions[extManifest.CommandName] = extManifest
	return extManifest, nil
}

// ParseExtensionManifest - Parse extension manifest from buffer
func ParseExtensionManifest(data []byte) (*ExtensionManifest, error) {
	extManifest := &ExtensionManifest{}
	err := json.Unmarshal(data, &extManifest)
	if err != nil {
		return nil, err
	}
	if extManifest.Name == "" {
		return nil, errors.New("missing `name` field in extension manifest")
	}
	if extManifest.CommandName == "" {
		return nil, errors.New("missing `command_name` field in extension manifest")
	}
	if len(extManifest.Files) == 0 {
		return nil, errors.New("missing `files` field in extension manifest")
	}
	for _, extFiles := range extManifest.Files {
		if extFiles.OS == "" {
			return nil, errors.New("missing `files.os` field in extension manifest")
		}
		if extFiles.Arch == "" {
			return nil, errors.New("missing `files.arch` field in extension manifest")
		}
		extFiles.Path = util.ResolvePath(extFiles.Path)
		if extFiles.Path == "" || extFiles.Path == "/" {
			return nil, errors.New("missing `files.path` field in extension manifest")
		}
		extFiles.OS = strings.ToLower(extFiles.OS)
		extFiles.Arch = strings.ToLower(extFiles.Arch)
	}
	if extManifest.Help == "" {
		return nil, errors.New("missing `help` field in extension manifest")
	}
	return extManifest, nil
}

// ExtensionRegisterCommand - Register a new extension command
func ExtensionRegisterCommand(extCmd *ExtensionManifest, con *console.SliverConsoleClient) {
	loadedExtensions[extCmd.CommandName] = extCmd
	helpMsg := extCmd.Help
	extensionCmd := &grumble.Command{
		Name: extCmd.CommandName,
		Help: helpMsg,
		Run: func(extCtx *grumble.Context) error {
			con.Println()
			runExtensionCmd(extCtx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("s", "save", false, "Save output to disk")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			if len(extCmd.Arguments) > 0 {
				// BOF specific
				for _, arg := range extCmd.Arguments {
					var (
						argFunc      func(string, string, ...grumble.ArgOption)
						defaultValue grumble.ArgOption
					)
					switch arg.Type {
					case "int", "short":
						argFunc = a.Int
						defaultValue = grumble.Default(0)
					case "string", "wstring", "file":
						argFunc = a.String
						defaultValue = grumble.Default("")
					}
					if arg.Optional {
						argFunc(arg.Name, arg.Desc, defaultValue)
					} else {
						argFunc(arg.Name, arg.Desc)
					}
				}
			} else {
				a.StringList("arguments", "arguments", grumble.Default([]string{}))
			}
		},
		HelpGroup: consts.ExtensionHelpGroup,
	}
	con.App.AddCommand(extensionCmd)
}

func loadExtension(ctx *grumble.Context, session *clientpb.Session, con *console.SliverConsoleClient, ext *ExtensionManifest) error {
	var extensionList []string
	binPath, err := ext.getFileForTarget(ctx.Command.Name, session.OS, session.Arch)
	if err != nil {
		return err
	}
	// Try to find the extension in the loaded extensions
	if len(session.Extensions) == 0 {
		extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
			Request: con.ActiveTarget.Request(ctx),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err.Error())
			return err
		}
		if extList.Response != nil && extList.Response.Err != "" {
			return errors.New(extList.Response.Err)
		}
		extensionList = extList.Names
	} else {
		extensionList = session.Extensions
	}
	depLoaded := false
	for _, extName := range extensionList {
		if !depLoaded && extName == ext.DependsOn {
			depLoaded = true
		}
		if ext.Name == extName {
			return nil
		}
	}
	// Extension not found, let's load it
	if filepath.Ext(binPath) == ".o" {
		// BOFs are not loaded by the DLL loader, but we make sure the loader itself is loaded
		// Auto load the coff loader if we have it
		if !depLoaded {
			if errLoad := loadDep(session, con, ctx, ext.DependsOn); errLoad != nil {
				return errLoad
			}
		}
		return nil
	}
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return err
	}
	if errRegister := registerExtension(con, ext, binData, session, ctx); errRegister != nil {
		return errRegister
	}
	// Update session info
	if filepath.Ext(binPath) != ".o" {
		// Don't update session for BOFs, we don't cache them
		// on the implant side (yet)
		// update the Session info
		_, err = con.Rpc.UpdateSession(context.Background(), &clientpb.UpdateSession{
			Extensions: extensionList,
			SessionID:  session.ID,
		})
		if err != nil {
			con.PrintErrorf("%s\n", err.Error())
			return err
		}
	}
	return nil
}

func registerExtension(con *console.SliverConsoleClient, ext *ExtensionManifest, binData []byte, session *clientpb.Session, ctx *grumble.Context) error {
	registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    ext.CommandName,
		Data:    binData,
		OS:      session.OS,
		Init:    ext.Init,
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return err
	}
	if registerResp.Response != nil && registerResp.Response.Err != "" {
		return errors.New(registerResp.Response.Err)
	}
	return nil
}

func loadDep(session *clientpb.Session, con *console.SliverConsoleClient, ctx *grumble.Context, depName string) error {
	depExt, f := loadedExtensions[depName]
	if f {
		depBinPath, err := depExt.getFileForTarget(depExt.CommandName, session.OS, session.Arch)
		if err != nil {
			return err
		}
		depBinData, err := ioutil.ReadFile(depBinPath)
		if err != nil {
			return err
		}
		return registerExtension(con, depExt, depBinData, session, ctx)
	}
	return fmt.Errorf("missing dependency %s", depName)
}

func runExtensionCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var (
		callExtension *sliverpb.CallExtension
		err           error
		extensionArgs []byte
		extName       string
		entryPoint    string
	)
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	ext, ok := loadedExtensions[ctx.Command.Name]
	if !ok {
		con.PrintErrorf("No extension command found for `%s` command\n", ctx.Command.Name)
		return
	}

	if err = loadExtension(ctx, session, con, ext); err != nil {
		con.PrintErrorf("Could not load extension: %s\n", err)
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
	binPath, err := ext.getFileForTarget(ctx.Command.Name, session.OS, session.Arch)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	isBOF := filepath.Ext(binPath) == ".o"

	// BOFs (Beacon Object Files) are a specific kind of extensions
	// that require another extension (a COFF loader) to be present.
	// BOFs also have strongly typed arguments that need to be parsed in the proper way.
	// This block will pack both the BOF data and its arguments into a single buffer that
	// the loader will extract and load.
	if isBOF {
		// Beacon Object File -- requires a COFF loader
		extensionArgs, err = getBOFArgs(ctx, binPath, ext)
		if err != nil {
			con.PrintErrorf("Error: %s\n", err)
			return
		}
		extName = ext.DependsOn
		entryPoint = loadedExtensions[extName].Entrypoint // should exist at this point
	} else {
		// Regular DLL
		extArgs := strings.Join(ctx.Args.StringList("arguments"), " ")
		extensionArgs = []byte(extArgs)
		extName = ext.CommandName
		entryPoint = ext.Entrypoint
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Executing %s ...", ctx.Command.Name)
	con.SpinUntil(msg, ctrl)
	callExtension, err = con.Rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:    extName,
		Export:  entryPoint,
		Args:    extensionArgs,
		Request: con.ActiveTarget.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	if callExtension.Response != nil && callExtension.Response.Err != "" {
		con.PrintErrorf("%s\n", callExtension.Response.Err)
		return
	}
	con.PrintInfof("Successfully executed %s\n", extName)
	if len(callExtension.Output) > 0 {
		con.PrintInfof("Got output:\n%s", string(callExtension.Output))
		if outFilePath != nil {
			outFilePath.Write(callExtension.Output)
			con.PrintInfof("Output saved to %s\n", outFilePath.Name())
		}
		con.Println()
	}
}

func getBOFArgs(ctx *grumble.Context, binPath string, ext *ExtensionManifest) ([]byte, error) {
	var extensionArgs []byte
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return nil, err
	}
	argsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}
	// Parse BOF arguments from grumble
	for _, arg := range ext.Arguments {
		switch arg.Type {
		case "int":
			val := ctx.Args.Int(arg.Name)
			err = argsBuffer.AddInt(uint32(val))
			if err != nil {
				return nil, err
			}
		case "short":
			val := ctx.Args.Int(arg.Name)
			err = argsBuffer.AddShort(uint16(val))
			if err != nil {
				return nil, err
			}
		case "string":
			val := ctx.Args.String(arg.Name)
			err = argsBuffer.AddString(val)
			if err != nil {
				return nil, err
			}
		case "wstring":
			val := ctx.Args.String(arg.Name)
			err = argsBuffer.AddWString(val)
			if err != nil {
				return nil, err
			}
		// Adding support for filepaths so we can
		// send binary data like shellcodes to BOFs
		case "file":
			val := ctx.Args.String(arg.Name)
			data, err := ioutil.ReadFile(val)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddData(data)
			if err != nil {
				return nil, err
			}
		}
	}
	parsedArgs, err := argsBuffer.GetBuffer()
	if err != nil {
		return nil, err
	}
	// Now build the extension's argument buffer
	extensionArgsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}
	err = extensionArgsBuffer.AddString(ext.Entrypoint)
	if err != nil {
		return nil, err
	}
	err = extensionArgsBuffer.AddData(binData)
	if err != nil {
		return nil, err
	}
	err = extensionArgsBuffer.AddData(parsedArgs)
	if err != nil {
		return nil, err
	}
	extensionArgs, err = extensionArgsBuffer.GetBuffer()
	if err != nil {
		return nil, err
	}
	return extensionArgs, nil

}

// CmdExists - checks if a command exists
func CmdExists(name string, app *grumble.App) bool {
	for _, c := range app.Commands().All() {
		if name == c.Name {
			return true
		}
	}
	return false
}
