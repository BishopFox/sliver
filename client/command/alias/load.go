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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/packages"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	app "github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout = 60

	ManifestFileName = "alias.json"

	windowsDefaultHostProc = `c:\windows\system32\notepad.exe`
	linuxDefaultHostProc   = "/bin/bash"
	macosDefaultHostProc   = "/Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment"
)

var (
	// alias name -> manifest/command.
	loadedAliases = map[string]*loadedAlias{}

	defaultHostProc = map[string]string{
		"windows": windowsDefaultHostProc,
		"linux":   linuxDefaultHostProc,
		"darwin":  macosDefaultHostProc,
	}
)

// Ties the manifest struct to the command struct.
type loadedAlias struct {
	Manifest *AliasManifest
	Command  *cobra.Command
}

// AliasFile - An OS/Arch specific file.
type AliasFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

// AliasManifest - The manifest for an alias, contains metadata.
type AliasManifest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	CommandName    string `json:"command_name"`
	OriginalAuthor string `json:"original_author"`
	RepoURL        string `json:"repo_url"`
	Help           string `json:"help"`
	LongHelp       string `json:"long_help"`

	Entrypoint   string                 `json:"entrypoint"`
	AllowArgs    bool                   `json:"allow_args"`
	DefaultArgs  string                 `json:"default_args"`
	Files        []*AliasFile           `json:"files"`
	IsReflective bool                   `json:"is_reflective"`
	IsAssembly   bool                   `json:"is_assembly"`
	Schema       *packages.OutputSchema `json:"schema"`

	RootPath   string `json:"-"`
	ArmoryName string `json:"-"`
	ArmoryPK   string `json:"-"`
}

func (ec *AliasManifest) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

func (a *AliasManifest) getFileForTarget(_ string, targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range a.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = path.Join(assets.GetAliasesDir(), a.CommandName, extFile.Path)
			break
		}
	}
	if filePath == "" {
		err := fmt.Errorf("no alias file found for %s/%s", targetOS, targetArch)
		return "", err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = fmt.Errorf("alias file not found: %s", filePath)
		return "", err
	}
	return filePath, nil
}

// AliasesLoadCmd - Locally load a alias into the Sliver shell.
func AliasesLoadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	dirPath := args[0]
	// dirPath := ctx.Args.String("dir-path")
	alias, err := LoadAlias(dirPath, cmd.Root(), con)
	if err != nil {
		con.PrintErrorf("Failed to load alias: %s\n", err)
	} else {
		con.PrintInfof("%s alias has been loaded\n", alias.Name)
	}
}

// LoadAlias - Load an alias into the Sliver shell from a given directory.
func LoadAlias(manifestPath string, cmd *cobra.Command, con *console.SliverClient) (*AliasManifest, error) {
	// retrieve alias manifest
	var err error
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		return nil, err
	}

	// parse it
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	aliasManifest, err := ParseAliasManifest(data)
	if err != nil {
		return nil, err
	}
	aliasManifest.RootPath = filepath.Dir(manifestPath)

	// for each alias command, add a new app command
	helpMsg := fmt.Sprintf("[%s] %s", aliasManifest.Name, aliasManifest.Help)
	longHelpMsg := help.FormatHelpTmpl(aliasManifest.LongHelp)
	longHelpMsg += "\n\n⚠️  If you're having issues passing arguments to the alias please read:\n"
	longHelpMsg += "https://github.com/BishopFox/sliver/wiki/Aliases-&-Extensions#aliases-command-parsing"
	addAliasCmd := &cobra.Command{
		Use:   aliasManifest.CommandName,
		Short: helpMsg,
		Long:  longHelpMsg,
		Run: func(cmd *cobra.Command, args []string) {
			runAliasCommand(cmd, con, args)
		},
		Args:        cobra.ArbitraryArgs, // 	a.StringList("arguments", "arguments", grumble.Default([]string{}))
		GroupID:     consts.AliasHelpGroup,
		Annotations: makeAliasPlatformFilters(aliasManifest),
	}

	if aliasManifest.IsAssembly {
		f := pflag.NewFlagSet("assembly", pflag.ContinueOnError)
		f.StringP("method", "m", "", "Optional method (a method is required for a .NET DLL)")
		f.StringP("class", "c", "", "Optional class name (required for .NET DLL)")
		f.StringP("app-domain", "d", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
		f.StringP("arch", "a", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
		f.BoolP("in-process", "i", false, "Run in the current sliver process")
		f.StringP("runtime", "r", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
		f.BoolP("amsi-bypass", "M", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
		f.BoolP("etw-bypass", "E", false, "Bypass ETW on Windows (only supported when used with --in-process)")
		addAliasCmd.Flags().AddFlagSet(f)
	}

	f := pflag.NewFlagSet(aliasManifest.Name, pflag.ContinueOnError)
	f.StringP("process", "p", "", "Path to process to host the shared object")
	f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")
	f.Uint32P("ppid", "P", 0, "parent process ID to use when creating the hosting process (Windows only)")
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	addAliasCmd.Flags().AddFlagSet(f)

	cmd.AddCommand(addAliasCmd)

	// Have to use a global map here, as passing the aliasCmd
	// either by value or by ref fucks things up
	loadedAliases[aliasManifest.CommandName] = &loadedAlias{
		Manifest: aliasManifest,
		Command:  addAliasCmd,
	}

	return aliasManifest, nil
}

// ParseAliasManifest - Parse an alias manifest.
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

	if alias.Schema != nil {
		if !packages.IsValidSchemaType(alias.Schema.Name) {
			return nil, fmt.Errorf("%s is not a valid schema type", alias.Schema.Name)
		}
		alias.Schema.IngestColumns()
	}

	return alias, nil
}

func runAliasCommand(cmd *cobra.Command, con *console.SliverClient, args []string) {
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

	loadedAlias, ok := loadedAliases[cmd.Name()]
	if !ok {
		con.PrintErrorf("No alias found for `%s` command\n", cmd.Name())
		return
	}
	aliasManifest := loadedAlias.Manifest
	binPath, err := aliasManifest.getFileForTarget(cmd.Name(), goos, goarch)
	if err != nil {
		con.PrintErrorf("Fail to find alias file: %s\n", err)
		return
	}
	// args := ctx.Args.StringList("arguments")
	var extArgsStr string
	if len(aliasManifest.DefaultArgs) != 0 && len(args) == 0 {
		extArgsStr = aliasManifest.DefaultArgs
	} else {
		extArgsStr = strings.Join(args, " ")
	}

	extArgsStr = strings.TrimSpace(extArgsStr)
	entryPoint := aliasManifest.Entrypoint
	processArgsStr, _ := cmd.Flags().GetString("process-arguments")
	// Special case for payloads with pass to Donut (.NET assemblies and sideloaded payloads):
	// The Donut loader has a hard limit of 256 characters for the command line arguments, so
	// we're alerting the user that the arguments will be truncated.
	if len(extArgsStr) > 256 && (aliasManifest.IsAssembly || !aliasManifest.IsReflective) {
		msgStr := ""
		// The --in-process flag only exists for .NET assemblies (aliasManifest.IsAssembly == true).
		// Groupping the two conditions together could crash the client since ctx.Flags.Type panics
		// if the flag is not registered.
		if aliasManifest.IsAssembly {
			inProcess, _ := cmd.Flags().GetBool("in-process")
			runtime, _ := cmd.Flags().GetString("runtime")
			amsiBypass, _ := cmd.Flags().GetBool("amsi-bypass")
			etwBypass, _ := cmd.Flags().GetBool("etw-bypass")
			if !inProcess {
				msgStr = " Arguments are limited to 256 characters when using the default fork/exec model for .NET assemblies.\nConsider using the --in-process flag to execute .NET assemblies in-process and work around this limitation.\n"
			}
			if !inProcess && (runtime != "" || etwBypass || amsiBypass) {
				con.PrintErrorf("The --runtime, --etw-bypass, and --amsi-bypass flags can only be used with the --in-process flag\n")
				return
			}
		} else if !aliasManifest.IsReflective {
			msgStr = " Arguments are limited to 256 characters when using the default fork/exec model for non-reflective PE payloads.\n"
		}
		con.PrintWarnf(msgStr)
		confirm := false
		prompt := &survey.Confirm{Message: "Do you want to continue?"}
		survey.AskOne(prompt, &confirm, nil)
		if !confirm {
			return
		}
	}
	processArgs := strings.Split(processArgsStr, " ")
	processName, _ := cmd.Flags().GetString("process")
	if processName == "" {
		processName, err = aliasManifest.getDefaultProcess(goos)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	isDLL := false
	if strings.ToLower(filepath.Ext(binPath)) == ".dll" {
		isDLL = true
	}
	binData, err := os.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	var outFilePath *os.File
	if save, _ := cmd.Flags().GetBool("save"); save {
		outFile := filepath.Base(fmt.Sprintf("%s_%s*.log", filepath.Base(cmd.Name()), filepath.Base(session.GetHostname())))
		outFilePath, err = os.CreateTemp("", outFile)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	if aliasManifest.IsAssembly {

		// Flags
		arch, _ := cmd.Flags().GetString("arch")
		method, _ := cmd.Flags().GetString("method")
		className, _ := cmd.Flags().GetString("class")
		appDomain, _ := cmd.Flags().GetString("app-domain")
		pPid, _ := cmd.Flags().GetUint32("ppid")
		inProcess, _ := cmd.Flags().GetBool("in-process")
		runtime, _ := cmd.Flags().GetString("runtime")
		amsiBypass, _ := cmd.Flags().GetBool("amsi-bypass")
		etwBypass, _ := cmd.Flags().GetBool("etw-bypass")

		// Execute Assembly
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", cmd.Name(), extArgsStr)
		con.SpinUntil(msg, ctrl)
		executeAssemblyResp, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
			Request:     con.ActiveTarget.Request(cmd),
			IsDLL:       isDLL,
			Process:     processName,
			Arguments:   args,
			Assembly:    binData,
			Arch:        arch,
			Method:      method,
			ClassName:   className,
			AppDomain:   appDomain,
			ProcessArgs: processArgs,
			PPid:        pPid,
			InProcess:   inProcess,
			Runtime:     runtime,
			AmsiBypass:  amsiBypass,
			EtwBypass:   etwBypass,
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		if executeAssemblyResp.Response != nil && executeAssemblyResp.Response.Async {
			con.AddBeaconCallback(executeAssemblyResp.Response.TaskID, func(task *clientpb.BeaconTask) {
				err = proto.Unmarshal(task.Response, executeAssemblyResp)
				if err != nil {
					con.PrintErrorf("Failed to decode call ext response %s\n", err)
					return
				}
				PrintAssemblyOutput(cmd.Name(), aliasManifest.Schema, executeAssemblyResp, outFilePath, con)
			})
			con.PrintAsyncResponse(executeAssemblyResp.Response)
		} else {
			PrintAssemblyOutput(cmd.Name(), aliasManifest.Schema, executeAssemblyResp, outFilePath, con)
		}

	} else if aliasManifest.IsReflective {
		// Flags
		pPid, _ := cmd.Flags().GetUint32("ppid")

		// Spawn DLL
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", cmd.Name(), extArgsStr)
		con.SpinUntil(msg, ctrl)
		spawnDllResp, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
			Request:     con.ActiveTarget.Request(cmd),
			Args:        args,
			Data:        binData,
			ProcessName: processName,
			EntryPoint:  aliasManifest.Entrypoint,
			Kill:        true,
			ProcessArgs: processArgs,
			PPid:        pPid,
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		if spawnDllResp.Response != nil && spawnDllResp.Response.Async {
			con.AddBeaconCallback(spawnDllResp.Response.TaskID, func(task *clientpb.BeaconTask) {
				err = proto.Unmarshal(task.Response, spawnDllResp)
				if err != nil {
					con.PrintErrorf("Failed to decode call ext response %s\n", err)
					return
				}
				PrintSpawnDLLOutput(cmd.Name(), aliasManifest.Schema, spawnDllResp, outFilePath, con)
			})
			con.PrintAsyncResponse(spawnDllResp.Response)
		} else {
			PrintSpawnDLLOutput(cmd.Name(), aliasManifest.Schema, spawnDllResp, outFilePath, con)
		}

	} else {
		// Flags
		pPid, _ := cmd.Flags().GetUint32("ppid")

		// Sideload
		ctrl := make(chan bool)
		msg := fmt.Sprintf("Executing %s %s ...", cmd.Name(), extArgsStr)
		con.SpinUntil(msg, ctrl)
		sideloadResp, err := con.Rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
			Request:     con.ActiveTarget.Request(cmd),
			Args:        args,
			Data:        binData,
			EntryPoint:  entryPoint,
			ProcessName: processName,
			Kill:        true,
			IsDLL:       isDLL,
			ProcessArgs: processArgs,
			PPid:        pPid,
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		if sideloadResp.Response != nil && sideloadResp.Response.Async {
			con.AddBeaconCallback(sideloadResp.Response.TaskID, func(task *clientpb.BeaconTask) {
				err = proto.Unmarshal(task.Response, sideloadResp)
				if err != nil {
					con.PrintErrorf("Failed to decode call ext response %s\n", err)
					return
				}
				PrintSideloadOutput(cmd.Name(), aliasManifest.Schema, sideloadResp, outFilePath, con)
			})
			con.PrintAsyncResponse(sideloadResp.Response)
		} else {
			PrintSideloadOutput(cmd.Name(), aliasManifest.Schema, sideloadResp, outFilePath, con)
		}
	}
}

func getOutputWithSchema(schema *packages.OutputSchema, result string) string {
	if schema == nil {
		return result
	}

	outputSchema := packages.GetNewPackageOutput(schema.Name)
	if outputSchema == nil {
		return result
	}

	err := outputSchema.IngestData([]byte(result), schema.Columns(), schema.GroupBy)
	if err != nil {
		return result
	}
	return outputSchema.CreateTable()
}

// PrintSpawnDLLOutput - Prints the output of a spawn dll command.
func PrintSpawnDLLOutput(cmdName string, schema *packages.OutputSchema, spawnDllResp *sliverpb.SpawnDll, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, spawnDllResp.GetResult())
	} else {
		result = spawnDllResp.GetResult()
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	if outFilePath != nil {
		outFilePath.WriteString(spawnDllResp.GetResult())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}

// PrintSideloadOutput - Prints the output of a sideload command.
func PrintSideloadOutput(cmdName string, schema *packages.OutputSchema, sideloadResp *sliverpb.Sideload, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, sideloadResp.GetResult())
	} else {
		result = sideloadResp.GetResult()
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	if outFilePath != nil {
		outFilePath.WriteString(sideloadResp.GetResult())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}

// PrintAssemblyOutput - Prints the output of an execute-assembly command.
func PrintAssemblyOutput(cmdName string, schema *packages.OutputSchema, execAsmResp *sliverpb.ExecuteAssembly, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, string(execAsmResp.GetOutput()))
	} else {
		result = string(execAsmResp.GetOutput())
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	if outFilePath != nil {
		outFilePath.Write(execAsmResp.GetOutput())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}

func makeAliasPlatformFilters(alias *AliasManifest) map[string]string {
	filtersOS := make(map[string]bool)
	filtersArch := make(map[string]bool)

	var all []string

	// Only add filters for architectures when there OS matters.
	for _, file := range alias.Files {
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
		app.CommandFilterKey: strings.Join(all, ","),
	}
}
