package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

const (
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

func load(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "Please provide an extension path\n")
		return
	}

	// retrieve extension manifest
	manifestPath := fmt.Sprintf("%s/%s", ctx.Args[0], "manifest.json")
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		fmt.Printf(Warn+"%v", err)
	}
	// parse it
	ext := &extension{}
	err = json.Unmarshal(jsonBytes, ext)
	if err != nil {
		fmt.Printf(Warn+"error loading extension: %v", err)
		return
	}
	ext.Path = ctx.Args[0]
	// for each extension command, add a new app command
	for _, extCmd := range ext.Commands {
		// do not add if the command already exists
		if cmdExists(extCmd.Name, ctx.App) {
			fmt.Printf(Warn+"%s command already exists\n", extCmd.Name)
			return
		}
		fmt.Printf(Info+"Adding %s command: %s\n", extCmd.Name, extCmd.Help)

		// Have to use a global map here, as passing the extCmd
		// either by value or by ref fucks things up
		commandMap[extCmd.Name] = *ext
		helpMsg := fmt.Sprintf("[%s] %s", ext.Name, extCmd.Help)
		ctx.App.AddCommand(&grumble.Command{
			Name:      extCmd.Name,
			Help:      helpMsg,
			LongHelp:  help.FormatHelpTmpl(extCmd.LongHelp),
			AllowArgs: extCmd.AllowArgs,
			Run: func(extCtx *grumble.Context) error {
				fmt.Println()
				runExtensionCommand(extCtx, rpc)
				fmt.Println()
				return nil
			},
			Flags: func(f *grumble.Flags) {
				f.String("p", "process", "", "Path to process to host the shared object")
				f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			},
			HelpGroup: consts.ExtensionHelpGroup,
		})
	}
	fmt.Printf(Info+"%s extension has been loaded\n", ext.Name)
}

func runExtensionCommand(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	ext, ok := commandMap[ctx.Command.Name]
	if !ok {
		fmt.Printf(Warn+"No extension command found for `%s` command\n", ctx.Command.Name)
		return
	}

	binPath, err := ext.getFileForTarget(ctx.Command.Name, session.GetOS(), session.GetArch())
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}

	c, err := ext.getCommandFromName(ctx.Command.Name)
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}

	var args string
	if len(c.DefaultArgs) != 0 {
		args = c.DefaultArgs
	}
	if ctx.Command.AllowArgs {
		if len(ctx.Args) > 0 {
			args = strings.Join(ctx.Args[0:], " ")
		}
	}

	entryPoint := c.Entrypoint
	processName := ctx.Flags.String("process")
	if processName == "" {
		processName, err = c.getDefaultProcess(session.GetOS())
		if err != nil {
			fmt.Printf(Warn+"Error: %v\n", err)
			return
		}
	}

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}
	if c.IsAssembly {
		executeAssemblyResp, err := rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
			Request:    ActiveSession.Request(ctx),
			AmsiBypass: true,
			Arguments:  args,
			Assembly:   binData,
			Process:    processName,
		})

		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}
		fmt.Printf(Info+"Output:\n%s", executeAssemblyResp.GetOutput())
	} else if c.IsReflective {
		offset, err := getExportOffset(binPath, c.Entrypoint)
		if err != nil {
			fmt.Printf(Warn+"Error: %v\n", err)
			return
		}
		spawnDllResp, err := rpc.SpawnDll(context.Background(), &sliverpb.SpawnDllReq{
			Request:  ActiveSession.Request(ctx),
			Args:     strings.Trim(args, " "),
			Data:     binData,
			ProcName: processName,
			Offset:   offset,
		})

		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}

		fmt.Printf(Info+"Output:\n%s", spawnDllResp.GetResult())
	} else {
		sideloadResp, err := rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
			Request:    ActiveSession.Request(ctx),
			Args:       args,
			Data:       binData,
			EntryPoint: entryPoint,
			ProcName:   processName,
		})

		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}

		fmt.Printf(Info+"Output:\n%s", sideloadResp.GetResult())
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
