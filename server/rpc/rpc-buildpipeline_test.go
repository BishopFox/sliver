package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

type testRegisteredCoreAdapter struct{}

func (testRegisteredCoreAdapter) ExecutorID() string { return "test-core-adapter-m3" }

func (testRegisteredCoreAdapter) Supports(target buildplan.Target, toolchain buildplan.Toolchain) bool {
	return target.OS == "windows" && target.Arch == "amd64" && toolchain.Driver == "go"
}

func (testRegisteredCoreAdapter) Build(_ context.Context, plan buildplan.Plan) (buildcore.Result, error) {
	if plan.Steps[0].Spec.Name != "test/core" {
		return buildcore.Result{}, errors.New("unexpected core spec")
	}
	return buildcore.Result{
		BuildID: "test-core-build", ImplantName: "example", FileName: "example.exe",
		Data: testCorePE(), ToolchainIdentity: "test compiler",
	}, nil
}

func testCorePE() []byte {
	data := make([]byte, 0x188)
	copy(data, "MZ")
	binary.LittleEndian.PutUint32(data[0x3c:], 0x80)
	copy(data[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(data[0x84:], 0x8664) // AMD64.
	binary.LittleEndian.PutUint16(data[0x94:], 0xf0)   // PE32+ optional header size.
	binary.LittleEndian.PutUint16(data[0x96:], 0x0002) // Executable.
	binary.LittleEndian.PutUint16(data[0x98:], 0x20b)  // PE32+.
	binary.LittleEndian.PutUint32(data[0x98+108:], 16) // Data directory count.
	return data
}

func TestBuildCatalogRPCFiltersAndExactSpec(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	list, err := rpc.ListBuildSpecs(context.Background(), &clientpb.BuildSpecQuery{
		Kind: clientpb.BuildSpecKind_BUILD_SPEC_CORE, TargetGOOS: "windows", TargetGOARCH: "amd64",
		Availability: clientpb.BuildSpecAvailability_BUILD_SPEC_AVAILABLE,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Specs) != 1 || list.Specs[0].Ref.Name != "builtin/sliver" {
		t.Fatalf("unexpected core catalog result: %+v", list.Specs)
	}
	got, err := rpc.GetBuildSpec(context.Background(), &clientpb.BuildSpecRequest{
		Kind: clientpb.BuildSpecKind_BUILD_SPEC_CORE, Ref: list.Specs[0].Ref,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ref.Digest != list.Specs[0].Ref.Digest || !bytes.Contains(got.YAML, []byte("kind: CoreSpec")) {
		t.Fatalf("exact spec lost immutable identity or YAML: %+v", got.Ref)
	}
	for _, capability := range got.Capabilities {
		if capability.Output == nil || capability.ToolchainDriver == "" {
			t.Fatalf("capability missing typed output or compiler requirements: %+v", capability)
		}
	}
	unavailable, err := rpc.ListBuildSpecs(context.Background(), &clientpb.BuildSpecQuery{
		Kind:         clientpb.BuildSpecKind_BUILD_SPEC_CORE,
		Availability: clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(unavailable.Specs) != 0 {
		t.Fatalf("available builtin appeared in unavailable filter: %+v", unavailable.Specs)
	}
}

//nolint:gocyclo // The test observes the same catalog spec before and after a builder disconnect.
func TestBuildCatalogRPCPreservesOfflineRemoteSpec(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := catalog.Get(buildcatalog.Reference{Name: "builtin/xor"})
	if err != nil {
		t.Fatal(err)
	}
	raw := bytes.Replace(builtin.RawYAML, []byte("name: builtin/xor"), []byte("name: test/xor"), 1)
	if bytes.Equal(raw, builtin.RawYAML) {
		t.Fatal("fixture did not change the spec name")
	}
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	metadata := document["metadata"].(map[string]any)
	previousDigest := metadata["digest"].(string)
	delete(metadata, "digest")
	canonical, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	raw = bytes.Replace(raw, []byte(previousDigest), []byte("sha256:"+hex.EncodeToString(digest[:])), 1)
	seen := time.Now().UTC().Truncate(time.Second)
	if err := catalog.AdvertiseRemoteBatch("worker-one", [][]byte{raw}, seen); err != nil {
		t.Fatal(err)
	}
	query := &clientpb.BuildSpecQuery{
		Kind: clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM, Name: "test/xor",
		Availability: clientpb.BuildSpecAvailability_BUILD_SPEC_AVAILABLE,
	}
	online, err := rpc.ListBuildSpecs(context.Background(), query)
	if err != nil || len(online.Specs) != 1 || !online.Specs[0].Available || len(online.Specs[0].Providers) != 1 || !online.Specs[0].Providers[0].Online {
		t.Fatalf("online remote spec was not listed: %+v, %v", online, err)
	}
	if err := catalog.SetRemoteOnline("worker-one", false, seen); err != nil {
		t.Fatal(err)
	}
	query.Availability = clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE
	offline, err := rpc.ListBuildSpecs(context.Background(), query)
	if err != nil || len(offline.Specs) != 1 || offline.Specs[0].Available || offline.Specs[0].Providers[0].Online {
		t.Fatalf("offline remote spec was not preserved: %+v, %v", offline, err)
	}
	if offline.Specs[0].Ref.Digest != online.Specs[0].Ref.Digest || !bytes.Equal(offline.Specs[0].YAML, raw) {
		t.Fatalf("offline spec lost its immutable identity or YAML")
	}
}

func TestPlanBuildRPCIsDeterministicWithoutCreatingRecord(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	request := &clientpb.BuildRequest{
		Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}, Mode: "session", Format: "exe",
		C2: []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: "mtls://c2.example.invalid:8888"}},
	}
	first, err := rpc.PlanBuild(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rpc.PlanBuild(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanSHA256 == "" || first.PlanSHA256 != second.PlanSHA256 || len(first.Steps) != 1 ||
		first.Steps[0].Spec.Digest == "" || first.Steps[0].Provider != "server" || first.FinalArtifact.Kind != "executable" {
		t.Fatalf("PlanBuild did not resolve a stable exact route: %+v", first)
	}
	if _, err := rpc.buildRecordStore(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(os.Getenv("SLIVER_ROOT_DIR"), "build-records"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("PlanBuild unexpectedly wrote %d records", len(entries))
	}
}

func TestCoreAdapterForPlanSelectsRegisteredExecutor(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	plan, err := rpc.resolveBuildPlan(context.Background(), buildplan.Request{
		Target: buildplan.Target{OS: "windows", Arch: "amd64"}, Mode: "session", FinalFormat: "exe",
		C2: []buildplan.Endpoint{{Protocol: "mtls", Address: "mtls://c2.example.invalid:8888"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := rpc.coreAdapterForPlan(plan)
	if err != nil || adapter == nil || adapter.ExecutorID() != "builtin-sliver-go-v1" {
		t.Fatalf("resolved core did not select its registered executor: %v, %v", adapter, err)
	}
	plan.Steps[0].Provider = "remote-builder"
	if _, err := rpc.coreAdapterForPlan(plan); status.Code(buildPlanRPCError(err)) != codes.Unimplemented {
		t.Fatalf("remote core should wait for M5 execution: %v", err)
	}
}

//nolint:gocyclo // Registration, execution, and artifact assertions exercise the same fixture.
func TestRegisteredCoreAdapterExecutesWithoutSliverChanges(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	adapter := testRegisteredCoreAdapter{}
	if _, exists := buildcore.DefaultAdapter(adapter.ExecutorID()); !exists {
		if err := buildcore.RegisterDefault(adapter); err != nil {
			t.Fatal(err)
		}
	}
	rpc := &Server{}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := catalog.Get(buildcatalog.Reference{Name: "builtin/sliver"})
	if err != nil {
		t.Fatal(err)
	}
	raw := bytes.Replace(builtin.RawYAML, []byte("name: builtin/sliver"), []byte("name: test/core"), 1)
	raw = bytes.Replace(raw, []byte("id: builtin-sliver-go-v1"), []byte("id: "+adapter.ExecutorID()), 1)
	raw = bytes.Replace(raw,
		[]byte("commandSource: installed-catalog\n  commandCatalog: builtin-sliver-v1"),
		[]byte("commandSource: declared-subset\n  commands:\n  - MsgPing"), 1)
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	metadata := document["metadata"].(map[string]any)
	previousDigest := metadata["digest"].(string)
	delete(metadata, "digest")
	canonical, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	raw = bytes.Replace(raw, []byte(previousDigest), []byte("sha256:"+hex.EncodeToString(digest[:])), 1)
	spec, err := catalog.ImportAdministrator(raw)
	if err != nil {
		t.Fatal(err)
	}
	record, err := rpc.ExecuteBuild(context.Background(), &clientpb.BuildRequest{
		Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}, Mode: "session", Format: "exe",
		C2:   []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: "mtls://c2.example.invalid:8888"}},
		Core: &clientpb.BuildSpecRef{Name: spec.Reference.Name, Version: spec.Reference.Version, Digest: spec.Reference.Digest},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "test-core-build" || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED ||
		record.FinalArtifact == nil || record.FinalArtifact.FileName != "example.exe" {
		t.Fatalf("new core was not dispatched and recorded: %+v", record)
	}
	file, err := rpc.GetBuildArtifact(context.Background(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || file.File == nil || !bytes.Equal(file.File.Data, testCorePE()) {
		t.Fatalf("new core output was not frozen: %+v, %v", file, err)
	}
}

func TestSliverAdapterConfigFromPlanSupportsConverterOptions(t *testing.T) {
	plan := buildplan.Plan{
		Request: buildplan.Request{
			Target: buildplan.Target{OS: "windows", Arch: "amd64"}, Mode: "session", FinalFormat: "shellcode",
			C2:               []buildplan.Endpoint{{Protocol: "wireguard", Address: "wireguard://192.0.2.10:53"}},
			IncludeProtocols: []string{"mtls"},
		},
		Steps: []buildplan.Step{
			{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "builtin/sliver"}, Provider: "server", Options: []buildplan.Option{
				{Name: "exports", ValueJSON: json.RawMessage(`["StartW"]`)},
				{Name: "obfuscateSymbols", ValueJSON: json.RawMessage(`true`)},
			}},
			{ID: "transform-1", Kind: "transform", Spec: buildplan.Reference{Name: "builtin/donut"}, Provider: "server", Options: []buildplan.Option{
				{Name: "compress", ValueJSON: json.RawMessage(`true`)},
				{Name: "exitOption", ValueJSON: json.RawMessage(`2`)},
			}},
		},
	}
	config, err := (generate.SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if config.Format != clientpb.OutputFormat_SHELLCODE || !config.IsShellcode || !config.IncludeWG || !config.IncludeMTLS {
		t.Fatalf("incorrect shellcode config: %+v", config)
	}
	if len(config.C2) != 1 || config.C2[0].URL != "wg://192.0.2.10:53" {
		t.Fatalf("legacy WireGuard URL was not normalized: %+v", config.C2)
	}
	if config.ShellcodeConfig.Compress != 2 || config.ShellcodeConfig.ExitOpt != 2 || !config.ObfuscateSymbols {
		t.Fatalf("converter/core options were dropped: %+v", config)
	}
}

func TestSliverAdapterConfigFromPlanAcceptsOrderedTransform(t *testing.T) {
	plan := buildplan.Plan{
		Request: buildplan.Request{Target: buildplan.Target{OS: "linux", Arch: "amd64"}, FinalFormat: "shellcode"},
		Steps: []buildplan.Step{
			{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "builtin/sliver"}, Provider: "server"},
			{ID: "transform-1", Kind: "transform", Spec: buildplan.Reference{Name: "builtin/malasada"}, Provider: "server"},
			{ID: "transform-2", Kind: "transform", Spec: buildplan.Reference{Name: "builtin/xor"}, Provider: "server"},
		},
	}
	config, err := (generate.SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil || config.Format != clientpb.OutputFormat_SHELLCODE {
		t.Fatalf("ordered converter and encoder should be accepted: %+v, %v", config, err)
	}
	plan.Steps[2].Provider = "remote-builder"
	config, err = (generate.SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil || config.Format != clientpb.OutputFormat_SHELLCODE {
		t.Fatalf("remote transform should preserve shellcode core configuration: %+v, %v", config, err)
	}
}

//nolint:gocyclo // The record and downloaded bytes are checked for the same frozen build.
func TestBuildRecordRPCServesFrozenBytes(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildplan.Artifact{Kind: "executable", Container: "elf", Target: buildplan.Target{OS: "linux", Arch: "amd64"}, GoBuildMode: "default"}
	plan := buildplan.Plan{Steps: []buildplan.Step{{ID: "core", Kind: "core", Output: artifact}}, FinalArtifact: artifact}
	record, err := store.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.GetBuildArtifact(context.Background(), &clientpb.BuildRecordRef{ID: record.ID}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("pending artifact should be unavailable, got %v", err)
	}
	data := []byte("frozen build bytes")
	hash := sha256.Sum256(data)
	info := buildplan.ArtifactInfo{Type: artifact, Name: "sliver.elf", Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
	if _, err := store.UpdateStep(record.ID, 0, buildplan.StepUpdate{State: buildplan.StepRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateStep(record.ID, 0, buildplan.StepUpdate{State: buildplan.StepComplete, Progress: 100, Artifact: &info}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Complete(record.ID, info, data); err != nil {
		t.Fatal(err)
	}
	got, err := rpc.GetBuildRecord(context.Background(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || got.Steps[0].ProgressPercent != 100 ||
		got.FinalArtifact.SHA256 != info.SHA256 || got.FinalRevision != 1 || len(got.FinalHistory) != 1 ||
		got.FinalHistory[0].Artifact.SHA256 != info.SHA256 {
		t.Fatalf("record lost state, progress, or hash: %+v", got)
	}
	file, err := rpc.GetBuildArtifact(context.Background(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(file.File.Data, data) || file.Metadata.SHA256 != info.SHA256 || !strings.HasSuffix(file.File.Name, ".elf") {
		t.Fatalf("artifact response does not match frozen bytes: %+v", file.Metadata)
	}
	revisedData := []byte("frozen build bytes with metadata")
	revisedHash := sha256.Sum256(revisedData)
	revisedInfo := info
	revisedInfo.Size = int64(len(revisedData))
	revisedInfo.SHA256 = hex.EncodeToString(revisedHash[:])
	if _, err := store.ReplaceFinal(record.ID, revisedInfo, revisedData); err != nil {
		t.Fatal(err)
	}
	revised, err := rpc.GetBuildRecord(context.Background(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if revised.FinishedUnix != got.FinishedUnix || revised.FinalRevision != 2 || len(revised.FinalHistory) != 2 ||
		revised.FinalHistory[0].Artifact.SHA256 != info.SHA256 || revised.FinalHistory[1].Artifact.SHA256 != revisedInfo.SHA256 {
		t.Fatalf("final revision changed original build completion or lost history: %+v", revised)
	}
	latest, err := rpc.GetBuildArtifact(context.Background(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || !bytes.Equal(latest.File.Data, revisedData) {
		t.Fatalf("artifact RPC did not serve latest revision: %v", err)
	}
}
