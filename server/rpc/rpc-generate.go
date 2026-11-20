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
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
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
	if req == nil || req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "missing implant config")
	}
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
	}

	adapter := generate.SliverCoreAdapter{KeepWorkspace: req.GetRetainWorkspace()}
	if err := adapter.Normalize(config); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	planRequest, err := legacyBuildRequest(name, config)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if snapshot, ok := ctx.Value(profileSnapshotContextKey{}).(*buildplan.ProfileSnapshot); ok {
		planRequest.ProfileSnapshot = snapshot
	}
	planRequest, resolveInput, err := legacyMetadataBuildRequest(planRequest, config, req.GetSpoofMetadata())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	plan, err := rpc.resolveBuildPlan(ctx, planRequest)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	if req.GetRetainWorkspace() && hasRemoteStep(plan) {
		return nil, status.Error(codes.InvalidArgument, "retaining a build workspace requires a local server provider")
	}
	if snapshot := planRequest.ProfileSnapshot; snapshot != nil && snapshot.Migrated {
		if err := verifyMigratedProfileSnapshot(snapshot, plan); err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}
	result, _, err := rpc.executeResolvedPlan(ctx, plan, adapter, config, resolveInput)
	if err != nil {
		return nil, rpcError(err)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(result.FileName),
	})

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: result.FileName,
			Data: result.Data,
		},
		ImplantName:    result.ImplantName,
		ImplantBuildID: result.BuildID,
		WorkspacePath:  result.WorkspacePath,
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

	fileName := build.Name + config.Extension
	var fileData []byte
	store, err := rpc.buildRecordStore()
	if err != nil {
		return nil, rpcError(err)
	}
	fileData, final, err := store.FinalBytes(build.ID)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, buildplan.ErrArtifactNotReady) {
		// Builds made before build records were introduced still use the
		// existing database-backed artifact path. A record interrupted while
		// being persisted may also have no final bytes yet.
		fileData, err = generate.ImplantFileFromBuild(build)
	} else if err == nil {
		if build.SHA256 != "" && build.SHA256 != final.SHA256 {
			// A legacy post-processing RPC may have updated the database file
			// before its build-record revision could be persisted.
			fileData, err = generate.ImplantFileFromBuild(build)
		} else {
			fileName = final.Name
		}
	}
	if err != nil {
		return nil, rpcError(err)
	}

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: fileName,
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
	if err = rpc.replaceRecordedFinal(build.ID, modifiedBuildData); err != nil {
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
	config, err := generate.SpoofMetadataConfigFromProto(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return config, nil
}

// ImplantBuilds - List existing implant builds
func (rpc *Server) ImplantBuilds(_ context.Context, _ *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	builds, ids, err := db.ImplantBuildsWithIDs()
	if err != nil {
		return nil, rpcError(err)
	}
	if len(builds.Configs) == 0 {
		return builds, nil
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		rcpGenLog.Warnf("Could not open build records for implant listing: %v", err)
		return builds, nil
	}
	builds.CoreSpecs = make(map[string]*clientpb.BuildSpecRef, len(builds.Configs))
	for name, id := range ids {
		record, err := store.Get(id)
		if errors.Is(err, os.ErrNotExist) {
			continue // Builds predating the record store have no exact core identity.
		}
		if err != nil {
			rcpGenLog.Warnf("Could not read build record for %s: %v", id, err)
			continue
		}
		if record.State != buildplan.BuildComplete || len(record.Plan.Steps) == 0 || record.Plan.Steps[0].Kind != "core" {
			continue
		}
		ref := record.Plan.Steps[0].Spec
		if ref.Name != "" {
			builds.CoreSpecs[name] = buildReferenceProto(ref)
		}
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
	if profile != nil && profile.Config != nil && validProfileLookupName(profile.Name) {
		rpcLog.Infof("Saving new profile with name %#v", profile.Name)
		profile, err := generate.SaveImplantProfile(profile)
		if err != nil {
			return nil, rpcError(err)
		}
		if err := rpc.upsertLegacyProfileDocument(ctx, profile); err != nil {
			rpcLog.Warnf("Legacy profile %q saved without canonical BuildProfile mirror: %v", profile.Name, err)
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
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing profile name")
	}
	return rpc.DeleteBuildProfile(ctx, &clientpb.BuildProfileRef{Name: req.Name})
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

// BuilderRegister registers an authenticated assigned-step worker and streams
// only assignments from its private queue.
func (rpc *Server) BuilderRegister(req *clientpb.Builder, stream rpcpb.SliverRPC_BuilderRegisterServer) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "missing builder registration")
	}
	req.OperatorName = rpc.getClientCommonName(stream.Context())
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "missing builder name")
	}
	if !stepRegistrationRequested(req) || len(req.GetSpecYAML()) == 0 {
		return status.Error(codes.InvalidArgument, "builder must advertise immutable specs and assigned-step capabilities")
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return rpcError(err)
	}
	if err := core.AddBuilder(req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	registered := false
	defer func() {
		if registered {
			rpcEventsLog.Infof("Builder %s disconnected", req.Name)
			rpc.disconnectStepBuilder(req.Name)
			if err := catalog.SetRemoteOnline(req.Name, false, time.Now().UTC()); err != nil {
				rcpGenLog.Warnf("Failed to mark builder %s catalog specs offline: %s", req.Name, err)
			}
		}
		core.RemoveBuilderIfCurrent(req)
	}()
	raw := make([][]byte, 0, len(req.GetSpecYAML()))
	for _, spec := range req.GetSpecYAML() {
		raw = append(raw, []byte(spec))
	}
	if err := rpc.registerStepBuilderBeforeCatalog(req, catalog, raw); err != nil {
		return err
	}
	registered = true
	rpcEventsLog.Infof("Builder %s (%s) connected", req.Name, req.OperatorName)
	if err := stream.Send(&clientpb.Event{EventType: "builder-ready"}); err != nil {
		return rpcError(err)
	}
	signals := stepSignals(req.Name)
	if signals == nil {
		return status.Error(codes.Internal, "assigned-step queue is unavailable")
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-signals.stop:
			return status.Error(codes.Unavailable, "builder step lease expired; reconnect required")
		case assigned := <-signals.events:
			if err := stream.Send(assigned); err != nil {
				return rpcError(err)
			}
		}
	}
}

// Builders - Get a list of all builders
func (rpc *Server) Builders(ctx context.Context, _ *commonpb.Empty) (*clientpb.Builders, error) {
	return rpc.stepBuilderInventory(), nil
}

// TrafficEncoderMap - Get a map of the server's traffic encoders
func (rpc *Server) TrafficEncoderMap(ctx context.Context, _ *commonpb.Empty) (*clientpb.TrafficEncoderMap, error) {
	trafficEncoderMap := make(map[string]*clientpb.TrafficEncoder)
	for id, encoder := range encoders.TrafficEncoderMapSnapshot() {
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
	defer func() {
		_ = encoder.Close()
	}()

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
	if req == nil || req.Profile == "" {
		return nil, status.Error(codes.InvalidArgument, "missing stage profile")
	}
	name := req.Name
	if name == "" {
		generated, err := codenames.GetCodename()
		if err != nil {
			return nil, rpcError(err)
		}
		name = generated
	} else if err := util.AllowedName(name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	profile, err := db.ImplantProfileByName(req.Profile)
	if err != nil {
		return nil, rpcError(err)
	}
	if profile == nil || profile.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "stage profile has no implant configuration")
	}
	adapter := generate.SliverCoreAdapter{}
	if err := adapter.Normalize(profile.Config); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	baseRequest, err := legacyBuildRequest(name, profile.Config)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Resolve the profile's original output first. Stage framing can accept
	// several core formats; pinning this selection preserves the profile's
	// intended compiler output instead of letting the planner choose another.
	basePlan, err := rpc.resolveBuildPlan(ctx, baseRequest)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	stageRequest, resolveInput, err := legacyStageBuildRequest(baseRequest, basePlan.Steps[0].Output, req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	plan, err := rpc.resolveBuildPlan(ctx, stageRequest)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	result, _, err := rpc.executeResolvedPlan(ctx, plan, adapter, profile.Config, resolveInput)
	if err != nil {
		return nil, rpcError(err)
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.BuildCompletedEvent,
		Data:      []byte(result.FileName),
	})
	return &clientpb.Generate{
		File:        &commonpb.File{Name: result.FileName, Data: result.Data},
		ImplantName: result.ImplantName, ImplantBuildID: result.BuildID,
	}, nil
}
