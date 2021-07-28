package extensions

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/unicode"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

const (
	defaultTimeout = 60
)

var commandMap map[string]extensionCommand

type extensionCommand struct {
	Name       string              `json:"name"`
	Help       string              `json:"help"`
	Files      []extensionFiles    `json:"extFiles"`
	Arguments  []extensionArgument `json:"arguments"`
	Entrypoint string              `json:"entrypoint"`
	DependsOn  string              `json:"dependsOn"`
	Init       string              `json:"init"`
	Path       string
}
type binFiles struct {
	Ext64Path string `json:"x64"`
	Ext32Path string `json:"x86"`
}

type extensionFiles struct {
	OS    string   `json:"os"`
	Files binFiles `json:"files"`
}

type extensionArgument struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"desc"`
	Optional bool   `json:"optional"`
}

func (e *extensionCommand) getFileForTarget(cmdName string, targetOS string, targetArch string) (filePath string, err error) {
	for _, ef := range e.Files {
		if targetOS == ef.OS {
			switch targetArch {
			case "x86":
				filePath = fmt.Sprintf("%s/%s/%s/%s", e.Path, targetOS, targetArch, ef.Files.Ext32Path)
			case "x64":
				filePath = fmt.Sprintf("%s/%s/%s/%s", e.Path, targetOS, targetArch, ef.Files.Ext64Path)
			default:
				filePath = fmt.Sprintf("%s/%s/%s/%s", e.Path, targetOS, targetArch, ef.Files.Ext64Path)
			}
		}
	}
	if filePath == "" {
		err = fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
	}
	return
}

func LoadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	dirPath := ctx.Args.String("dir-path")
	extCmds, err := ParseExtensions(dirPath)
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}

	for _, extCmd := range extCmds {
		// do not add if the command already exists
		if cmdExists(extCmd.Name, con.App) {
			con.PrintErrorf("%s command already exists\n", extCmd.Name)
			continue
		}
		RegisterExtensionCommand(extCmd, con)
		con.PrintInfof("Added %s command: %s\n", extCmd.Name, extCmd.Help)
	}
}

func ParseExtensions(extPath string) ([]*extensionCommand, error) {
	manifestPath := fmt.Sprintf("%s/%s", extPath, "manifest.json")
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	extensionsCmds := make([]*extensionCommand, 0)
	err = json.Unmarshal(jsonBytes, &extensionsCmds)
	if err != nil {
		return nil, err
	}
	for _, extCmd := range extensionsCmds {
		extCmd.Path = extPath
		commandMap[extCmd.Name] = *extCmd
	}
	return extensionsCmds, nil
}

func RegisterExtensionCommand(extCmd *extensionCommand, con *console.SliverConsoleClient) {
	commandMap[extCmd.Name] = *extCmd
	helpMsg := extCmd.Help
	extensionCmd := &grumble.Command{
		Name: extCmd.Name,
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

func ListCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: con.ActiveSession.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("Error: %s", err)
		return
	}

	if extList.Response != nil && extList.Response.Err != "" {
		con.PrintErrorf("Error: %s", extList.Response.Err)
		return
	}
	if len(extList.Names) > 0 {
		con.PrintInfof("Loaded extensions:\n")
		for _, ext := range extList.Names {
			con.Printf("- %s\n", ext)
		}
	}
}

func loadExtension(ctx *grumble.Context, session *clientpb.Session, con *console.SliverConsoleClient, ext *extensionCommand) error {
	var extensionList []string
	binPath, err := ext.getFileForTarget(ctx.Command.Name, session.OS, session.Arch)
	if err != nil {
		return err
	}
	// Try to find the extension in the loaded extensions
	if len(session.Extensions) == 0 {
		extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
			Request: con.ActiveSession.Request(ctx),
		})
		if err != nil {
			con.PrintErrorf("Error: %s\n", err.Error())
			return err
		}
		if extList.Response != nil && extList.Response.Err != "" {
			return errors.New(extList.Response.Err)
		}
		extensionList = extList.Names
		if filepath.Ext(binPath) != ".o" {
			// Don't update session for BOFs, we don't cache them
			// on the implant side (yet)
			// update the Session info
			_, err = con.Rpc.UpdateSession(context.Background(), &clientpb.UpdateSession{
				Extensions: extensionList,
				SessionID:  session.ID,
			})
			if err != nil {
				con.PrintErrorf("Error: %s\n", err.Error())
				return err
			}
		}
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
	return nil
}

func registerExtension(con *console.SliverConsoleClient, ext *extensionCommand, binData []byte, session *clientpb.Session, ctx *grumble.Context) error {
	registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    ext.Name,
		Data:    binData,
		OS:      session.OS,
		Init:    ext.Init,
		Request: con.ActiveSession.Request(ctx),
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
	depExt, f := commandMap[depName]
	if f {
		depBinPath, err := depExt.getFileForTarget(depExt.Name, session.OS, session.Arch)
		if err != nil {
			return err
		}
		depBinData, err := ioutil.ReadFile(depBinPath)
		if err != nil {
			return err
		}
		return registerExtension(con, &depExt, depBinData, session, ctx)
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
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	ext, ok := commandMap[ctx.Command.Name]
	if !ok {
		con.PrintErrorf("No extension command found for `%s` command\n", ctx.Command.Name)
		return
	}

	if err = loadExtension(ctx, session, con, &ext); err != nil {
		con.PrintErrorf("Error: %s\n", err)
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
	// than require another extension (a COFF loader) to be present.
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
		entryPoint = commandMap[extName].Entrypoint // should exist at this point
	} else {
		// Regular DLL
		extArgs := strings.Join(ctx.Args.StringList("arguments"), " ")
		extensionArgs = []byte(extArgs)
		extName = ext.Name
		entryPoint = ext.Entrypoint
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Executing %s ...", ctx.Command.Name)
	con.SpinUntil(msg, ctrl)
	callExtension, err = con.Rpc.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:    extName,
		Export:  entryPoint,
		Args:    extensionArgs,
		Request: con.ActiveSession.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %s\n", err.Error())
		return
	}
	if callExtension.Response != nil && callExtension.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", callExtension.Response.Err)
		return
	}
	con.PrintInfof("Sucessfuly executed %s\n", extName)
	if len(callExtension.Output) > 0 {
		con.PrintInfof("Got output:\n%s", string(callExtension.Output))
		if outFilePath != nil {
			outFilePath.Write(callExtension.Output)
			con.PrintInfof("Output saved to %s\n", outFilePath.Name())
		}
	}
}

func getBOFArgs(ctx *grumble.Context, binPath string, ext extensionCommand) ([]byte, error) {
	var extensionArgs []byte
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return nil, err
	}
	argsBuffer := BOFArgsBuffer{
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
	extensionArgsBuffer := BOFArgsBuffer{
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

func cmdExists(name string, app *grumble.App) bool {
	for _, c := range app.Commands().All() {
		if name == c.Name {
			return true
		}
	}
	return false
}

func init() {
	commandMap = make(map[string]extensionCommand)
}

// BOF Specific code

type BOFArgsBuffer struct {
	Buffer *bytes.Buffer
}

func (b *BOFArgsBuffer) AddData(d []byte) error {
	dataLen := uint32(len(d))
	err := binary.Write(b.Buffer, binary.LittleEndian, &dataLen)
	if err != nil {
		return err
	}
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddShort(d uint16) error {
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddInt(d uint32) error {
	return binary.Write(b.Buffer, binary.LittleEndian, &d)
}

func (b *BOFArgsBuffer) AddString(d string) error {
	stringLen := uint32(len(d)) + 1
	err := binary.Write(b.Buffer, binary.LittleEndian, &stringLen)
	if err != nil {
		return err
	}
	dBytes := append([]byte(d), 0x00)
	return binary.Write(b.Buffer, binary.LittleEndian, dBytes)
}

func (b *BOFArgsBuffer) AddWString(d string) error {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	strBytes := append([]byte(d), 0x00)
	utf16Data, err := encoder.Bytes(strBytes)
	if err != nil {
		return err
	}
	stringLen := uint32(len(utf16Data))
	err = binary.Write(b.Buffer, binary.LittleEndian, &stringLen)
	if err != nil {
		return err
	}
	return binary.Write(b.Buffer, binary.LittleEndian, utf16Data)
}

func (b *BOFArgsBuffer) GetBuffer() ([]byte, error) {
	outBuffer := new(bytes.Buffer)
	err := binary.Write(outBuffer, binary.LittleEndian, uint32(b.Buffer.Len()))
	if err != nil {
		return nil, err
	}
	err = binary.Write(outBuffer, binary.LittleEndian, b.Buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return outBuffer.Bytes(), nil
}
