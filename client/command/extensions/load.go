package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/client/packages"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/carapace-sh/carapace"
	appConsole "github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout = 60

	// ManifestFileName - Extension manifest file name.
	// ManifestFileName - Extension 清单文件 name.
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
	PackageName     string `json:"package_name"`
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
//类型 MultiManifest []*ExtensionManifest

type extensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type extensionArgument struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Desc     string      `json:"desc"`
	Optional bool        `json:"optional"`
	Default  interface{} `json:"default,omitempty"`
	Choices  []string    `json:"choices,omitempty"`
}

func (e *ExtCommand) getFileForTarget(targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range e.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			if e.Manifest.RootPath != "" {
				// Use RootPath for temporarily loaded extensions
				// Use RootPath 用于临时加载的扩展
				filePath = path.Join(e.Manifest.RootPath, extFile.Path)
			} else {
				// Fall back to extensions dir for installed extensions
				// Fall 返回已安装扩展的扩展目录
				filePath = path.Join(assets.GetExtensionsDir(), e.Manifest.Name, extFile.Path)
			}
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

// ExtensionLoadCmd - Temporarily installs an extension from a local directory into the client.
// ExtensionLoadCmd - Temporarily 将本地目录中的扩展安装到 client. 中
// The extension must contain a valid manifest file. If commands from the extension
// The 扩展必须包含来自扩展的有效清单 file. If 命令
// already exist, the user will be prompted to overwrite them.
// 已存在，将提示用户覆盖 them.
func ExtensionLoadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	dirPath := args[0]

	// Add directory check
	// Add 目录检查
	fileInfo, err := os.Stat(dirPath)
	if err != nil || !fileInfo.IsDir() {
		con.PrintErrorf("Path is not a directory: %s\n", dirPath)
		return
	}

	manifest, err := LoadExtensionManifest(filepath.Join(dirPath, ManifestFileName))
	if err != nil {
		return
	}
	// do not add if the command already exists
	// 如果命令已存在，则不要添加
	sliverMenu := con.App.Menu("implant")
	for _, extCmd := range manifest.ExtCommand {
		if CmdExists(extCmd.CommandName, sliverMenu.Command) {
			con.PrintErrorf("%s command already exists\n", extCmd.CommandName)
			confirm := false
			_ = forms.Confirm("Overwrite current command?", &confirm)
			if !confirm {
				return
			}
		}
		ExtensionRegisterCommand(extCmd, cmd.Root(), con)
		con.PrintInfof("Added %s command: %s\n", extCmd.CommandName, extCmd.Help)
	}
}

// LoadExtensionManifest loads and parses an extension manifest file from the given path.
// LoadExtensionManifest 从给定的 path. 加载并解析扩展清单文件
// It registers each command defined in the manifest into the loadedExtensions map
// It 将清单中定义的每个命令注册到 loadedExtensions 映射中
// and registers the complete manifest into loadedManifests. A single manifest may
// 并将完整的清单注册到 loadedManifests. A 单个清单中
// contain multiple extension commands. The manifest's RootPath is set to its containing
// 包含多个扩展 commands. The 清单的 RootPath 设置为其包含的
// directory. Returns the parsed manifest and any errors encountered.
// directory. Returns 已解析的清单和任何错误 encountered.
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
		Name:            old.CommandName, //将旧命令名称视为清单名称，以避免出现奇怪的字符
		Version:         old.Version,
		ExtensionAuthor: old.ExtensionAuthor,
		OriginalAuthor:  old.OriginalAuthor,
		RepoURL:         old.RepoURL,
		RootPath:        old.RootPath,
		//only one command exists in the old manifest, so we can 'confidently' create it here
		//旧清单中只存在一个命令，因此我们可以 __PH0__ 在这里创建它
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
	//Manifest ref 在调用它的解析器中完成

	return ret
}

// parseExtensionManifest - Parse extension manifest from buffer (legacy, only parses one)
// parseExtensionManifest - Parse 缓冲区中的扩展清单（旧版，仅解析一个）
func ParseExtensionManifest(data []byte) (*ExtensionManifest, error) {
	extManifest := &ExtensionManifest{}
	err := json.Unmarshal(data, &extManifest)
	if err != nil || len(extManifest.ExtCommand) == 0 { //extensions must have at least one command to be sensible
	if err != nil || len(extManifest.ExtCommand) == 0 { //扩展必须至少有一个命令才有意义
		//maybe it's an old manifest
		//也许这是一个旧的清单
		if err != nil {
			log.Printf("extension load error: %s", err)
		}
		oldmanifest := &ExtensionManifest_{}
		err := json.Unmarshal(data, &oldmanifest)
		if err != nil {
			//nope, just broken
			//不，只是坏了
			return nil, err
		}
		//yes, ok, lets jigger it to a new manifest
		//是的，好的，让我们将其添加到新的清单中
		extManifest = convertOldManifest(oldmanifest)
	}
	//pass ref to manifest to each command and initialize output schema if applicable
	//将 ref 传递给每个命令的清单并初始化输出模式（如果适用）
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

// ExtensionRegisterCommand adds an extension command to the cobra command system.
// ExtensionRegisterCommand 在 cobra 命令 system. 的基础上添加了一个扩展命令
// It validates the extension's arguments, updates the loadedExtensions map, and
// It 验证扩展的参数，更新 loadedExtensions 映射，并且
// creates a cobra.Command with proper usage text, help documentation, and argument
// 使用正确的用法文本、帮助文档和参数创建 cobra.Command
// handling. The command is added as a subcommand to the provided parent cobra.Command.
// handling. The 命令作为子命令添加到提供的父命令 cobra.Command.
// Arguments are displayed in the help text as uppercase, with optional args in square
// Arguments 在帮助文本中显示为大写，可选参数为方形
// brackets. The help text includes sections for command usage, description, and detailed
// brackets. The 帮助文本包括命令用法、描述和详细信息部分
// argument specifications.
// 参数 specifications.
func ExtensionRegisterCommand(extCmd *ExtCommand, cmd *cobra.Command, con *console.SliverClient) {
	if errInvalidArgs := checkExtensionArgs(extCmd); errInvalidArgs != nil {
		con.PrintErrorf("%s", errInvalidArgs.Error())
		return
	}

	loadedExtensions[extCmd.CommandName] = extCmd

	usage := strings.Builder{}
	usage.WriteString(extCmd.CommandName)
	//build usage including args
	//构建用法，包括参数
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
	//预先考虑帮助值，因为否则 I 看不到它应该显示在哪里
	//build the command ref
	//构建命令参考
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
	//如果指定的参数超过 0 个，请在长帮助文本的底部描述每个参数（以防清单不包含它）
	for _, arg := range extCmd.Arguments {
		longHelp.WriteString("\n\t")
		optStr := ""
		if arg.Optional {
			optStr = "[OPTIONAL]"
		}
		aType := arg.Type
		if aType == "wstring" {
			aType = "string" //avoid confusion, as this is mostly for telling operator what to shove into the args
			aType = "string" //避免混淆，因为这主要是为了告诉 operator 将什么推入参数中
		}
		//idk how to make this look nice, tabs don't work especially good - maybe should use the table stuff other things do? Pls help.
		//我不知道如何使它看起来不错，选项卡工作得不是特别好 - 也许应该使用其他东西做的表格东西？ Pls help.
		longHelp.WriteString(fmt.Sprintf("%s (%s):\t%s%s", strings.ToUpper(arg.Name), aType, optStr, arg.Desc))
	}

	// Command
	extensionCmd := &cobra.Command{
		Use:   usage.String(),
		Short: extCmd.Help,
		Long:  help.FormatHelpTmpl(longHelp.String()),
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
	// Try 在加载的扩展中查找扩展
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
	//extensionList 包含所有 *implant* 加载的 extensions. Is 一个 sha256 相关 file. 的总和

	//get the file hash to compare against implant extensions later
	//稍后获取文件哈希以与 implant 扩展名进行比较
	toberunfilepath, err := ext.getFileForTarget(goos, goarch)
	if err != nil {
		return err
	}
	//we need to check the dependent if it's a bof
	//我们需要检查依赖项是否是 bof
	if ext.DependsOn != "" {
		//verify extension is in loaded list
		//验证扩展名是否在加载列表中
		if extension, found := loadedExtensions[ext.DependsOn]; found {
			toberunfilepath, err = extension.getFileForTarget(goos, goarch)
			if err != nil {
				return err
			}
		} else {
			// handle error
			// 处理错误
			return fmt.Errorf("attempted to load non-existing extension: %s", ext.DependsOn)
		}
	}

	//todo, maybe cache these values somewhere
	//todo，也许将这些值缓存在某个地方
	toberunfiledata, err := os.ReadFile(toberunfilepath)
	if err != nil {
		return err
	}
	if len(toberunfiledata) == 0 {
		//read an empty file, bail out
		//读取一个空文件，退出
		return errors.New("read empty extension file content")
	}
	bd := sha256.Sum256(toberunfiledata)
	toberunhash := hex.EncodeToString(bd[:])

	for _, extName := range extensionList {
		//check if extension we are trying to run (ext.CommandName or ext.DependsOn, for bofs) is already loaded
		//检查我们尝试运行的扩展（ext.CommandName 或 ext.DependsOn，对于 bofs）是否已加载
		if extName == toberunhash {
			//exists on the other side, exit early
			//另一边有，早点退出
			return nil
		}
	}
	// Extension not found, let's load it
	// Extension 未找到，让我们加载它
	// BOFs are not loaded by the DLL loader, but we need to load the loader (more load more good)
	// BOFs 不是由 DLL 加载器加载的，但是我们需要加载该加载器（越加载越好）
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
	//将扩展名称设置为数据的哈希值以避免加载多个实例
	bd := sha256.Sum256(binData)
	name := hex.EncodeToString(bd[:])
	sess, beac := con.ActiveTarget.GetInteractive()
	ctrl := make(chan bool)
	//first time run of an extension will require some waiting depending on the size
	//第一次运行扩展将需要一些等待，具体取决于大小
	if sess != nil {
		msg := fmt.Sprintf("Sending %s to implant ...", ext.CommandName)
		con.SpinUntil(msg, ctrl)
	}
	//don't block if we are in beacon mode
	//如果我们处于 beacon 模式，则不要阻止
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
	//session 模式（希望如此）
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
	// BOFs (Beacon Object Files) 是一种特定类型的扩展
	// that require another extension (a COFF loader) to be present.
	// 需要另一个扩展（COFF 加载程序）为 present.
	// BOFs also have strongly typed arguments that need to be parsed in the proper way.
	// BOFs 还具有强类型参数，需要在正确的 way. 中进行解析
	// This block will pack both the BOF data and its arguments into a single buffer that
	// This 块会将 BOF 数据及其参数打包到一个缓冲区中
	// the loader will extract and load.
	// 加载程序将提取并 load.
	if isBOF {
		// Beacon Object File -- requires a COFF loader
		// Beacon Object File -- 需要 COFF 加载程序
		extensionArgs, err = getBOFArgs(cmd, args, binPath, ext)
		if err != nil {
			con.PrintErrorf("BOF args error: %s\n", err)
			return
		}
		extName = ext.DependsOn
		entryPoint = loadedExtensions[extName].Entrypoint // should exist at this point
		entryPoint = loadedExtensions[extName].Entrypoint // 此时应该存在
	} else {
		// Regular DLL - Just join the arguments with spaces
		// Regular DLL - Just 用空格连接参数
		if len(args) > 0 {
			extensionArgs = []byte(strings.Join(args, " "))
		} else {
			extensionArgs = []byte{}
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
		//如果我们使用 bof，我们实际上是在调用 coffloader 扩展 - 因此从 dep ref 获取文件并使用该 shasum

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
			// 处理错误
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
// PrintExtOutput - Print 外部执行 output.
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
			// Get 输出模式
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
		con.PrintErrorf("%s", callExtension.Response.Err)
		return
	}
}

func getBOFArgs(cmd *cobra.Command, args []string, binPath string, ext *ExtCommand) ([]byte, error) {
	var extensionArgs []byte
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return nil, err
	}

	// Now build the extension's argument buffer
	// Now 构建扩展的参数缓冲区
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
	parsedArgs, err := ParseFlagArgumentsToBuffer(cmd, args, binPath, ext)
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
// CmdExists - 检查命令是否 exists.
func CmdExists(name string, cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

// makeExtensionArgParser builds the valid positional arguments cobra handler for the extension.
// makeExtensionArgParser 为 extension. 构建有效的位置参数 cobra 处理程序
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

// makeExtensionArgCompleter builds the positional and dash arguments completer for the extension.
// makeExtensionArgCompleter 为 extension. 构建位置和破折号参数完成器
// It provides completion for:
// It 提供完成：
// 1. Positional arguments (before --)
// 1. Positional 参数（在 -- 之前）
// 2. Flag-style arguments after -- (e.g., --process, --shellcode)
// 2. -- (e.g., __PH1__, __PH2__) 之后的 Flag__PH0__ 参数
func makeExtensionArgCompleter(extCmd *ExtCommand, _ *cobra.Command, comps *carapace.Carapace) {
	var actions []carapace.Action

	for _, arg := range extCmd.Arguments {
		var action carapace.Action

		// If choices are defined, use them for completion
		// If 选择已定义，使用它们来完成
		if len(arg.Choices) > 0 {
			action = carapace.ActionValues(arg.Choices...).Tag("choices")
		} else {
			// Fall back to type-based completion
			// Fall 返回 type__PH0__ 完成
			switch arg.Type {
			case "file":
				action = carapace.ActionFiles().Tag("extension data")
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
	// -- 之后的 Add 破折号完成 flag__PH0__ 参数
	// The extension arguments are parsed as flags (--name value) by argparser.go
	// The 扩展参数被 argparser.go 解析为标志（__PH0__ 值）
	if len(extCmd.Arguments) > 0 {
		// Pre-build the flag completions at registration time (not in a callback)
		// Pre__PH0__ 注册时完成的标志（不在 callback 中）
		// This ensures the values are captured correctly
		// This 确保正确捕获值
		flagCompletion := makeExtensionFlagNameCompletion(extCmd)

		// Build value completions for each argument type
		// 每个参数类型的 Build 值完成
		valueCompletions := make(map[string]carapace.Action)
		for _, arg := range extCmd.Arguments {
			// If choices are defined, use them for completion
			// If 选择已定义，使用它们来完成
			if len(arg.Choices) > 0 {
				valueCompletions[arg.Name] = carapace.ActionValues(arg.Choices...).Tag("choices")
			} else {
				// Fall back to type-based completion
				// Fall 返回 type__PH0__ 完成
				switch arg.Type {
				case "file":
					valueCompletions[arg.Name] = carapace.ActionFiles().Tag("file path")
				default:
					valueCompletions[arg.Name] = carapace.ActionValues()
				}
			}
		}

		// Use DashAnyCompletion with a smart action that determines context
		// Use DashAnyCompletion 通过智能操作确定上下文
		comps.DashAnyCompletion(
			carapace.ActionCallback(func(c carapace.Context) carapace.Action {
				// If typing a flag (starts with -)
				// If 输入一个标志（以 - 开头）
				if strings.HasPrefix(c.Value, "-") {
					return flagCompletion
				}

				// If previous arg was a flag, complete its value
				// If 之前的 arg 是一个标志，完成它的值
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
				// Default: 显示标志名称
				return flagCompletion
			}),
		)
	}
}

// makeExtensionFlagNameCompletion creates completion for flag names
// makeExtensionFlagNameCompletion 创建标志名称的补全
func makeExtensionFlagNameCompletion(extCmd *ExtCommand) carapace.Action {
	var results []string

	for _, arg := range extCmd.Arguments {
		flagName := fmt.Sprintf("--%s", arg.Name)
		desc := arg.Desc
		if arg.Optional {
			desc = fmt.Sprintf("[optional] %s", desc)
		}
		desc = fmt.Sprintf("(%s) %s", arg.Type, desc)
		results = append(results, flagName, desc)
	}

	return carapace.ActionValuesDescribed(results...).Tag("extension arguments")
}

func makeCommandPlatformFilters(extCmd *ExtCommand) map[string]string {
	filtersOS := make(map[string]bool)
	filtersArch := make(map[string]bool)

	var all []string

	// Only add filters for architectures when there OS matters.
	// Only 在存在 OS matters. 时为架构添加过滤器
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
