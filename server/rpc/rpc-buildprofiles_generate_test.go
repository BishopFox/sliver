//go:build sliver_generate

package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildprofile"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/generate"
)

// TestM6AuthoredProfileBuildFreezesSnapshot proves that imported YAML runs the
// installed Sliver compiler and keeps its resolved plan and bytes after a
// profile edit and definition deletion.
func TestM6AuthoredProfileBuildFreezesSnapshot(t *testing.T) {
	if !generate.SupportedCompilerTargets[runtime.GOOS+"/"+runtime.GOARCH] {
		t.Skip("native host is not a supported Sliver target")
	}
	setupGenerateSpoofTest(t)
	rpc := &Server{}
	profileName := "m6-native-" + models.NewUUID().String()
	buildName := "m6-build-" + models.NewUUID().String()
	profile := m6TestProfile(t, rpc, profileName, runtime.GOOS, runtime.GOARCH)
	yaml, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ImportBuildProfile(t.Context(), &clientpb.BuildProfileImportRequest{YAML: string(yaml)}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = rpc.DeleteBuildProfile(context.Background(), &clientpb.BuildProfileRef{Name: profileName})
		_ = RemoveBuildByName(buildName)
		_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), runtime.GOOS, runtime.GOARCH, buildName))
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	record, err := rpc.ExecuteProfileBuild(ctx, &clientpb.BuildProfileExecuteRequest{Name: profileName, BuildName: buildName})
	if err != nil {
		t.Fatal(err)
	}
	if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.FinalArtifact == nil || record.Plan == nil {
		t.Fatalf("profile build did not complete: %+v", record)
	}
	stored, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || stored.File == nil || len(stored.File.Data) == 0 {
		t.Fatalf("final artifact missing: %+v, %v", stored, err)
	}
	hash := sha256.Sum256(stored.File.Data)
	wantSHA := hex.EncodeToString(hash[:])
	if stored.Metadata.SHA256 != wantSHA || record.FinalArtifact.SHA256 != wantSHA {
		t.Fatalf("final artifact checksum mismatch: record=%s stored=%s computed=%s", record.FinalArtifact.SHA256, stored.Metadata.SHA256, wantSHA)
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := store.Get(record.ID)
	if err != nil || frozen.Plan.Request.ProfileSnapshot == nil || frozen.Plan.Request.ProfileSnapshot.Name != profileName ||
		!bytes.Equal(frozen.Plan.Request.ProfileSnapshot.YAML, yaml) || frozen.Plan.Request.ProfileSnapshot.Revision != 1 {
		t.Fatalf("profile snapshot missing from build record: %+v, %v", frozen.Plan.Request.ProfileSnapshot, err)
	}
	profile.Metadata.Revision = 2
	profile.C2.Endpoints[0].Address = "mtls://changed.example.invalid:8888"
	newYAML, err := buildprofile.CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ImportBuildProfile(ctx, &clientpb.BuildProfileImportRequest{YAML: string(newYAML)}); err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.DeleteBuildProfile(ctx, &clientpb.BuildProfileRef{Name: profileName}); err != nil {
		t.Fatal(err)
	}
	again, err := store.Get(record.ID)
	if err != nil || again.Plan.SHA256 != frozen.Plan.SHA256 ||
		!bytes.Equal(again.Plan.Request.ProfileSnapshot.YAML, yaml) {
		t.Fatalf("edit or deletion changed completed build plan: %+v, %v", again, err)
	}
	after, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || !bytes.Equal(after.File.Data, stored.File.Data) {
		t.Fatalf("edit or deletion changed completed artifact: %v", err)
	}
}

// TestM6MigratedProfileBuildPreservesLegacyConfig uses the compatibility path
// and checks settings not previously carried by the neutral request.
func TestM6MigratedProfileBuildPreservesLegacyConfig(t *testing.T) {
	if !generate.SupportedCompilerTargets[runtime.GOOS+"/"+runtime.GOARCH] {
		t.Skip("native host is not a supported Sliver target")
	}
	setupGenerateSpoofTest(t)
	rpc := &Server{}
	profileName := "m6-legacy-native-" + models.NewUUID().String()
	buildName := "m6-legacy-build-" + models.NewUUID().String()
	saved, err := generate.SaveImplantProfile(&clientpb.ImplantProfile{Name: profileName, Config: &clientpb.ImplantConfig{
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Format: clientpb.OutputFormat_EXECUTABLE,
		IsBeacon: true, TemplateName: "sliver", C2: []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
		ConnectionStrategy: "rd", ReconnectInterval: int64(17 * time.Second), PollTimeout: int64(91 * time.Second),
		MaxConnectionErrors: 23, BeaconInterval: int64(19 * time.Second), BeaconJitter: int64(3 * time.Second),
		HTTPC2ConfigName: "default", WGKeyExchangePort: 1337, WGTcpCommsPort: 1338, Debug: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = rpc.DeleteBuildProfile(context.Background(), &clientpb.BuildProfileRef{Name: profileName})
		_ = db.Session().Where("id = ?", saved.Config.ID).Delete(&models.ImplantConfig{}).Error
		_ = RemoveBuildByName(buildName)
		_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), runtime.GOOS, runtime.GOARCH, buildName))
	})
	document, err := rpc.GetBuildProfile(t.Context(), &clientpb.BuildProfileRef{Name: profileName})
	if err != nil || !document.Migrated || document.MigrationError != "" || document.FinalArtifact.GOOS != runtime.GOOS {
		t.Fatalf("legacy profile did not backfill: %+v, %v", document, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	record, err := rpc.ExecuteProfileBuild(ctx, &clientpb.BuildProfileExecuteRequest{Name: profileName, BuildName: buildName})
	if err != nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED {
		t.Fatalf("migrated build failed: %+v, %v", record, err)
	}
	build, err := db.ImplantBuildByName(buildName)
	if err != nil {
		t.Fatal(err)
	}
	config, err := db.ImplantConfigByID(build.ImplantConfigID)
	if err != nil {
		t.Fatal(err)
	}
	if !config.IsBeacon || config.BeaconInterval != int64(19*time.Second) || config.BeaconJitter != int64(3*time.Second) ||
		config.ConnectionStrategy != "rd" || config.ReconnectInterval != int64(17*time.Second) ||
		config.PollTimeout != int64(91*time.Second) || config.MaxConnectionErrors != 23 ||
		config.WGKeyExchangePort != 1337 || config.WGTcpCommsPort != 1338 {
		t.Fatalf("legacy config changed in profile execution: %+v", config)
	}
	stored, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || stored.File == nil || len(stored.File.Data) == 0 {
		t.Fatalf("migrated final bytes missing: %+v, %v", stored, err)
	}
	internal, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := internal.Get(record.ID)
	if err != nil || frozen.Plan.Request.ProfileSnapshot == nil || !frozen.Plan.Request.ProfileSnapshot.Migrated ||
		frozen.Plan.Request.ProfileSnapshot.LegacyConfigSHA256 == "" ||
		!bytes.Equal(frozen.Plan.Request.ProfileSnapshot.YAML, []byte(document.YAML)) {
		t.Fatalf("legacy profile snapshot missing: %+v, %v", frozen.Plan.Request.ProfileSnapshot, err)
	}
}
