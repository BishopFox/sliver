package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildprofile"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func m6TestProfile(t *testing.T, rpc *Server, name, goos, goarch string) buildprofile.Profile {
	t.Helper()
	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	core, err := catalog.Get(buildcatalog.Reference{Name: "builtin/sliver"})
	if err != nil {
		t.Fatal(err)
	}
	request := buildplan.Request{
		Name: name, Target: buildplan.Target{OS: goos, Arch: goarch}, Mode: "session", FinalFormat: "exe",
		C2:       []buildplan.Endpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:8888"}},
		Core:     buildplan.Reference{Name: core.Reference.Name, Version: core.Reference.Version, Digest: core.Reference.Digest},
		Provider: buildplan.ProviderPolicy{Mode: "automatic"},
	}
	plan, err := staticProfilePlan(context.Background(), catalog, request)
	if err != nil {
		t.Fatal(err)
	}
	return buildprofile.Profile{
		APIVersion: "sliver.bishopfox.com/v1alpha1", Kind: "BuildProfile",
		Metadata: buildprofile.Metadata{Name: name, Revision: 1}, Target: request.Target, Mode: "session",
		C2: buildprofile.C2{Endpoints: request.C2, ConnectionStrategy: "random", ReconnectSeconds: 60,
			PollTimeoutSeconds: 360, MaxConnectionErrors: 1000, HTTPC2Profile: "default"},
		Core: request.Core, CoreOutput: buildprofile.ArtifactFromPlan(plan.Steps[0].Output),
		CoreOptions: []buildplan.Option{}, Transforms: []buildplan.TransformRequest{},
		FinalOutput: buildprofile.FinalOutput{Format: "exe", Artifact: buildprofile.ArtifactFromPlan(plan.FinalArtifact)},
		Provider:    buildprofile.Provider{Policy: "automatic"},
	}
}

func TestBuildProfileRPCImportExportRevisionAndOverrideValidation(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	name := "m6-import-" + models.NewUUID().String()
	profile := m6TestProfile(t, rpc, name, "windows", "amd64")
	yaml, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := rpc.ValidateBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(yaml)})
	if err != nil || validated.Name != name || validated.FinalArtifact.GOOS != "windows" || validated.FinalArtifact.Kind != "executable" {
		t.Fatalf("validate profile: %+v, %v", validated, err)
	}
	if _, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); status.Code(err) != codes.NotFound {
		t.Fatalf("validation persisted profile: %v", err)
	}
	imported, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(yaml)})
	if err != nil || imported.Migrated || imported.Revision != 1 {
		t.Fatalf("import profile: %+v, %v", imported, err)
	}
	got, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name})
	if err != nil || got.YAML != imported.YAML {
		t.Fatalf("export profile: %+v, %v", got, err)
	}
	list, err := rpc.ListBuildProfiles(t.Context(), &commonpb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list.Profiles {
		found = found || item.Name == name
	}
	if !found {
		t.Fatal("imported profile missing from list")
	}
	for label, request := range map[string]*clientpb.BuildProfileExecuteRequest{
		"unsafe name":           {Name: name, BuildName: "../escape"},
		"encoder on executable": {Name: name, ShellcodeEncoder: "xor"},
		"undeclared input":      {Name: name, Inputs: []*clientpb.BuildInput{{Kind: "secret", Name: "missing", Data: []byte("key")}}},
	} {
		t.Run(label, func(t *testing.T) {
			if _, err := rpc.ExecuteProfileBuild(t.Context(), request); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", err)
			}
		})
	}
	profile.Metadata.Revision = 2
	profile.C2.Endpoints[0].Address = "mtls://changed.example.invalid:8888"
	updatedYAML, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(updatedYAML)})
	if err != nil || updated.Revision != 2 || !strings.Contains(updated.YAML, "changed.example.invalid") || strings.Contains(imported.YAML, "changed.example.invalid") {
		t.Fatalf("profile edit mutated prior snapshot: %+v, %v", updated, err)
	}
	if _, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(yaml)}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("stale revision accepted: %v", err)
	}
	if _, err := rpc.DeleteBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted profile was visible: %v", err)
	}
}

func TestBuildProfileLegacyBackfillAndMigrationFailureIsolation(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	name := "m6-legacy-" + models.NewUUID().String()
	badName := "m6-invalid-" + models.NewUUID().String()
	makeLegacy := func(name string, reconnect int64) *clientpb.ImplantProfile {
		return &clientpb.ImplantProfile{Name: name, Config: &clientpb.ImplantConfig{
			GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE,
			C2:                []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
			ReconnectInterval: reconnect, PollTimeout: int64(360 * time.Second), MaxConnectionErrors: 1000,
			HTTPC2ConfigName: "default", Debug: true,
		}}
	}
	good, err := generate.SaveImplantProfile(makeLegacy(name, int64(60*time.Second)))
	if err != nil {
		t.Fatal(err)
	}
	bad, err := generate.SaveImplantProfile(makeLegacy(badName, int64(1500*time.Millisecond)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, item := range []*clientpb.ImplantProfile{good, bad} {
			_ = db.DeleteProfile(item.Name)
			_ = db.Session().Where("id = ?", item.Config.ID).Delete(&models.ImplantConfig{}).Error
		}
	})
	list, err := rpc.ListBuildProfiles(t.Context(), &commonpb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	var goodDoc, badDoc *clientpb.BuildProfileDocument
	for _, item := range list.Profiles {
		if item.Name == name {
			goodDoc = item
		}
		if item.Name == badName {
			badDoc = item
		}
	}
	if goodDoc == nil || !goodDoc.Migrated || goodDoc.MigrationError != "" || goodDoc.FinalArtifact.GOOS != "windows" ||
		badDoc == nil || !badDoc.Migrated || badDoc.MigrationError == "" {
		t.Fatalf("per-profile backfill or migration diagnostic missing: good=%+v bad=%+v", goodDoc, badDoc)
	}
	getAgain, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name})
	if err != nil || getAgain.Revision != goodDoc.Revision || getAgain.YAML != goodDoc.YAML {
		t.Fatalf("backfill was not idempotent: %+v, %v", getAgain, err)
	}
	store, err := rpc.buildProfileStore()
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := store.Get(name)
	if err != nil || len(frozen.LegacyConfig) == 0 {
		t.Fatalf("legacy config not frozen: %+v, %v", frozen, err)
	}
	if _, err := rpc.ExecuteProfileBuild(t.Context(), &clientpb.BuildProfileExecuteRequest{Name: badName}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("untranslatable profile executed: %v", err)
	}
	if _, err := rpc.DeleteBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: badName}); err != nil {
		t.Fatalf("delete untranslatable legacy definition: %v", err)
	}
	if _, err := db.ImplantProfileByName(badName); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("legacy row survived definition delete: %v", err)
	}
	if _, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: badName}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted invalid profile was revived: %v", err)
	}
}

func TestBuildProfileLegacySnapshotStripsGeneratedKeys(t *testing.T) {
	config := &clientpb.ImplantConfig{ImplantBuilds: []*clientpb.ImplantBuild{{MtlsKey: "secret-key", PeerPrivateKey: "private-peer"}},
		Assets: []*commonpb.File{{Name: "asset", Data: []byte("contents")}}}
	encoded, err := legacyConfigSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("secret-key")) || bytes.Contains(encoded, []byte("private-peer")) {
		t.Fatal("generated private keys were persisted in profile snapshot")
	}
	var restored clientpb.ImplantConfig
	if err := proto.Unmarshal(encoded, &restored); err != nil || len(restored.ImplantBuilds) != 0 || len(restored.Assets) != 1 {
		t.Fatalf("sanitized profile config lost required assets: %+v, %v", restored, err)
	}
}

func TestBuildProfileRPCRejectsUnsafeLookupNames(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	for _, name := range []string{"", "../escape", "..\\escape", strings.Repeat("a", 256)} {
		if _, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("unsafe GetBuildProfile name %q accepted: %v", name, err)
		}
		if _, err := rpc.DeleteBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("unsafe DeleteBuildProfile name %q accepted: %v", name, err)
		}
		if _, err := rpc.ExecuteProfileBuild(t.Context(), &clientpb.BuildProfileExecuteRequest{Name: name}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("unsafe ExecuteProfileBuild name %q accepted: %v", name, err)
		}
	}
}

func TestBuildProfileRPCNameLengthBoundary(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	name := strings.Repeat("a", 255)
	profile := m6TestProfile(t, rpc, name, "windows", "amd64")
	valid, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(valid)}); err != nil {
		t.Fatalf("import maximum-length profile: %v", err)
	}
	if _, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); err != nil {
		t.Fatalf("get maximum-length profile: %v", err)
	}
	if _, err := rpc.DeleteBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: name}); err != nil {
		t.Fatalf("delete maximum-length profile: %v", err)
	}
	profile.Metadata.Name += "a"
	overlong, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(overlong)}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("overlong profile import returned %v", err)
	}
}

func TestBuildProfileAuthoringRejectsMismatchedDeclaredArtifact(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	profile := m6TestProfile(t, rpc, "m6-mismatch-"+models.NewUUID().String(), "windows", "amd64")
	profile.FinalOutput.Artifact.Container = "elf"
	yaml, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ValidateBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(yaml)}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("mismatched artifact was accepted: %v", err)
	}
}

func TestBuildProfileMigratedSnapshotRejectsExtraTransform(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	profile := m6TestProfile(t, rpc, "m6-snapshot-"+models.NewUUID().String(), "windows", "amd64")
	yaml, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &buildplan.ProfileSnapshot{YAML: yaml, Migrated: true}
	plan := buildplan.Plan{
		Steps:         []buildplan.Step{{ID: "core", Kind: "core", Spec: profile.Core}},
		FinalArtifact: profile.FinalOutput.Artifact.PlanArtifact(profile.Target),
	}
	if err := verifyMigratedProfileSnapshot(snapshot, plan); err != nil {
		t.Fatalf("unchanged migrated plan was rejected: %v", err)
	}
	plan.Steps = append(plan.Steps, buildplan.Step{ID: "extra", Kind: "transform",
		Spec: buildplan.Reference{Name: "builtin/xor", Version: "1.0.0", Digest: "sha256:extra"}})
	if err := verifyMigratedProfileSnapshot(snapshot, plan); err == nil {
		t.Fatal("extra transform was accepted against frozen migrated snapshot")
	}
}

func TestBuildProfileOldRPCRejectsTraversalWithoutRenaming(t *testing.T) {
	rpc := &Server{}
	_, err := rpc.SaveImplantProfile(context.Background(), &clientpb.ImplantProfile{Name: "../bad", Config: &clientpb.ImplantConfig{}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("legacy profile traversal was accepted: %v", err)
	}
	if _, err := rpc.SaveImplantProfile(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil legacy profile was accepted: %v", err)
	}
}
