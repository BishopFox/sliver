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
	"strconv"
	"strings"

	appConsole "github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
)

const (
	defaultTimeout = 60

	// ManifestFileName - Extension manifest file name
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
	LongHelp        string               `json:"long_help"`
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
func ExtensionLoadCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	dirPath := args[0]
	// dirPath := ctx.Args.String("dir-path")
	extCmd, err := LoadExtensionManifest(filepath.Join(dirPath, ManifestFileName))
	if err != nil {
		return
	}
	// do not add if the command already exists
	sliverMenu := con.App.Menu("implant")
	if CmdExists(extCmd.CommandName, sliverMenu.Command) {
		con.PrintErrorf("%s command already exists\n", extCmd.CommandName)
		return
	}
	ExtensionRegisterCommand(extCmd, cmd.Root(), con)
	con.PrintInfof("Added %s command: %s\n", extCmd.CommandName, extCmd.Help)
}

// LoadExtensionManifest - Parse extension files
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
func ExtensionRegisterCommand(extCmd *ExtensionManifest, cmd *cobra.Command, con *console.SliverConsoleClient) {
	if errInvalidArgs := checkExtensionArgs(extCmd); errInvalidArgs != nil {
		con.PrintErrorf(errInvalidArgs.Error())
		return
	}

	loadedExtensions[extCmd.CommandName] = extCmd
	helpMsg := extCmd.Help

	// Command
	extensionCmd := &cobra.Command{
		Use:   extCmd.CommandName,
		Short: helpMsg,
		Long:  help.FormatHelpTmpl(extCmd.LongHelp),
		Run: func(cmd *cobra.Command, args []string) {
			runExtensionCmd(cmd, con, args)
		},
		GroupID:     consts.ExtensionHelpGroup,
		Annotations: makeCommandPlatformFilters(extCmd),
	}

	// Flags
	f := pflag.NewFlagSet(extCmd.Name, pflag.ContinueOnError)
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	extensionCmd.Flags().AddFlagSet(f)
	extensionCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

	// Completions
	comps := carapace.Gen(extensionCmd)
	makeExtensionArgCompleter(extCmd, cmd, comps)

	cmd.AddCommand(extensionCmd)
}

func loadExtension(goos string, goarch string, checkCache bool, ext *ExtensionManifest, cmd *cobra.Command, con *console.SliverConsoleClient) error {
	var extensionList []string
	binPath, err := ext.getFileForTarget(cmd.Name(), goos, goarch)
	if err != nil {
		return err
	}

	// Try to find the extension in the loaded extensions
	if checkCache {
		extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
			Request: con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("List extensions error: %s\n", err.Error())
			return err
		}
		if extList.Response != nil && extList.Response.Err != "" {
			return errors.New(extList.Response.Err)
		}
		extensionList = extList.Names
	}
	depLoaded := false
	for _, extName := range extensionList {
		if !depLoaded && extName == ext.DependsOn {
			depLoaded = true
		}
		if ext.CommandName == extName {
			return nil
		}
	}
	// Extension not found, let's load it
	if filepath.Ext(binPath) == ".o" {
		// BOFs are not loaded by the DLL loader, but we make sure the loader itself is loaded
		// Auto load the coff loader if we have it
		if !depLoaded {
			if errLoad := loadDep(goos, goarch, ext.DependsOn, cmd, con); errLoad != nil {
				return errLoad
			}
		}
		return nil
	}
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return err
	}
	if errRegister := registerExtension(goos, ext, binData, cmd, con); errRegister != nil {
		return errRegister
	}
	return nil
}

func registerExtension(goos string, ext *ExtensionManifest, binData []byte, cmd *cobra.Command, con *console.SliverConsoleClient) error {
	registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    ext.CommandName,
		Data:    binData,
		OS:      goos,
		Init:    ext.Init,
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		return err
	}
	if registerResp.Response != nil && registerResp.Response.Err != "" {
		return errors.New(registerResp.Response.Err)
	}
	return nil
}

func loadDep(goos string, goarch string, depName string, cmd *cobra.Command, con *console.SliverConsoleClient) error {
	depExt, ok := loadedExtensions[depName]
	if ok {
		depBinPath, err := depExt.getFileForTarget(depExt.CommandName, goos, goarch)
		if err != nil {
			return err
		}
		depBinData, err := ioutil.ReadFile(depBinPath)
		if err != nil {
			return err
		}
		return registerExtension(goos, depExt, depBinData, cmd, con)
	}
	return fmt.Errorf("missing dependency %s", depName)
}

func runExtensionCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	var (
		err           error
		extensionArgs []byte
		extName       string
		entryPoint    string
	)
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	var goos string
	var goarch string
	if session != nil {
		goos = session.OS
		goarch = session.Arch
	} else {
		goos = beacon.OS
		goarch = beacon.Arch
	}

	ext, ok := loadedExtensions[cmd.Name()]
	if !ok {
		con.PrintErrorf("No extension command found for `%s` command\n", cmd.Name())
		return
	}

	checkCache := session != nil
	if err = loadExtension(goos, goarch, checkCache, ext, cmd, con); err != nil {
		con.PrintErrorf("Could not load extension: %s\n", err)
		return
	}

	binPath, err := ext.getFileForTarget(cmd.Name(), goos, goarch)
	if err != nil {
		con.PrintErrorf("Failed to read extension file: %s\n", err)
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
		extensionArgs, err = getBOFArgs(cmd, args, binPath, ext)
		if err != nil {
			con.PrintErrorf("BOF args error: %s\n", err)
			return
		}
		extName = ext.DependsOn
		entryPoint = loadedExtensions[extName].Entrypoint // should exist at this point
	} else {
		// Regular DLL
		extArgs := strings.Join(args, " ")
		extensionArgs = []byte(extArgs)
		extName = ext.CommandName
		entryPoint = ext.Entrypoint
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Executing %s ...", cmd.Name())
	con.SpinUntil(msg, ctrl)
	callExtResp, err := con.Rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:    extName,
		Export:  entryPoint,
		Args:    extensionArgs,
		Request: con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Call extension error: %s\n", err.Error())
		return
	}

	if callExtResp.Response != nil && callExtResp.Response.Async {
		con.AddBeaconCallback(callExtResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, callExtResp)
			if err != nil {
				con.PrintErrorf("Failed to decode call ext response %s\n", err)
				return
			}
			PrintExtOutput(extName, ext.CommandName, callExtResp, con)
		})
		con.PrintAsyncResponse(callExtResp.Response)
	} else {
		PrintExtOutput(extName, ext.CommandName, callExtResp, con)
	}
}

// PrintExtOutput - Print the ext execution output
func PrintExtOutput(extName string, commandName string, callExtension *sliverpb.CallExtension, con *console.SliverConsoleClient) {
	if extName == commandName {
		con.PrintInfof("Successfully executed %s", extName)
	} else {
		con.PrintInfof("Successfully executed %s (%s)", commandName, extName)
	}
	if 0 < len(string(callExtension.Output)) {
		con.PrintInfof("Got output:\n%s", callExtension.Output)
	}
	if callExtension.Response != nil && callExtension.Response.Err != "" {
		con.PrintErrorf(callExtension.Response.Err)
		return
	}
}

func getBOFArgs(cmd *cobra.Command, args []string, binPath string, ext *ExtensionManifest) ([]byte, error) {
	var extensionArgs []byte
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return nil, err
	}
	argsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	// Parse BOF arguments from grumble
	missingRequiredArgs := make([]string, 0)

	for _, arg := range ext.Arguments {
		// If we don't have any positional words left to consume,
		// add the remaining required extension arguments in the
		// error message.
		if len(args) == 0 {
			if !arg.Optional {
				missingRequiredArgs = append(missingRequiredArgs, "`"+arg.Name+"`")
			}
			continue
		}

		// Else pop a word from the list
		word := args[0]
		args = args[1:]

		switch arg.Type {
		case "integer":
			fallthrough
		case "int":
			val, err := strconv.Atoi(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddInt(uint32(val))
			if err != nil {
				return nil, err
			}
		case "short":
			val, err := strconv.Atoi(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddShort(uint16(val))
			if err != nil {
				return nil, err
			}
		case "string":
			err = argsBuffer.AddString(word)
			if err != nil {
				return nil, err
			}
		case "wstring":
			err = argsBuffer.AddWString(word)
			if err != nil {
				return nil, err
			}
		// Adding support for filepaths so we can
		// send binary data like shellcodes to BOFs
		case "file":
			data, err := ioutil.ReadFile(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddData(data)
			if err != nil {
				return nil, err
			}
		}
	}

	// Return if we have missing required arguments
	if len(missingRequiredArgs) > 0 {
		return nil, fmt.Errorf("required arguments %s were not provided", strings.Join(missingRequiredArgs, ", "))
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
func CmdExists(name string, cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

// makeExtensionArgParser builds the valid positional arguments cobra handler for the extension.
func checkExtensionArgs(extCmd *ExtensionManifest) error {
	if 0 < len(extCmd.Arguments) {
		for _, arg := range extCmd.Arguments {
			switch arg.Type {
			case "int", "integer", "short":
			case "string", "wstring", "file":
			default:
				return fmt.Errorf("invalid argument type: %s", arg.Type)
			}
		}
	}

	return nil
}

// makeExtensionArgCompleter builds the positional arguments completer for the extension.
func makeExtensionArgCompleter(extCmd *ExtensionManifest, _ *cobra.Command, comps *carapace.Carapace) {
	var actions []carapace.Action

	for _, arg := range extCmd.Arguments {
		var action carapace.Action

		switch arg.Type {
		case "file":
			action = carapace.ActionFiles().Tag("extension data")
		}

		usage := fmt.Sprintf("(%s) %s", arg.Type, arg.Desc)
		if arg.Optional {
			usage += " (optional)"
		}

		actions = append(actions, action.Usage(usage))
	}

	comps.PositionalCompletion(actions...)
}

func makeCommandPlatformFilters(extCmd *ExtensionManifest) map[string]string {
	filtersOS := make(map[string]bool)
	filtersArch := make(map[string]bool)

	var all []string

	// Only add filters for architectures when there OS matters.
	for _, file := range extCmd.Files {
		filtersOS[file.OS] = true

		if filtersOS[file.OS] {
			filtersArch[file.Arch] = true
		}
	}

	for os, enabled := range filtersOS {
		if enabled {
			all = append(all, os)
		}
	}

	for arch, enabled := range filtersArch {
		if enabled {
			all = append(all, arch)
		}
	}

	if len(all) == 0 {
		return map[string]string{}
	}

	return map[string]string{
		appConsole.CommandFilterKey: strings.Join(all, ","),
	}
}
