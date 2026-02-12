package alias

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2019  Bishop Fox
	版权所有 (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	本程序是自由软件：你可以再发布和/或修改它
	it under the terms of the GNU General Public License as published by
	在自由软件基金会发布的 GNU General Public License 条款下，
	the Free Software Foundation, either version 3 of the License, or
	可以使用许可证第 3 版，或
	(at your option) any later version.
	（由你选择）任何更高版本。

	This program is distributed in the hope that it will be useful,
	发布本程序是希望它能发挥作用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但不提供任何担保；甚至不包括
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	对适销性或特定用途适用性的默示担保。请参阅
	GNU General Public License for more details.
	GNU General Public License 以获取更多细节。

	You should have received a copy of the GNU General Public License
	你应当已随本程序收到一份 GNU General Public License 副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	如果没有，请参见 <https://www.gnu.org/licenses/>。
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/client/packages"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/carapace-sh/carapace"
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
	// alias 名称 -> manifest/command。
	loadedAliases = map[string]*loadedAlias{}

	defaultHostProc = map[string]string{
		"windows": windowsDefaultHostProc,
		"linux":   linuxDefaultHostProc,
		"darwin":  macosDefaultHostProc,
	}
)

// Ties the manifest struct to the command struct.
// 将 manifest 结构体与命令结构体绑定。
type loadedAlias struct {
	Manifest *AliasManifest
	Command  *cobra.Command
}

// AliasFile - An OS/Arch specific file.
// AliasFile - OS/Arch 特定文件。
type AliasFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

// AliasArgument - An argument for an alias command.
// AliasArgument - alias 命令的参数。
type AliasArgument struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Desc     string      `json:"desc"`
	Optional bool        `json:"optional"`
	Default  interface{} `json:"default,omitempty"`
	Choices  []string    `json:"choices,omitempty"`
}

// AliasManifest - The manifest for an alias, contains metadata.
// AliasManifest - alias 的 manifest，包含 metadata。
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
	Arguments    []*AliasArgument       `json:"arguments"`
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
// AliasesLoadCmd - 在本地将 alias 加载到 Sliver shell。
func AliasesLoadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	dirPath := args[0]
	// dirPath := ctx.Args.String("dir-path")
	// 从 ctx 读取 dir-path 参数
	alias, err := LoadAlias(dirPath, cmd.Root(), con)
	if err != nil {
		con.PrintErrorf("Failed to load alias: %s\n", err)
	} else {
		con.PrintInfof("%s alias has been loaded\n", alias.Name)
	}
}

// LoadAlias - Load an alias into the Sliver shell from a given directory.
// LoadAlias - 从给定目录将 alias 加载到 Sliver shell。
func LoadAlias(manifestPath string, cmd *cobra.Command, con *console.SliverClient) (*AliasManifest, error) {
	// retrieve alias manifest
	// 获取 alias manifest
	var err error
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		return nil, err
	}

	// parse it
	// 解析它
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	aliasManifest, err := ParseAliasManifest(data)
	if err != nil {
		return nil, err
	}
	aliasManifest.RootPath = filepath.Dir(manifestPath)

	// Build usage string including arguments
	// 构建包含参数的用法字符串
	usage := strings.Builder{}
	usage.WriteString(aliasManifest.CommandName)
	for _, arg := range aliasManifest.Arguments {
		usage.WriteString(" ")
		if arg.Optional {
			usage.WriteString("[")
		}
		usage.WriteString(strings.ToUpper(arg.Name))
		if arg.Optional {
			usage.WriteString("]")
		}
	}

	// Build long help message
	// 构建长帮助信息
	longHelp := strings.Builder{}
	longHelp.WriteString("[[.Bold]]Command:[[.Normal]]")
	longHelp.WriteString(usage.String())
	longHelp.WriteString("\n")
	if len(aliasManifest.Help) > 0 || len(aliasManifest.LongHelp) > 0 {
		longHelp.WriteString("[[.Bold]]About:[[.Normal]]")
		if len(aliasManifest.Help) > 0 {
			longHelp.WriteString(aliasManifest.Help)
			longHelp.WriteString("\n")
		}
		if len(aliasManifest.LongHelp) > 0 {
			longHelp.WriteString(aliasManifest.LongHelp)
			longHelp.WriteString("\n")
		}
	}
	if len(aliasManifest.Arguments) > 0 {
		longHelp.WriteString("[[.Bold]]Arguments:[[.Normal]]")
		for _, arg := range aliasManifest.Arguments {
			longHelp.WriteString("\n\t")
			optStr := ""
			if arg.Optional {
				optStr = "[OPTIONAL]"
			}
			aType := arg.Type
			if aType == "wstring" {
				aType = "string"
			}
			longHelp.WriteString(fmt.Sprintf("%s (%s):\t%s%s", strings.ToUpper(arg.Name), aType, optStr, arg.Desc))
		}
	}
	longHelp.WriteString("\n\n⚠️  If you're having issues passing arguments to the alias please read:\n")
	longHelp.WriteString("https://github.com/BishopFox/sliver/wiki/Aliases-&-Extensions#aliases-command-parsing")

	// for each alias command, add a new app command
	// 为每个 alias 命令添加一个新的 app 命令
	helpMsg := fmt.Sprintf("[%s] %s", aliasManifest.Name, aliasManifest.Help)
	addAliasCmd := &cobra.Command{
		Use:   usage.String(),
		Short: helpMsg,
		Long:  help.FormatHelpTmpl(longHelp.String()),
		Run: func(cmd *cobra.Command, args []string) {
			runAliasCommand(cmd, con, args)
		},
		Args:        cobra.ArbitraryArgs, // 	a.StringList("arguments", "arguments", grumble.Default([]string{}))
		Args:        cobra.ArbitraryArgs, // 	a.StringList(__PH0__, __PH1__, grumble.Default([]字符串{}))
		// Args:        cobra.ArbitraryArgs, // 	a.StringList("arguments", "arguments", grumble.Default([]string{}))
		// 使用任意参数（对应原 grumble 参数列表）
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
		f.StringP("runtime", "r", "v4.0.30319", "Runtime to use for running the assembly")
		f.BoolP("amsi-bypass", "M", false, "Bypass AMSI on Windows")
		f.BoolP("etw-bypass", "E", false, "Bypass ETW on Windows")
		addAliasCmd.Flags().AddFlagSet(f)
	}

	f := pflag.NewFlagSet(aliasManifest.Name, pflag.ContinueOnError)
	f.StringP("process", "p", "", "Path to process to host the shared object")
	f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")
	f.Uint32P("ppid", "P", 0, "parent process ID to use when creating the hosting process (Windows only)")
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	addAliasCmd.Flags().AddFlagSet(f)

	// Setup completions for alias arguments
	// 为 alias 参数设置补全
	comps := carapace.Gen(addAliasCmd)
	makeAliasArgCompleter(aliasManifest, comps)

	cmd.AddCommand(addAliasCmd)

	// Have to use a global map here, as passing the aliasCmd
	// 这里必须使用全局 map，因为传递 aliasCmd
	// either by value or by ref fucks things up
	// 无论按值还是按引用都会把事情搞乱
	loadedAliases[aliasManifest.CommandName] = &loadedAlias{
		Manifest: aliasManifest,
		Command:  addAliasCmd,
	}

	return aliasManifest, nil
}

// ParseAliasManifest - Parse an alias manifest.
// ParseAliasManifest - 解析 alias manifest。
func ParseAliasManifest(data []byte) (*AliasManifest, error) {
	// parse it
	// 解析它
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
	// 从 ctx 读取 arguments 参数列表
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
	// 传递到 Donut 的 payload 特殊情况（.NET assembly 和 sideloaded payload）：
	// The Donut loader has a hard limit of 256 characters for the command line arguments, so
	// Donut loader 对命令行参数有 256 字符的硬限制，因此
	// we're alerting the user that the arguments will be truncated.
	// 我们会提醒用户参数将被截断。
	if len(extArgsStr) > 256 && (aliasManifest.IsAssembly || !aliasManifest.IsReflective) {
		msgStr := ""
		// The --in-process flag only exists for .NET assemblies (aliasManifest.IsAssembly == true).
		// --in-process 参数仅适用于 .NET assembly（aliasManifest.IsAssembly == true）。
		// Groupping the two conditions together could crash the client since ctx.Flags.Type panics
		// 将这两个条件合并可能导致 client 崩溃，因为 ctx.Flags.Type 会 panic
		// if the flag is not registered.
		// 当该参数未注册时。
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
		con.PrintWarnf("%s", msgStr)
		confirm := false
		forms.Confirm("Do you want to continue?", &confirm)
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
		// 参数
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
		// 执行 Assembly
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
		// 参数
		pPid, _ := cmd.Flags().GetUint32("ppid")

		// Spawn DLL
		// 执行 Spawn DLL
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
		// 参数
		pPid, _ := cmd.Flags().GetUint32("ppid")

		// Sideload
		// 执行 Sideload
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
// PrintSpawnDLLOutput - 打印 spawn dll 命令输出。
func PrintSpawnDLLOutput(cmdName string, schema *packages.OutputSchema, spawnDllResp *sliverpb.SpawnDll, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, spawnDllResp.GetResult())
	} else {
		result = spawnDllResp.GetResult()
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	// 将原始结果输出到文件
	if outFilePath != nil {
		outFilePath.WriteString(spawnDllResp.GetResult())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}

// PrintSideloadOutput - Prints the output of a sideload command.
// PrintSideloadOutput - 打印 sideload 命令输出。
func PrintSideloadOutput(cmdName string, schema *packages.OutputSchema, sideloadResp *sliverpb.Sideload, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, sideloadResp.GetResult())
	} else {
		result = sideloadResp.GetResult()
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	// 将原始结果输出到文件
	if outFilePath != nil {
		outFilePath.WriteString(sideloadResp.GetResult())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}

// PrintAssemblyOutput - Prints the output of an execute-assembly command.
// PrintAssemblyOutput - 打印 execute-assembly 命令输出。
func PrintAssemblyOutput(cmdName string, schema *packages.OutputSchema, execAsmResp *sliverpb.ExecuteAssembly, outFilePath *os.File, con *console.SliverClient) {
	var result string

	if schema != nil {
		result = getOutputWithSchema(schema, string(execAsmResp.GetOutput()))
	} else {
		result = string(execAsmResp.GetOutput())
	}
	con.PrintInfof("%s output:\n%s", cmdName, result)

	// Output the raw result to the file
	// 将原始结果输出到文件
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
	// 仅在 OS 相关时为 architecture 添加过滤器。
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

// makeAliasArgCompleter builds the positional and dash arguments completer for the alias.
// makeAliasArgCompleter 为 alias 构建位置参数与 dash 参数补全器。
// It provides completion for:
// 它提供以下补全：
// 1. Positional arguments (before --)
// 1. 位置参数（-- 之前）
// 2. Flag-style arguments after -- (e.g., --target, --port)
// 2. -- 之后的 flag 风格参数（例如 --target、--port）
func makeAliasArgCompleter(alias *AliasManifest, comps *carapace.Carapace) {
	if len(alias.Arguments) == 0 {
		return
	}

	var actions []carapace.Action

	for _, arg := range alias.Arguments {
		var action carapace.Action

		// If choices are defined, use them for completion
		// 如果定义了 choices，则用其进行补全
		if len(arg.Choices) > 0 {
			action = carapace.ActionValues(arg.Choices...).Tag("choices")
		} else {
			// Fall back to type-based completion
			// 否则回退到基于类型的补全
			switch arg.Type {
			case "file":
				action = carapace.ActionFiles().Tag("alias data")
			default:
				action = carapace.ActionValues()
			}
		}

		usage := fmt.Sprintf("(%s) %s", arg.Type, arg.Desc)
		if arg.Optional {
			usage += " (optional)"
		}

		actions = append(actions, action.Usage("%s", usage))
	}

	comps.PositionalCompletion(actions...)

	// Add dash completion for flag-style arguments after --
	// 为 -- 之后的 flag 风格参数添加 dash 补全
	// Pre-build the flag completions at registration time (not in a callback)
	// 在注册阶段预构建 flag 补全（不在回调中）
	flagCompletion := makeAliasFlagNameCompletion(alias)

	// Build value completions for each argument type
	// 为每种参数类型构建取值补全
	valueCompletions := make(map[string]carapace.Action)
	for _, arg := range alias.Arguments {
		// If choices are defined, use them for completion
		// 如果定义了 choices，则用其进行补全
		if len(arg.Choices) > 0 {
			valueCompletions[arg.Name] = carapace.ActionValues(arg.Choices...).Tag("choices")
		} else {
			// Fall back to type-based completion
			// 否则回退到基于类型的补全
			switch arg.Type {
			case "file":
				valueCompletions[arg.Name] = carapace.ActionFiles().Tag("file path")
			case "bool":
				valueCompletions[arg.Name] = carapace.ActionValues("true", "false").Tag("boolean")
			default:
				valueCompletions[arg.Name] = carapace.ActionValues()
			}
		}
	}

	// Use DashAnyCompletion with a smart action that determines context
	// 使用 DashAnyCompletion，通过智能动作判断上下文
	comps.DashAnyCompletion(
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			// If typing a flag (starts with -)
			// 如果正在输入 flag（以 - 开头）
			if strings.HasPrefix(c.Value, "-") {
				return flagCompletion
			}

			// If previous arg was a flag, complete its value
			// 如果前一个参数是 flag，则补全其取值
			if len(c.Args) > 0 {
				lastArg := c.Args[len(c.Args)-1]
				if strings.HasPrefix(lastArg, "-") {
					flagName := strings.TrimLeft(lastArg, "-")
					if action, ok := valueCompletions[flagName]; ok {
						return action
					}
				}
			}

			// Default: show flag names
			// 默认：显示 flag 名称
			return flagCompletion
		}),
	)
}

// makeAliasFlagNameCompletion creates completion for flag names
// makeAliasFlagNameCompletion 为 flag 名称创建补全
func makeAliasFlagNameCompletion(alias *AliasManifest) carapace.Action {
	var results []string

	for _, arg := range alias.Arguments {
		flagName := fmt.Sprintf("--%s", arg.Name)
		desc := arg.Desc
		if arg.Optional {
			desc = fmt.Sprintf("[optional] %s", desc)
		}
		desc = fmt.Sprintf("(%s) %s", arg.Type, desc)
		results = append(results, flagName, desc)
	}

	return carapace.ActionValuesDescribed(results...).Tag("alias arguments")
}
