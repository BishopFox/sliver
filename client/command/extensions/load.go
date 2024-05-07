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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/packages"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	appConsole "github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout = 60

	// ManifestFileName - Extension manifest file name.
	ManifestFileName = "extension.json"
)

var loadedExtensions = map[string]*ExtCommand{}
var loadedManifests = map[string]*ExtensionManifest{}

type ExtensionManifest_ struct {
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

type ExtensionManifest struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ExtensionAuthor string `json:"extension_author"`
	OriginalAuthor  string `json:"original_author"`
	RepoURL         string `json:"repo_url"`

	ExtCommand []*ExtCommand `json:"commands"`

	RootPath   string `json:"-"`
	ArmoryName string `json:"-"`
	ArmoryPK   string `json:"-"`
}

type ExtCommand struct {
	CommandName string                 `json:"command_name"`
	Help        string                 `json:"help"`
	LongHelp    string                 `json:"long_help"`
	Files       []*extensionFile       `json:"files"`
	Arguments   []*extensionArgument   `json:"arguments"`
	Entrypoint  string                 `json:"entrypoint"`
	DependsOn   string                 `json:"depends_on"`
	Init        string                 `json:"init"`
	Schema      *packages.OutputSchema `json:"schema"`

	Manifest *ExtensionManifest
}

//type MultiManifest []*ExtensionManifest

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

func (e *ExtCommand) getFileForTarget(targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range e.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = path.Join(assets.GetExtensionsDir(), e.Manifest.Name, extFile.Path)
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

// ExtensionLoadCmd - Load extension command.
func ExtensionLoadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	dirPath := args[0]
	// dirPath := ctx.Args.String("dir-path")
	manifest, err := LoadExtensionManifest(filepath.Join(dirPath, ManifestFileName))
	if err != nil {
		return
	}
	// do not add if the command already exists
	sliverMenu := con.App.Menu("implant")
	for _, extCmd := range manifest.ExtCommand {
		if CmdExists(extCmd.CommandName, sliverMenu.Command) {
			con.PrintErrorf("%s command already exists\n", extCmd.CommandName)
			confirm := false
			prompt := &survey.Confirm{Message: "Overwrite current command?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return
			}
		}
		ExtensionRegisterCommand(extCmd, cmd.Root(), con)
		con.PrintInfof("Added %s command: %s\n", extCmd.CommandName, extCmd.Help)
	}
}

// LoadExtensionManifest - Parse extension files.
func LoadExtensionManifest(manifestPath string) (*ExtensionManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	manifest.RootPath = filepath.Dir(manifestPath)
	for _, extManifest := range manifest.ExtCommand {
		loadedExtensions[extManifest.CommandName] = extManifest
	}
	loadedManifests[manifest.Name] = manifest

	return manifest, nil

}

func convertOldManifest(old *ExtensionManifest_) *ExtensionManifest {
	ret := &ExtensionManifest{
		Name:            old.CommandName, //treating old command name as the manifest name to avoid weird chars mostly
		Version:         old.Version,
		ExtensionAuthor: old.ExtensionAuthor,
		OriginalAuthor:  old.OriginalAuthor,
		RepoURL:         old.RepoURL,
		RootPath:        old.RootPath,
		//only one command exists in the old manifest, so we can 'confidently' create it here
		ExtCommand: []*ExtCommand{
			{
				CommandName: old.CommandName,
				DependsOn:   old.DependsOn,
				Help:        old.Help,
				LongHelp:    old.LongHelp,
				Entrypoint:  old.Entrypoint,
				Files:       old.Files,
				Arguments:   old.Arguments,
				Schema:      nil,
			},
		},
	}

	//Manifest ref is done in the parser that calls this

	return ret
}

// parseExtensionManifest - Parse extension manifest from buffer (legacy, only parses one)
func ParseExtensionManifest(data []byte) (*ExtensionManifest, error) {
	extManifest := &ExtensionManifest{}
	err := json.Unmarshal(data, &extManifest)
	if err != nil || len(extManifest.ExtCommand) == 0 { //extensions must have at least one command to be sensible
		//maybe it's an old manifest
		if err != nil {
			log.Printf("extension load error: %s", err)
		}
		oldmanifest := &ExtensionManifest_{}
		err := json.Unmarshal(data, &oldmanifest)
		if err != nil {
			//nope, just broken
			return nil, err
		}
		//yes, ok, lets jigger it to a new manifest
		extManifest = convertOldManifest(oldmanifest)
	}
	//pass ref to manifest to each command and initialize output schema if applicable
	for i := range extManifest.ExtCommand {
		command := extManifest.ExtCommand[i]
		command.Manifest = extManifest
		if command.Schema != nil {
			command.Schema.IngestColumns()
		}
	}
	return extManifest, validManifest(extManifest)
}

func validManifest(manifest *ExtensionManifest) error {
	if manifest.Name == "" {
		return errors.New("missing `name` field in extension manifest")
	}
	for _, extManifest := range manifest.ExtCommand {
		if extManifest.CommandName == "" {
			return errors.New("missing `command_name` field in extension manifest")
		}
		if len(extManifest.Files) == 0 {
			return errors.New("missing `files` field in extension manifest")
		}
		for _, extFiles := range extManifest.Files {
			if extFiles.OS == "" {
				return errors.New("missing `files.os` field in extension manifest")
			}
			if extFiles.Arch == "" {
				return errors.New("missing `files.arch` field in extension manifest")
			}
			extFiles.Path = util.ResolvePath(extFiles.Path)
			if extFiles.Path == "" || extFiles.Path == "/" {
				return errors.New("missing `files.path` field in extension manifest")
			}
			extFiles.OS = strings.ToLower(extFiles.OS)
			extFiles.Arch = strings.ToLower(extFiles.Arch)
		}
		if extManifest.Help == "" {
			return errors.New("missing `help` field in extension manifest")
		}
		if extManifest.Schema != nil {
			if !packages.IsValidSchemaType(extManifest.Schema.Name) {
				return fmt.Errorf("%s is not a valid schema type", extManifest.Schema.Name)
			}
		}
	}
	return nil
}

// ExtensionRegisterCommand - Register a new extension command
func ExtensionRegisterCommand(extCmd *ExtCommand, cmd *cobra.Command, con *console.SliverClient) {
	if errInvalidArgs := checkExtensionArgs(extCmd); errInvalidArgs != nil {
		con.PrintErrorf(errInvalidArgs.Error())
		return
	}

	loadedExtensions[extCmd.CommandName] = extCmd
	//helpMsg := extCmd.Help

	usage := strings.Builder{}
	usage.WriteString(extCmd.CommandName)
	//build usage including args
	for _, arg := range extCmd.Arguments {
		usage.WriteString(" ")
		if arg.Optional {
			usage.WriteString("[")
		}
		usage.WriteString(strings.ToUpper(arg.Name))
		if arg.Optional {
			usage.WriteString("]")
		}
	}
	longHelp := strings.Builder{}
	//prepend the help value, because otherwise I don't see where it is meant to be shown
	//build the command ref
	longHelp.WriteString("[[.Bold]]Command:[[.Normal]]")
	longHelp.WriteString(usage.String())
	longHelp.WriteString("\n")
	if len(extCmd.Help) > 0 || len(extCmd.LongHelp) > 0 {
		longHelp.WriteString("[[.Bold]]About:[[.Normal]]")
		if len(extCmd.Help) > 0 {
			longHelp.WriteString(extCmd.Help)
			longHelp.WriteString("\n")
		}
		if len(extCmd.LongHelp) > 0 {
			longHelp.WriteString(extCmd.LongHelp)
			longHelp.WriteString("\n")
		}
	}
	if len(extCmd.Arguments) > 0 {
		longHelp.WriteString("[[.Bold]]Arguments:[[.Normal]]")
	}
	//if more than 0 args specified, describe each arg at the bottom of the long help text (incase the manifest doesn't include it)
	for _, arg := range extCmd.Arguments {
		longHelp.WriteString("\n\t")
		optStr := ""
		if arg.Optional {
			optStr = "[OPTIONAL]"
		}
		aType := arg.Type
		if aType == "wstring" {
			aType = "string" //avoid confusion, as this is mostly for telling operator what to shove into the args
		}
		//idk how to make this look nice, tabs don't work especially good - maybe should use the table stuff other things do? Pls help.
		longHelp.WriteString(fmt.Sprintf("%s (%s):\t%s%s", strings.ToUpper(arg.Name), aType, optStr, arg.Desc))
	}

	// Command
	extensionCmd := &cobra.Command{
		Use: usage.String(), //extCmd.CommandName,
		//Short: helpMsg.String(), doesn't appear to be used?
		Long: help.FormatHelpTmpl(longHelp.String()),
		Run: func(cmd *cobra.Command, args []string) {
			runExtensionCmd(cmd, con, args)
		},
		GroupID:     consts.ExtensionHelpGroup,
		Annotations: makeCommandPlatformFilters(extCmd),
	}

	// Flags
	f := pflag.NewFlagSet(extCmd.CommandName, pflag.ContinueOnError)
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	extensionCmd.Flags().AddFlagSet(f)
	extensionCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

	// Completions
	comps := carapace.Gen(extensionCmd)
	makeExtensionArgCompleter(extCmd, cmd, comps)

	cmd.AddCommand(extensionCmd)
}

func loadExtension(goos string, goarch string, checkCache bool, ext *ExtCommand, cmd *cobra.Command, con *console.SliverClient) error {
	var extensionList []string
	binPath, err := ext.getFileForTarget(goos, goarch)
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
	//extensionList contains all *implant* loaded extensions. Is a sha256 sum of the relevant file.

	//get the file hash to compare against implant extensions later
	toberunfilepath, err := ext.getFileForTarget(goos, goarch)
	if err != nil {
		return err
	}
	//we need to check the dependent if it's a bof
	if ext.DependsOn != "" {
		//verify extension is in loaded list
		if extension, found := loadedExtensions[ext.DependsOn]; found {
			toberunfilepath, err = extension.getFileForTarget(goos, goarch)
			if err != nil {
				return err
			}
		} else {
			// handle error
			return fmt.Errorf("attempted to load non-existing extension: %s", ext.DependsOn)
		}
	}

	//todo, maybe cache these values somewhere
	toberunfiledata, err := os.ReadFile(toberunfilepath)
	if err != nil {
		return err
	}
	if len(toberunfiledata) == 0 {
		//read an empty file, bail out
		return errors.New("read empty extension file content")
	}
	bd := sha256.Sum256(toberunfiledata)
	toberunhash := hex.EncodeToString(bd[:])

	for _, extName := range extensionList {
		//check if extension we are trying to run (ext.CommandName or ext.DependsOn, for bofs) is already loaded
		if extName == toberunhash {
			//exists on the other side, exit early
			return nil
		}
	}
	// Extension not found, let's load it
	// BOFs are not loaded by the DLL loader, but we need to load the loader (more load more good)
	if ext.DependsOn != "" {
		return loadDep(goos, goarch, ext.DependsOn, cmd, con)
	}
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	if errRegister := registerExtension(goos, ext, binData, cmd, con); errRegister != nil {
		return errRegister
	}
	return nil
}

func registerExtension(goos string, ext *ExtCommand, binData []byte, cmd *cobra.Command, con *console.SliverClient) error {
	//set extension name to a hash of the data to avoid loading more than one instance
	bd := sha256.Sum256(binData)
	name := hex.EncodeToString(bd[:])
	sess, beac := con.ActiveTarget.GetInteractive()
	ctrl := make(chan bool)
	//first time run of an extension will require some waiting depending on the size
	if sess != nil {
		msg := fmt.Sprintf("Sending %s to implant ...", ext.CommandName)
		con.SpinUntil(msg, ctrl)
	}
	//don't block if we are in beacon mode
	if beac != nil && sess == nil {
		go func() {
			registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
				Name:    name,
				Data:    binData,
				OS:      goos,
				Init:    ext.Init,
				Request: con.ActiveTarget.Request(cmd),
			})
			if err != nil {
				con.PrintErrorf("Error registering extension: %s\n", err)
			}
			if registerResp.Response != nil && registerResp.Response.Err != "" {
				con.PrintErrorf("Error registering extension: %s\n", errors.New(registerResp.Response.Err))
			}
		}()
		return nil
	}
	//session mode (hopefully)
	registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    name,
		Data:    binData,
		OS:      goos,
		Request: con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		return err
	}
	if registerResp.Response != nil && registerResp.Response.Err != "" {
		return errors.New(registerResp.Response.Err)
	}
	return nil
}

func loadDep(goos string, goarch string, depName string, cmd *cobra.Command, con *console.SliverClient) error {
	depExt, ok := loadedExtensions[depName]
	if ok {
		depBinPath, err := depExt.getFileForTarget(goos, goarch)
		if err != nil {
			return err
		}
		depBinData, err := os.ReadFile(depBinPath)
		if err != nil {
			return err
		}
		return registerExtension(goos, depExt, depBinData, cmd, con)
	}
	return fmt.Errorf("missing dependency %s", depName)
}

func runExtensionCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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

	binPath, err := ext.getFileForTarget(goos, goarch)
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
		extensionArgs, err = getExtArgs(cmd, args, binPath, ext)
		if err != nil {
			con.PrintErrorf("ext args error: %s\n", err)
			return
		}
		extName = ext.CommandName
		entryPoint = ext.Entrypoint
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Executing %s ...", cmd.Name())
	con.SpinUntil(msg, ctrl)
	extdata, err := os.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("ext read file error: %s\n", err)
	}
	bd := sha256.Sum256(extdata)
	name := hex.EncodeToString(bd[:])
	if isBOF {
		//if we are using a bof, we are actually calling the coffloader extension - so get the file from dep ref and use that shasum

		if extension, found := loadedExtensions[ext.DependsOn]; found {
			dep, err := extension.getFileForTarget(goos, goarch)
			if err != nil {
				con.PrintErrorf("could not get file for extension %s", ext.DependsOn)
				return
			}
			depdata, err := os.ReadFile(dep)
			if err != nil {
				con.PrintErrorf("dep read file error: %s\n", err)
				return
			}
			if len(depdata) == 0 {
				con.PrintErrorf("read empty file: %s\n", dep)
				return
			}
			bd = sha256.Sum256(depdata)
			name = hex.EncodeToString(bd[:])
		} else {
			// handle error
			con.PrintErrorf("attempted to load non-existing extension: %s", ext.DependsOn)
			return
		}

	}
	callExtResp, err := con.Rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:    name,
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
			PrintExtOutput(extName, ext.CommandName, ext.Schema, callExtResp, con)
		})
		con.PrintAsyncResponse(callExtResp.Response)
	} else {
		PrintExtOutput(extName, ext.CommandName, ext.Schema, callExtResp, con)
	}
}

// PrintExtOutput - Print the ext execution output.
func PrintExtOutput(extName string, commandName string, outputSchema *packages.OutputSchema, callExtension *sliverpb.CallExtension, con *console.SliverClient) {
	if extName == commandName {
		con.PrintInfof("Successfully executed %s\n", extName)
	} else {
		con.PrintInfof("Successfully executed %s (%s)\n", commandName, extName)
	}
	if 0 < len(string(callExtension.Output)) {
		if outputSchema == nil {
			con.PrintInfof("Got output:\n%s", callExtension.Output)
		} else {
			// Get output schema
			schema := packages.GetNewPackageOutput(outputSchema.Name)
			if schema != nil {
				ingestErr := schema.IngestData(callExtension.Output, outputSchema.Columns(), outputSchema.GroupBy)
				if ingestErr != nil {
					con.PrintInfof("Got output:\n%s", callExtension.Output)
				} else {
					con.Printf("%s\n", schema.CreateTable())
				}
			} else {
				con.PrintInfof("Got output:\n%s", callExtension.Output)
			}
		}
	}
	if callExtension.Response != nil && callExtension.Response.Err != "" {
		con.PrintErrorf(callExtension.Response.Err)
		return
	}
}

func getExtArgs(_ *cobra.Command, args []string, _ string, ext *ExtCommand) ([]byte, error) {
	var err error
	argsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	// Parse BOF arguments from grumble
	missingRequiredArgs := make([]string, 0)

	// If we have an extension that expects a single string, but more than one has been parsed, combine them
	if len(ext.Arguments) == 1 && strings.Contains(ext.Arguments[0].Type, "string") && len(args) > 0 {
		// The loop below will only read the first element of args because ext.Arguments is 1
		args[0] = strings.Join(args, " ")
	}

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
			data, err := os.ReadFile(word)
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

	return parsedArgs, nil
}

func getBOFArgs(cmd *cobra.Command, args []string, binPath string, ext *ExtCommand) ([]byte, error) {
	var extensionArgs []byte
	binData, err := os.ReadFile(binPath)
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
	parsedArgs, err := getExtArgs(cmd, args, binPath, ext)
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

// CmdExists - checks if a command exists.
func CmdExists(name string, cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

// makeExtensionArgParser builds the valid positional arguments cobra handler for the extension.
func checkExtensionArgs(extCmd *ExtCommand) error {
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
func makeExtensionArgCompleter(extCmd *ExtCommand, _ *cobra.Command, comps *carapace.Carapace) {
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

func makeCommandPlatformFilters(extCmd *ExtCommand) map[string]string {
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
