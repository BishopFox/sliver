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
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	utilEncoders "github.com/bishopfox/sliver/util/encoders"
	"github.com/bishopfox/sliver/util/encoders/traffic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	rcpGenLog = log.NamedLogger("rpc", "generate")
)

// Generate - Generate a new implant
func (rpc *Server) Generate(ctx context.Context, req *clientpb.GenerateReq) (*clientpb.Generate, error) {
	var (
		err    error
		config *clientpb.ImplantConfig
		name   string
	)

	if req.Name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, rpcError(err)
		}
	} else if err := util.AllowedName(req.Name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else {
		name = req.Name
	}

	if req.Config.ID != "" {
		// if this is a profile reuse existing configuration
		config, err = db.ImplantConfigByID(req.Config.ID)
		if err != nil {
			return nil, rpcError(err)
		}
	} else {
		// configure c2 channels to enable
		config = req.Config
		config.IncludeMTLS = models.IsC2Enabled([]string{"mtls"}, config.C2)
		config.IncludeWG = models.IsC2Enabled([]string{"wg"}, config.C2)
		config.IncludeHTTP = models.IsC2Enabled([]string{"http", "https"}, config.C2)
		config.IncludeDNS = models.IsC2Enabled([]string{"dns"}, config.C2)
		config.IncludeNamePipe = models.IsC2Enabled([]string{"namedpipe"}, config.C2)
		config.IncludeTCP = models.IsC2Enabled([]string{"tcppivot"}, config.C2)
	}

	if len(config.Exports) == 0 {
		config.Exports = []string{"StartW"}
	}

	// generate config
	build, err := generate.GenerateConfig(name, config)
	if err != nil {
		return nil, rpcError(err)
	}

	// retrieve http c2 implant config
	httpC2Config, err := db.LoadHTTPC2ConfigByName(req.Config.HTTPC2ConfigName)
	if err != nil {
		return nil, rpcError(err)
	}

	var fPath string
	switch req.Config.Format {
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	case clientpb.OutputFormat_EXECUTABLE:
		fPath, err = generate.SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	case clientpb.OutputFormat_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(name, build, config, httpC2Config.ImplantConfig)
	case clientpb.OutputFormat_SHELLCODE:
		fPath, err = generate.SliverShellcode(name, build, config, httpC2Config.ImplantConfig)
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid output format: %s", req.Config.Format))
	}
	if err != nil {
		return nil, rpcError(err)
	}

	fileName := filepath.Base(fPath)
	fileData, err := os.ReadFile(fPath)
	if err != nil {
		return nil, rpcError(err)
	}

	err = generate.ImplantBuildSave(build, config, fPath)
	if err != nil {
		rpcLog.Errorf("Failed to save external build: %s", err)
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(fileName),
	})

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: fileName,
			Data: fileData,
		},
		ImplantName:    build.Name,
		ImplantBuildID: build.ID,
	}, err
}

// Regenerate - Regenerate a previously generated implant
func (rpc *Server) Regenerate(ctx context.Context, req *clientpb.RegenerateReq) (*clientpb.Generate, error) {

	build, err := db.ImplantBuildByName(req.ImplantName)
	if err != nil {
		rpcLog.Errorf("Failed to find implant %s: %s", req.ImplantName, err)
		return nil, status.Error(codes.InvalidArgument, "invalid implant name")
	}

	config, err := db.ImplantConfigByID(build.ImplantConfigID)
	if err != nil {
		return nil, rpcError(err)
	}

	fileData, err := generate.ImplantFileFromBuild(build)
	if err != nil {
		return nil, rpcError(err)
	}

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: build.Name + config.Extension,
			Data: fileData,
		},
		ImplantName:    build.Name,
		ImplantBuildID: build.ID,
	}, nil
}

// GenerateSpoofMetadata applies spoof metadata to an existing build artifact.
func (rpc *Server) GenerateSpoofMetadata(_ context.Context, req *clientpb.GenerateSpoofMetadataReq) (*commonpb.Empty, error) {
	build, err := spoofMetadataTargetBuild(req)
	if err != nil {
		return nil, err
	}

	spoofConfig, err := spoofMetadataConfigFromProto(req.GetSpoofMetadata())
	if err != nil {
		return nil, err
	}

	buildData, err := generate.ImplantFileFromBuild(build)
	if err != nil {
		return nil, rpcError(err)
	}

	tmpFile, err := os.CreateTemp(assets.GetRootAppDir(), "tmp-spoof-metadata-*")
	if err != nil {
		rpcLog.Errorf("Failed to create temporary file: %s", err)
		return nil, status.Error(codes.Internal, "failed to create temporary file")
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write(buildData); err != nil {
		_ = tmpFile.Close()
		rcpGenLog.Errorf("Failed to write temporary build file: %s", err)
		return nil, status.Error(codes.Internal, "failed to write temporary build file")
	}
	if err = tmpFile.Close(); err != nil {
		rcpGenLog.Errorf("Failed to close temporary build file: %s", err)
		return nil, status.Error(codes.Internal, "failed to close temporary build file")
	}

	if err = generate.SpoofMetadata(tmpFile.Name(), spoofConfig); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	modifiedBuildData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, rpcError(err)
	}

	if err = generate.ImplantBuildUpdateFile(build, modifiedBuildData); err != nil {
		rpcLog.Errorf("Failed to update build %s: %s", build.ID, err)
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(build.Name),
	})

	return &commonpb.Empty{}, nil
}

func spoofMetadataTargetBuild(req *clientpb.GenerateSpoofMetadataReq) (*clientpb.ImplantBuild, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	var (
		build *clientpb.ImplantBuild
		err   error
	)
	selected := 0

	if req.GetImplantBuildID() != "" {
		selected++
		build, err = db.ImplantBuildByID(req.GetImplantBuildID())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid implant build id")
		}
	}
	if req.GetImplantName() != "" {
		selected++
		build, err = db.ImplantBuildByName(req.GetImplantName())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid implant name")
		}
	}
	if req.GetResourceID() != 0 {
		selected++
		build, err = db.ImplantBuildByResourceID(req.GetResourceID())
		if err != nil || build.GetID() == "" {
			return nil, status.Error(codes.InvalidArgument, "invalid resource id")
		}
	}
	if selected != 1 {
		return nil, status.Error(codes.InvalidArgument, "exactly one build selector is required (implant_build_id, implant_name, or resource_id)")
	}
	return build, nil
}

func spoofMetadataConfigFromProto(req *clientpb.SpoofMetadataConfig) (*generate.SpoofMetadataConfig, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing spoof metadata configuration")
	}

	config := &generate.SpoofMetadataConfig{}
	if req.GetPE() != nil {
		resourceDirectory, err := imageResourceDirectoryFromProto(req.GetPE().GetResourceDirectory())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		exportDirectory, err := imageExportDirectoryFromProto(req.GetPE().GetExportDirectory())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		config.PE = &generate.PESpoofMetadataConfig{
			Source:                   spoofMetadataFileFromProto(req.GetPE().GetSource()),
			Icon:                     spoofMetadataFileFromProto(req.GetPE().GetIcon()),
			ResourceDirectory:        resourceDirectory,
			ResourceDirectoryEntries: imageResourceDirectoryEntriesFromProto(req.GetPE().GetResourceDirectoryEntries()),
			ResourceDataEntries:      imageResourceDataEntriesFromProto(req.GetPE().GetResourceDataEntries()),
			ExportDirectory:          exportDirectory,
		}
	}

	if config.PE == nil {
		return nil, status.Error(codes.InvalidArgument, "spoof metadata has no supported executable format")
	}
	return config, nil
}

func spoofMetadataFileFromProto(req *clientpb.SpoofMetadataFile) *generate.SpoofMetadataFile {
	if req == nil {
		return nil
	}
	return &generate.SpoofMetadataFile{
		Name: req.GetName(),
		Data: req.GetData(),
	}
}

func imageResourceDirectoryFromProto(req *clientpb.IMAGE_RESOURCE_DIRECTORY) (*generate.IMAGE_RESOURCE_DIRECTORY, error) {
	if req == nil {
		return nil, nil
	}
	major, err := uint16Field("pe.resource_directory.major_version", req.GetMajorVersion())
	if err != nil {
		return nil, err
	}
	minor, err := uint16Field("pe.resource_directory.minor_version", req.GetMinorVersion())
	if err != nil {
		return nil, err
	}
	namedEntries, err := uint16Field("pe.resource_directory.number_of_named_entries", req.GetNumberOfNamedEntries())
	if err != nil {
		return nil, err
	}
	idEntries, err := uint16Field("pe.resource_directory.number_of_id_entries", req.GetNumberOfIdEntries())
	if err != nil {
		return nil, err
	}
	return &generate.IMAGE_RESOURCE_DIRECTORY{
		Characteristics:      req.GetCharacteristics(),
		TimeDateStamp:        req.GetTimeDateStamp(),
		MajorVersion:         major,
		MinorVersion:         minor,
		NumberOfNamedEntries: namedEntries,
		NumberOfIdEntries:    idEntries,
	}, nil
}

func imageResourceDirectoryEntriesFromProto(req []*clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY) []*generate.IMAGE_RESOURCE_DIRECTORY_ENTRY {
	if len(req) == 0 {
		return nil
	}
	entries := make([]*generate.IMAGE_RESOURCE_DIRECTORY_ENTRY, 0, len(req))
	for _, entry := range req {
		if entry == nil {
			continue
		}
		entries = append(entries, &generate.IMAGE_RESOURCE_DIRECTORY_ENTRY{
			Name:         entry.GetName(),
			OffsetToData: entry.GetOffsetToData(),
		})
	}
	return entries
}

func imageResourceDataEntriesFromProto(req []*clientpb.IMAGE_RESOURCE_DATA_ENTRY) []*generate.IMAGE_RESOURCE_DATA_ENTRY {
	if len(req) == 0 {
		return nil
	}
	entries := make([]*generate.IMAGE_RESOURCE_DATA_ENTRY, 0, len(req))
	for _, entry := range req {
		if entry == nil {
			continue
		}
		entries = append(entries, &generate.IMAGE_RESOURCE_DATA_ENTRY{
			OffsetToData: entry.GetOffsetToData(),
			Size:         entry.GetSize(),
			CodePage:     entry.GetCodePage(),
			Reserved:     entry.GetReserved(),
		})
	}
	return entries
}

func imageExportDirectoryFromProto(req *clientpb.IMAGE_EXPORT_DIRECTORY) (*generate.IMAGE_EXPORT_DIRECTORY, error) {
	if req == nil {
		return nil, nil
	}
	major, err := uint16Field("pe.export_directory.major_version", req.GetMajorVersion())
	if err != nil {
		return nil, err
	}
	minor, err := uint16Field("pe.export_directory.minor_version", req.GetMinorVersion())
	if err != nil {
		return nil, err
	}
	return &generate.IMAGE_EXPORT_DIRECTORY{
		Characteristics:       req.GetCharacteristics(),
		TimeDateStamp:         req.GetTimeDateStamp(),
		MajorVersion:          major,
		MinorVersion:          minor,
		Name:                  req.GetName(),
		Base:                  req.GetBase(),
		NumberOfFunctions:     req.GetNumberOfFunctions(),
		NumberOfNames:         req.GetNumberOfNames(),
		AddressOfFunctions:    req.GetAddressOfFunctions(),
		AddressOfNames:        req.GetAddressOfNames(),
		AddressOfNameOrdinals: req.GetAddressOfNameOrdinals(),
	}, nil
}

func uint16Field(fieldName string, value uint32) (uint16, error) {
	if value > 0xffff {
		return 0, fmt.Errorf("%s value %d exceeds uint16", fieldName, value)
	}
	return uint16(value), nil
}

// ImplantBuilds - List existing implant builds
func (rpc *Server) ImplantBuilds(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	builds, err := db.ImplantBuilds()
	if err != nil {
		return nil, rpcError(err)
	}
	return builds, nil
}

// StageImplantBuild - Serve a previously generated build
func (rpc *Server) StageImplantBuild(ctx context.Context, req *clientpb.ImplantStageReq) (*commonpb.Empty, error) {
	err := db.Session().Model(&models.ImplantBuild{}).Where("Stage = ?", true).Update("Stage", false).Error
	if err != nil {
		return nil, rpcError(err)
	}
	for _, name := range req.Build {
		err = db.Session().Model(&models.ImplantBuild{}).Where(&models.ImplantBuild{Name: name}).Update("Stage", true).Error
		if err != nil {
			return nil, rpcError(err)
		}
	}

	return &commonpb.Empty{}, nil
}

// Canaries - List existing canaries
func (rpc *Server) Canaries(ctx context.Context, _ *commonpb.Empty) (*clientpb.Canaries, error) {
	dbCanaries, err := db.ListCanaries()
	if err != nil {
		return nil, rpcError(err)
	}

	rpcLog.Infof("Found %d canaries", len(dbCanaries))
	canaries := []*clientpb.DNSCanary{}
	for _, canary := range dbCanaries {
		canaries = append(canaries, canary)
	}

	return &clientpb.Canaries{
		Canaries: canaries,
	}, nil
}

// GenerateUniqueIP - Wrapper around generate.GenerateUniqueIP
func (rpc *Server) GenerateUniqueIP(ctx context.Context, _ *commonpb.Empty) (*clientpb.UniqueWGIP, error) {
	uniqueIP, err := generate.GenerateUniqueIP()

	if err != nil {
		rpcLog.Infof("Failed to generate unique wg peer ip: %s\n", err)
		return nil, rpcError(err)
	}

	return &clientpb.UniqueWGIP{
		IP: uniqueIP.String(),
	}, nil
}

// ImplantProfiles - List profiles
func (rpc *Server) ImplantProfiles(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantProfiles, error) {
	implantProfiles, err := db.ImplantProfiles()
	if err != nil {
		return nil, rpcError(err)
	}

	return &clientpb.ImplantProfiles{Profiles: implantProfiles}, nil
}

// SaveImplantProfile - Save a new profile
func (rpc *Server) SaveImplantProfile(ctx context.Context, profile *clientpb.ImplantProfile) (*clientpb.ImplantProfile, error) {
	profile.Name = filepath.Base(profile.Name)
	if 0 < len(profile.Name) && profile.Name != "." {
		rpcLog.Infof("Saving new profile with name %#v", profile.Name)
		profile, err := generate.SaveImplantProfile(profile)
		if err != nil {
			return nil, rpcError(err)
		}
		core.EventBroker.Publish(core.Event{
			EventType: consts.ProfileEvent,
			Data:      []byte(profile.Name),
		})
		return profile, nil
	}
	return nil, status.Error(codes.InvalidArgument, "invalid profile name")
}

// DeleteImplantProfile - Delete an implant profile
func (rpc *Server) DeleteImplantProfile(ctx context.Context, req *clientpb.DeleteReq) (*commonpb.Empty, error) {
	profile, err := db.ProfileByName(req.Name)
	if err != nil {
		return nil, rpcError(err)
	}
	for _, build := range profile.Config.ImplantBuilds {
		err = RemoveBuildByName(build.Name)
		if err != nil {
			return nil, rpcError(err)
		}
	}
	err = db.DeleteProfile(req.Name)
	if err != nil {
		return nil, rpcError(err)
	}
	return &commonpb.Empty{}, nil
}

// DeleteImplantBuild - Delete an implant build
func (rpc *Server) DeleteImplantBuild(ctx context.Context, req *clientpb.DeleteReq) (*commonpb.Empty, error) {
	err := RemoveBuildByName(req.Name)
	if err != nil {
		return nil, rpcError(err)
	}
	return &commonpb.Empty{}, nil
}

// Remove Implant build given the build name
func RemoveBuildByName(name string) error {
	resourceID, err := db.ResourceIDByName(name)
	if err != nil {
		return err
	}

	build, err := db.ImplantBuildByName(name)
	if err != nil {
		return err
	}

	err = db.Session().Delete(build).Error
	if err != nil {
		return err
	}

	encoders.UnavailableID = util.RemoveElement(encoders.UnavailableID, resourceID.Value)
	err = db.Session().Where(&models.ResourceID{Name: name}).Delete(&models.ResourceID{}).Error
	if err != nil {
		return err
	}

	err = generate.ImplantFileDelete(build)
	if err == nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.BuildEvent,
			Data:      []byte(build.Name),
		})
	}
	return nil
}

// ShellcodeRDI - Generates a RDI shellcode from a given DLL
func (rpc *Server) ShellcodeRDI(ctx context.Context, req *clientpb.ShellcodeRDIReq) (*clientpb.ShellcodeRDI, error) {
	shellcode, err := generate.ShellcodeRDIFromBytes(req.GetData(), req.GetFunctionName(), req.GetArguments())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &clientpb.ShellcodeRDI{Data: shellcode}, nil
}

// GetCompiler - Get information about the internal Go compiler and its configuration
func (rpc *Server) GetCompiler(ctx context.Context, _ *commonpb.Empty) (*clientpb.Compiler, error) {
	compiler := &clientpb.Compiler{
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
		Targets:            generate.GetCompilerTargets(),
		UnsupportedTargets: generate.GetUnsupportedTargets(),
		CrossCompilers:     generate.GetCrossCompilers(),
	}
	rcpGenLog.Debugf("GetCompiler = %v", compiler)
	return compiler, nil
}

// *** External builder RPCs ***

// Generate - Generate a new implant
func (rpc *Server) GenerateExternal(ctx context.Context, req *clientpb.ExternalGenerateReq) (*clientpb.ExternalImplantConfig, error) {
	var (
		err  error
		name string
	)
	if req.BuilderName == "" {
		return nil, status.Error(codes.InvalidArgument, "missing builder name")
	}
	builder := core.GetBuilder(req.BuilderName)
	if builder == nil {
		return nil, status.Error(codes.NotFound, "unknown builder")
	}

	config := req.Config
	if req.Name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, rpcError(err)
		}
	} else if err := util.AllowedName(req.Name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else {
		name = req.Name
	}
	if config == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid implant config")
	}
	externalConfig, err := generate.SliverExternal(name, config)
	if err != nil {
		return nil, rpcError(err)
	}
	core.TrackExternalBuildAssignment(externalConfig.Build.ID, req.BuilderName, builder.OperatorName)

	core.EventBroker.Publish(core.Event{
		EventType: consts.ExternalBuildEvent,
		Data:      []byte(fmt.Sprintf("%s:%s", req.BuilderName, externalConfig.Build.ID)),
	})

	return externalConfig, err
}

// GenerateExternalSaveBuild - Allows an external builder to save the build to the server
func (rpc *Server) GenerateExternalSaveBuild(ctx context.Context, req *clientpb.ExternalImplantBinary) (*commonpb.Empty, error) {
	if req.GetFile() == nil {
		return nil, status.Error(codes.InvalidArgument, "missing build file")
	}
	if err := rpc.authorizeBuilderForExternalBuild(ctx, req.Name, req.ImplantBuildID); err != nil {
		return nil, err
	}

	implantBuild, err := db.ImplantBuildByID(req.ImplantBuildID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid implant build id")
	}

	implantConfig, err := db.ImplantConfigWithC2sByID(implantBuild.ImplantConfigID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid implant config id")
	}

	tmpFile, err := os.CreateTemp(assets.GetRootAppDir(), "tmp-external-build-*")
	if err != nil {
		rpcLog.Errorf("Failed to create temporary file: %s", err)
		return nil, status.Error(codes.Internal, "Failed to write implant binary to temp file")
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(req.File.Data)
	if err != nil {
		rcpGenLog.Errorf("Failed to write implant binary to temp file: %s", err)
		return nil, status.Error(codes.Internal, "Failed to write implant binary to temp file")
	}

	rpcLog.Infof("Saving external build %s from builder '%s' (%s)", req.ImplantBuildID, req.Name, tmpFile.Name())
	err = generate.ImplantBuildSave(implantBuild, implantConfig, tmpFile.Name())
	if err != nil {
		rpcLog.Errorf("Failed to save external build: %s", err)
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(implantBuild.Name),
	})

	return &commonpb.Empty{}, nil
}

// GenerateExternalGetImplantConfig - Get an implant config for external builder
func (rpc *Server) GenerateExternalGetBuildConfig(ctx context.Context, req *clientpb.ImplantBuild) (*clientpb.ExternalImplantConfig, error) {
	if err := rpc.authorizeBuilderForExternalBuild(ctx, req.Name, req.ID); err != nil {
		return nil, err
	}

	build, err := db.ImplantBuildByID(req.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid implant build id")
	}

	implantConfig, err := db.ImplantConfigWithC2sByID(build.ImplantConfigID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid implant config id")
	}

	// retrieve http c2 implant config
	httpC2Config, err := db.LoadHTTPC2ConfigByName(implantConfig.HTTPC2ConfigName)
	if err != nil {
		return nil, rpcError(err)
	}

	encoders := map[string]uint64{
		"base64":  encoders.Base64EncoderID,
		"base58":  encoders.Base58EncoderID,
		"base32":  encoders.Base32EncoderID,
		"hex":     encoders.HexEncoderID,
		"english": encoders.EnglishEncoderID,
		"gzip":    encoders.GzipEncoderID,
		"png":     encoders.PNGEncoderID,
	}

	return &clientpb.ExternalImplantConfig{
		Config:   implantConfig,
		Build:    build,
		HTTPC2:   httpC2Config,
		Encoders: encoders,
	}, nil
}

// BuilderRegister - Register a new builder with the server
func (rpc *Server) BuilderRegister(req *clientpb.Builder, stream rpcpb.SliverRPC_BuilderRegisterServer) error {
	req.OperatorName = rpc.getClientCommonName(stream.Context())
	if req.Name == "" {
		rcpGenLog.Warnf("Failed to register builder, missing builder name")
		return status.Error(codes.InvalidArgument, "missing builder name")
	}
	err := core.AddBuilder(req)
	if err != nil {
		rcpGenLog.Warnf("Failed to register builder: %s", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	rpcEventsLog.Infof("Builder %s (%s) connected", req.Name, req.OperatorName)
	events := core.EventBroker.Subscribe()
	defer func() {
		rpcEventsLog.Infof("Builder %s disconnected", req.Name)
		core.EventBroker.Unsubscribe(events)
		core.RemoveBuilder(req.Name)
	}()

	// Only forward these event types to the builder
	buildEvents := []string{
		consts.ExternalBuildEvent,
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-events:
			if !util.Contains(buildEvents, event.EventType) {
				continue // Skip events not relevant to the builder
			}
			if event.EventType == consts.ExternalBuildEvent {
				targetBuilderName, _, err := splitExternalBuildEventData(event.Data)
				if err != nil {
					rpcEventsLog.Warnf("Invalid external build event data: %s", err)
					continue
				}
				if targetBuilderName != req.Name {
					continue
				}
			}

			pbEvent := &clientpb.Event{
				EventType: event.EventType,
				Data:      event.Data,
			}
			if event.Job != nil {
				pbEvent.Job = event.Job.ToProtobuf()
			}
			if event.Client != nil {
				pbEvent.Client = event.Client.ToProtobuf()
			}
			if event.Session != nil {
				pbEvent.Session = event.Session.ToProtobuf()
			}
			if event.Err != nil {
				pbEvent.Err = event.Err.Error()
			}

			err := stream.Send(pbEvent)
			if err != nil {
				rpcEventsLog.Warnf(err.Error())
				return rpcError(err)
			}
		}
	}
}

// Builders - Get a list of all builders
func (rpc *Server) Builders(ctx context.Context, _ *commonpb.Empty) (*clientpb.Builders, error) {
	return &clientpb.Builders{Builders: core.AllBuilders()}, nil
}

// BuilderTrigger - Trigger a builder event
func (rpc *Server) BuilderTrigger(ctx context.Context, req *clientpb.Event) (*commonpb.Empty, error) {
	switch req.EventType {

	// Only allow certain event types to be triggered
	case consts.ExternalBuildFailedEvent:
		buildID, _, err := splitExternalBuildProgressData(req.Data)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if err = rpc.authorizeBuilderOperatorForExternalBuild(ctx, buildID); err != nil {
			return nil, err
		}
		core.RemoveExternalBuildAssignment(buildID)
		core.EventBroker.Publish(core.Event{
			EventType: req.EventType,
			Data:      req.Data,
		})
	case consts.AcknowledgeBuildEvent:
		buildID := strings.TrimSpace(string(req.Data))
		if buildID == "" {
			return nil, status.Error(codes.InvalidArgument, "missing implant build id")
		}
		err := rpc.authorizeBuilderOperatorForExternalBuild(ctx, buildID)
		if err != nil {
			return nil, err
		}
		core.EventBroker.Publish(core.Event{
			EventType: req.EventType,
			Data:      req.Data,
		})
	case consts.ExternalBuildCompletedEvent:
		buildID, _, err := splitExternalBuildProgressData(req.Data)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if err = rpc.authorizeBuilderOperatorForExternalBuild(ctx, buildID); err != nil {
			return nil, err
		}
		core.RemoveExternalBuildAssignment(buildID)
		core.EventBroker.Publish(core.Event{
			EventType: req.EventType,
			Data:      req.Data,
		})
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported builder event type")
	}

	return &commonpb.Empty{}, nil
}

func (rpc *Server) authorizeBuilderForExternalBuild(ctx context.Context, builderName string, buildID string) error {
	builderName = strings.TrimSpace(builderName)
	buildID = strings.TrimSpace(buildID)
	if builderName == "" {
		return status.Error(codes.InvalidArgument, "missing builder name")
	}
	if buildID == "" {
		return status.Error(codes.InvalidArgument, "missing implant build id")
	}

	callerName := rpc.getClientCommonName(ctx)
	if callerName == "" {
		return status.Error(codes.Unauthenticated, "Authentication failure")
	}
	builder := core.GetBuilder(builderName)
	if builder == nil {
		return status.Error(codes.PermissionDenied, "builder is not registered")
	}
	if builder.OperatorName != callerName {
		return status.Error(codes.PermissionDenied, "builder identity mismatch")
	}
	assignment := core.GetExternalBuildAssignment(buildID)
	if assignment == nil {
		return status.Error(codes.PermissionDenied, "build is not assigned")
	}
	if assignment.BuilderName != builderName || assignment.OperatorName != callerName {
		return status.Error(codes.PermissionDenied, "build is assigned to a different builder")
	}
	return nil
}

func (rpc *Server) authorizeBuilderOperatorForExternalBuild(ctx context.Context, buildID string) error {
	buildID = strings.TrimSpace(buildID)
	if buildID == "" {
		return status.Error(codes.InvalidArgument, "missing implant build id")
	}

	callerName := rpc.getClientCommonName(ctx)
	if callerName == "" {
		return status.Error(codes.Unauthenticated, "Authentication failure")
	}
	assignment := core.GetExternalBuildAssignment(buildID)
	if assignment == nil {
		return status.Error(codes.PermissionDenied, "build is not assigned")
	}
	if assignment.OperatorName != callerName {
		return status.Error(codes.PermissionDenied, "build is assigned to a different builder operator")
	}
	return nil
}

func splitExternalBuildEventData(data []byte) (builderName string, buildID string, err error) {
	payload := strings.TrimSpace(string(data))
	parts := strings.Split(payload, ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid build event data")
	}
	buildID = strings.TrimSpace(parts[len(parts)-1])
	builderName = strings.TrimSpace(strings.Join(parts[:len(parts)-1], ":"))
	if builderName == "" || buildID == "" {
		return "", "", fmt.Errorf("invalid build event data")
	}
	return builderName, buildID, nil
}

func splitExternalBuildProgressData(data []byte) (buildID string, details string, err error) {
	payload := strings.TrimSpace(string(data))
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid external build status data")
	}
	buildID = strings.TrimSpace(parts[0])
	details = strings.TrimSpace(parts[1])
	if buildID == "" {
		return "", "", fmt.Errorf("invalid external build status data")
	}
	return buildID, details, nil
}

// TrafficEncoderMap - Get a map of the server's traffic encoders
func (rpc *Server) TrafficEncoderMap(ctx context.Context, _ *commonpb.Empty) (*clientpb.TrafficEncoderMap, error) {
	trafficEncoderMap := make(map[string]*clientpb.TrafficEncoder)
	for id, encoder := range encoders.TrafficEncoderMap {
		trafficEncoderMap[encoder.FileName] = &clientpb.TrafficEncoder{
			ID: id,
			Wasm: &commonpb.File{
				Name: encoder.FileName,
				Data: encoder.Data,
			},
		}
	}
	return &clientpb.TrafficEncoderMap{Encoders: trafficEncoderMap}, nil
}

// TrafficEncoderAdd - Add a new traffic encoder, and test for correctness
func (rpc *Server) TrafficEncoderAdd(ctx context.Context, req *clientpb.TrafficEncoder) (*clientpb.TrafficEncoderTests, error) {
	req.ID = traffic.CalculateWasmEncoderID(req.Wasm.Data)
	req.Wasm.Name = filepath.Base(req.Wasm.Name)
	rpcLog.Infof("Adding new traffic encoder: %s (%d)", req.Wasm.Name, req.ID)
	progress := make(chan []byte, 1)
	go testProgress(progress)
	tests, err := testTrafficEncoder(ctx, req, progress)
	close(progress)
	if err != nil {
		return nil, rpcError(err)
	}
	return tests, nil
}

func testProgress(progress chan []byte) {
	for data := range progress {
		core.EventBroker.Publish(core.Event{
			EventType: consts.TrafficEncoderTestProgressEvent,
			Data:      data,
		})
	}
}

// testTrafficEncoder - Test a traffic encoder for correctness by encoding/decoding random samples
func testTrafficEncoder(ctx context.Context, req *clientpb.TrafficEncoder, progress chan []byte) (*clientpb.TrafficEncoderTests, error) {
	if req.Wasm == nil {
		return nil, status.Error(codes.InvalidArgument, "missing wasm file")
	}
	if req.Wasm.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing wasm name")
	}
	if !strings.HasSuffix(req.Wasm.Name, ".wasm") {
		return nil, status.Error(codes.InvalidArgument, "invalid wasm file name, must have a .wasm extension")
	}
	if req.Wasm.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "missing wasm data")
	}
	rpcLog.Infof("Testing traffic encoder %s (%d) - %d bytes", req.Wasm.Name, req.ID, len(req.Wasm.Data))
	encoder, err := traffic.CreateTrafficEncoder(strings.TrimSuffix(req.Wasm.Name, ".wasm"), req.Wasm.Data, func(s string) {
		rpcLog.Infof("[traffic encoder test] %s", s)
	})
	if err != nil {
		rpcLog.Errorf("Failed to create traffic encoder: %s", err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Test Suite for Traffic Encoders
	testSuite := []string{
		traffic.SmallRandom,
		traffic.SmallRandom,
		traffic.SmallRandom,
		traffic.MediumRandom,
		traffic.MediumRandom,
		traffic.MediumRandom,
		traffic.LargeRandom,
		traffic.LargeRandom,
		traffic.LargeRandom,
		traffic.VeryLargeRandom,
	}

	tests := []*clientpb.TrafficEncoderTest{}
	if !req.SkipTests {
		for index, testName := range testSuite {
			rpcLog.Infof("Running test '%s' ...", testName)
			tester := traffic.TrafficEncoderTesters[testName]
			if tester == nil {
				panic("invalid traffic encoder test")
			}
			test := tester(encoder)
			tests = append(tests, test)
			testData, _ := proto.Marshal(&clientpb.TrafficEncoderTests{
				Encoder:    req,
				TotalTests: int32(len(testSuite)),
				Tests:      tests,
			})
			progress <- testData
			if int64(time.Duration(30*time.Second)) < test.Duration {
				rpcLog.Warnf("Test '%s' took longer than 30 seconds to complete, skip remaining tests", testName)
				remainingTests := testSuite[index:]
				for _, skipTest := range remainingTests {
					tests = append(tests, &clientpb.TrafficEncoderTest{
						Name:    skipTest,
						Success: false,
						Err:     "test skipped, encoder too slow",
					})
				}
				break
			}
		}
	}

	if allTestsPassed(tests) || req.SkipTests {
		err = encoders.SaveTrafficEncoder(req.Wasm.Name, req.Wasm.Data)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}

	return &clientpb.TrafficEncoderTests{
		Encoder:    req,
		TotalTests: int32(len(testSuite)),
		Tests:      tests,
	}, nil
}

func allTestsPassed(tests []*clientpb.TrafficEncoderTest) bool {
	for _, test := range tests {
		if !test.Success {
			return false
		}
	}
	return true
}

// TrafficEncoderRm - Remove a traffic encoder
func (rpc *Server) TrafficEncoderRm(ctx context.Context, req *clientpb.TrafficEncoder) (*commonpb.Empty, error) {
	if req.Wasm == nil {
		return nil, status.Error(codes.InvalidArgument, "missing wasm file")
	}
	if req.Wasm.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing wasm file name")
	}
	if !strings.HasSuffix(req.Wasm.Name, ".wasm") {
		return nil, status.Error(codes.InvalidArgument, "invalid wasm file name, must have a .wasm extension")
	}
	err := encoders.RemoveTrafficEncoder(req.Wasm.Name)
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}
	return &commonpb.Empty{}, nil
}

// GenerateStage - Generate a new stage
func (rpc *Server) GenerateStage(ctx context.Context, req *clientpb.GenerateStageReq) (*clientpb.Generate, error) {
	var (
		err  error
		name string
	)

	profile, err := db.ImplantProfileByName(req.Profile)
	if err != nil {
		return nil, rpcError(err)
	}

	if req.Name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, rpcError(err)
		}
	} else if err := util.AllowedName(name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else {
		name = req.Name
	}

	// retrieve http c2 implant config
	httpC2Config, err := db.LoadHTTPC2ConfigByName(profile.Config.HTTPC2ConfigName)
	if err != nil {
		return nil, rpcError(err)
	}

	if len(profile.Config.Exports) == 0 {
		profile.Config.Exports = []string{"StartW"}
	}

	// generate config
	build, err := generate.GenerateConfig(name, profile.Config)
	if err != nil {
		return nil, rpcError(err)
	}

	var fPath string
	switch profile.Config.Format {
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	case clientpb.OutputFormat_EXECUTABLE:
		fPath, err = generate.SliverExecutable(name, build, profile.Config, httpC2Config.ImplantConfig)
	case clientpb.OutputFormat_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(name, build, profile.Config, httpC2Config.ImplantConfig)
	case clientpb.OutputFormat_SHELLCODE:
		fPath, err = generate.SliverShellcode(name, build, profile.Config, httpC2Config.ImplantConfig)
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid output format: %s", profile.Config.Format))
	}
	if err != nil {
		return nil, rpcError(err)
	}

	fileName := filepath.Base(fPath)
	fileData, err := os.ReadFile(fPath)
	if err != nil {
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(fileName),
	})

	var (
		stageType string
		stage2    []byte
	)

	if req.PrependSize {
		fileData = prependPayloadSize(fileData)
	}
	stage2 = fileData

	if req.Compress != "" {
		stageType = req.Compress + " - "
		stage2, err = Compress(stage2, req.Compress)
		if err != nil {
			return nil, rpcError(err)
		}
	}

	if req.AESEncryptKey != "" || req.RC4EncryptKey != "" {
		if req.RC4EncryptKey != "" {
			stageType += "RC4 - "
		} else {
			stageType += "AES - "
		}

		stage2, err = Encrypt(stage2, req)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	err = generate.SaveStage(build, profile.Config, stage2, stageType)
	if err != nil {
		rpcLog.Errorf("Failed to save external build: %s", err)
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(fileName),
	})

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: fileName,
			Data: fileData,
		},
	}, err
}

func Encrypt(stage2 []byte, req *clientpb.GenerateStageReq) ([]byte, error) {
	if req.RC4EncryptKey != "" && req.AESEncryptKey != "" {
		return nil, errors.New("Cannot use both RC4 and AES encryption\n")
	}
	if req.RC4EncryptKey != "" {
		// RC4 keysize can be between 1 to 256 bytes
		if len(req.RC4EncryptKey) < 1 || len(req.RC4EncryptKey) > 256 {
			return nil, errors.New("Incorrect length of RC4 Key\n")
		}
		stage2 = util.RC4EncryptUnsafe(stage2, []byte(req.RC4EncryptKey))
		return stage2, nil
	}

	if req.AESEncryptKey != "" {
		// check if aes encryption key is correct length
		if len(req.AESEncryptKey)%16 != 0 {
			return nil, errors.New("Incorrect length of AES Key\n")
		}

		// set default aes iv
		if req.AESEncryptIv == "" {
			req.AESEncryptIv = "0000000000000000"
		} else {
			// check if aes iv is correct length
			if len(req.AESEncryptIv)%16 != 0 {
				return nil, errors.New("Incorrect length of AES IV\n")
			}
		}
		stage2 = util.PreludeEncrypt(stage2, []byte(req.AESEncryptKey), []byte(req.AESEncryptIv))
		return stage2, nil
	}
	return stage2, nil
}

func Compress(stage2 []byte, compress string) ([]byte, error) {

	switch compress {
	case "zlib":
		// use zlib to compress the stage2
		var compBuff bytes.Buffer
		zlibWriter := zlib.NewWriter(&compBuff)
		zlibWriter.Write(stage2)
		zlibWriter.Close()
		stage2 = compBuff.Bytes()
	case "gzip":
		stage2, _ = utilEncoders.GzipBuf(stage2)
	case "deflate9":
		fallthrough
	case "deflate":
		stage2 = util.DeflateBuf(stage2)
	}
	return stage2, nil

}

func prependPayloadSize(payload []byte) []byte {
	payloadSize := uint32(len(payload))
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, payloadSize)
	return append(lenBuf, payload...)
}
