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
	"context"
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
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders/traffic"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	rcpGenLog = log.NamedLogger("rpc", "generate")
)

// Generate - Generate a new implant
func (rpc *Server) Generate(ctx context.Context, req *clientpb.GenerateReq) (*clientpb.Generate, error) {
	var fPath string
	var err error
	name, config := generate.ImplantConfigFromProtobuf(req.Config)
	if name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, err
		}
	}
	if config.TemplateName == "" {
		config.TemplateName = generate.SliverTemplateName
	}

	err = generate.GenerateConfig(name, config, true)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("invalid implant config")
	}
	switch req.Config.Format {
	case clientpb.OutputFormat_SERVICE:
		fallthrough
	case clientpb.OutputFormat_EXECUTABLE:
		fPath, err = generate.SliverExecutable(name, config, true)
	case clientpb.OutputFormat_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(name, config, true)
	case clientpb.OutputFormat_SHELLCODE:
		fPath, err = generate.SliverShellcode(name, config, true)
	default:
		return nil, fmt.Errorf("invalid output format: %s", req.Config.Format)
	}

	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(fPath)
	fileData, err := os.ReadFile(fPath)
	if err != nil {
		return nil, err
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

// Regenerate - Regenerate a previously generated implant
func (rpc *Server) Regenerate(ctx context.Context, req *clientpb.RegenerateReq) (*clientpb.Generate, error) {

	build, err := db.ImplantBuildByName(req.ImplantName)
	if err != nil {
		rpcLog.Errorf("Failed to find implant %s: %s", req.ImplantName, err)
		return nil, status.Error(codes.InvalidArgument, "invalid implant name")
	}

	fileData, err := generate.ImplantFileFromBuild(build)
	if err != nil {
		return nil, err
	}

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: build.ImplantConfig.FileName,
			Data: fileData,
		},
	}, nil
}

// ImplantBuilds - List existing implant builds
func (rpc *Server) ImplantBuilds(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	dbBuilds, err := db.ImplantBuilds()
	if err != nil {
		return nil, err
	}
	pbBuilds := &clientpb.ImplantBuilds{
		Configs: map[string]*clientpb.ImplantConfig{},
	}
	for _, dbBuild := range dbBuilds {
		pbBuilds.Configs[dbBuild.Name] = dbBuild.ImplantConfig.ToProtobuf()
	}
	return pbBuilds, nil
}

// Canaries - List existing canaries
func (rpc *Server) Canaries(ctx context.Context, _ *commonpb.Empty) (*clientpb.Canaries, error) {
	dbCanaries, err := db.ListCanaries()
	if err != nil {
		return nil, err
	}

	rpcLog.Infof("Found %d canaries", len(dbCanaries))
	canaries := []*clientpb.DNSCanary{}
	for _, canary := range dbCanaries {
		canaries = append(canaries, canary.ToProtobuf())
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
		return nil, err
	}

	return &clientpb.UniqueWGIP{
		IP: uniqueIP.String(),
	}, nil
}

// ImplantProfiles - List profiles
func (rpc *Server) ImplantProfiles(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantProfiles, error) {
	implantProfiles := &clientpb.ImplantProfiles{
		Profiles: []*clientpb.ImplantProfile{},
	}
	dbProfiles, err := db.ImplantProfiles()
	if err != nil {
		return implantProfiles, err
	}
	for _, dbProfile := range dbProfiles {
		implantProfiles.Profiles = append(implantProfiles.Profiles, &clientpb.ImplantProfile{
			Name:   dbProfile.Name,
			Config: dbProfile.ImplantConfig.ToProtobuf(),
		})
	}
	return implantProfiles, nil
}

// SaveImplantProfile - Save a new profile
func (rpc *Server) SaveImplantProfile(ctx context.Context, profile *clientpb.ImplantProfile) (*clientpb.ImplantProfile, error) {
	_, config := generate.ImplantConfigFromProtobuf(profile.Config)
	profile.Name = filepath.Base(profile.Name)
	if 0 < len(profile.Name) && profile.Name != "." {
		rpcLog.Infof("Saving new profile with name %#v", profile.Name)
		err := generate.SaveImplantProfile(profile.Name, config)
		if err != nil {
			return nil, err
		}
		core.EventBroker.Publish(core.Event{
			EventType: consts.ProfileEvent,
			Data:      []byte(profile.Name),
		})
		return profile, nil
	}
	return nil, errors.New("invalid profile name")
}

// DeleteImplantProfile - Delete an implant profile
func (rpc *Server) DeleteImplantProfile(ctx context.Context, req *clientpb.DeleteReq) (*commonpb.Empty, error) {
	profile, err := db.ProfileByName(req.Name)
	if err != nil {
		return nil, err
	}
	err = db.Session().Delete(profile).Error
	if err == nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.ProfileEvent,
			Data:      []byte(profile.Name),
		})
	}
	return &commonpb.Empty{}, err
}

// DeleteImplantBuild - Delete an implant build
func (rpc *Server) DeleteImplantBuild(ctx context.Context, req *clientpb.DeleteReq) (*commonpb.Empty, error) {
	build, err := db.ImplantBuildByName(req.Name)
	if err != nil {
		return nil, err
	}
	err = db.Session().Delete(build).Error
	if err != nil {
		return nil, err
	}
	err = generate.ImplantFileDelete(build)
	if err == nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.BuildEvent,
			Data:      []byte(build.Name),
		})
	}
	return &commonpb.Empty{}, err
}

// ShellcodeRDI - Generates a RDI shellcode from a given DLL
func (rpc *Server) ShellcodeRDI(ctx context.Context, req *clientpb.ShellcodeRDIReq) (*clientpb.ShellcodeRDI, error) {
	shellcode, err := generate.ShellcodeRDIFromBytes(req.GetData(), req.GetFunctionName(), req.GetArguments())
	return &clientpb.ShellcodeRDI{Data: shellcode}, err
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
	var err error
	name, config := generate.ImplantConfigFromProtobuf(req.Config)
	if name == "" {
		name, err = codenames.GetCodename()
		if err != nil {
			return nil, err
		}
	}
	if config == nil {
		return nil, errors.New("invalid implant config")
	}
	externalConfig, err := generate.SliverExternal(name, config)
	if err != nil {
		return nil, err
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.ExternalBuildEvent,
		Data:      []byte(fmt.Sprintf("%s:%s", req.BuilderName, config.ID.String())),
	})

	return externalConfig, err
}

// GenerateExternalSaveBuild - Allows an external builder to save the build to the server
func (rpc *Server) GenerateExternalSaveBuild(ctx context.Context, req *clientpb.ExternalImplantBinary) (*commonpb.Empty, error) {
	implantConfig, err := db.ImplantConfigWithC2sByID(req.ImplantConfigID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid implant config id")
	}
	if implantConfig.TemplateName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid payload name")
	}
	err = util.AllowedName(req.Name)
	if err != nil {
		rpcLog.Errorf("Invalid build name: %s", err)
		return nil, ErrInvalidName
	}
	_, err = db.ImplantBuildByName(req.Name)
	if err == nil {
		rpcLog.Errorf("Build '%s' already exists!", req.Name)
		return nil, ErrBuildExists
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
	rpcLog.Infof("Saving external build '%s' from %s", req.Name, tmpFile.Name())

	implantConfig.FileName = req.File.Name
	generate.ImplantConfigSave(implantConfig)
	err = generate.ImplantBuildSave(req.Name, implantConfig, tmpFile.Name())
	if err != nil {
		rpcLog.Errorf("Failed to save external build: %s", err)
		return nil, err
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(req.Name),
	})

	return &commonpb.Empty{}, nil
}

// GenerateExternalGetImplantConfig - Get an implant config for external builder
func (rpc *Server) GenerateExternalGetImplantConfig(ctx context.Context, req *clientpb.ImplantConfig) (*clientpb.ExternalImplantConfig, error) {
	implantConfig, err := db.ImplantConfigWithC2sByID(req.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid implant config id")
	}
	if implantConfig.ImplantBuildID != uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "implant config already has a build")
	}

	return &clientpb.ExternalImplantConfig{
		Config: implantConfig.ToProtobuf(),
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
				return err
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
		fallthrough
	case consts.AcknowledgeBuildEvent:
		fallthrough
	case consts.ExternalBuildCompletedEvent:
		core.EventBroker.Publish(core.Event{
			EventType: req.EventType,
			Data:      req.Data,
		})

	}

	return &commonpb.Empty{}, nil
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
		return nil, err
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
		return nil, err
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
