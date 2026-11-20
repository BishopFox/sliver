package rpc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/db"
)

func TestImplantBuildsReportsRecordedCore(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	for index, name := range []string{"recorded-core", "unrecorded-legacy"} {
		config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{
			GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE,
		})
		if err != nil {
			t.Fatal(err)
		}
		build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: name, ImplantConfigID: config.ID})
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			if err := db.SaveResourceID(&clientpb.ResourceID{Name: name, Value: 1}); err != nil {
				t.Fatal(err)
			}
			artifact := buildplan.Artifact{Kind: "executable", Container: "elf",
				Target: buildplan.Target{OS: "linux", Arch: "amd64"}, GoBuildMode: "default"}
			plan := buildplan.Plan{Steps: []buildplan.Step{{
				ID: "core", Kind: "core", Spec: buildplan.Reference{
					Name: "example/custom", Version: "2.3.1", Digest: "exact-core-digest",
				}, Provider: "server", Output: artifact,
			}}, FinalArtifact: artifact}
			if _, err := buildcore.CompleteRecord(store, plan, buildcore.Result{
				BuildID: build.ID, FileName: name, Data: []byte("build bytes"), ToolchainIdentity: "test compiler",
			}); err != nil {
				t.Fatal(err)
			}
		}
	}

	builds, err := rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	ref := builds.GetCoreSpecs()["recorded-core"]
	if ref == nil || ref.Name != "example/custom" || ref.Version != "2.3.1" || ref.Digest != "exact-core-digest" {
		t.Fatalf("recorded core identity missing: %+v", ref)
	}
	if _, ok := builds.GetCoreSpecs()["unrecorded-legacy"]; ok {
		t.Fatal("historical build was assigned an unverified core")
	}
	if builds.GetResourceIDs()["unrecorded-legacy"] == nil {
		t.Fatal("build without a resource ID was omitted from implant listing")
	}
}

func TestImplantBuildsWithoutRecordStore(t *testing.T) {
	appDir := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", appDir)
	if err := os.WriteFile(filepath.Join(appDir, "build-records"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	name := "build-without-record-store"
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{
		GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: name, ImplantConfigID: config.ID}); err != nil {
		t.Fatal(err)
	}
	builds, err := (&Server{}).ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if builds.GetConfigs()[name] == nil {
		t.Fatal("build listing was lost when the record store could not open")
	}
	if builds.GetCoreSpecs()[name] != nil {
		t.Fatal("build without a readable record was assigned a core")
	}
}
