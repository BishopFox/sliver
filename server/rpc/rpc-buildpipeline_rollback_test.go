package rpc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
)

type rollbackCoreAdapter struct {
	name    string
	appDir  string
	savedID *string
}

func (rollbackCoreAdapter) ExecutorID() string { return "test-rollback-core" }

func (rollbackCoreAdapter) Supports(buildplan.Target, buildplan.Toolchain) bool { return true }

func (rollbackCoreAdapter) Build(context.Context, buildplan.Plan) (buildcore.Result, error) {
	return buildcore.Result{}, errors.New("step adapter should be used")
}

func (adapter rollbackCoreAdapter) BuildCore(context.Context, buildplan.Plan) (buildcore.CoreProduct, error) {
	data := testCorePE()
	return buildcore.CoreProduct{
		Result: buildcore.Result{ImplantName: adapter.name, FileName: adapter.name + ".exe", Data: data},
		SaveFinal: func(_ context.Context, _ buildplan.Artifact, _ string, final []byte) (string, error) {
			path := filepath.Join(adapter.appDir, "rollback-final.exe")
			if err := os.WriteFile(path, final, 0600); err != nil {
				return "", err
			}
			build := &clientpb.ImplantBuild{Name: adapter.name}
			config := &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE}
			if err := generate.ImplantBuildSave(build, config, path); err != nil {
				return "", err
			}
			*adapter.savedID = build.ID
			return build.ID, nil
		},
	}, nil
}

func TestResolvedPipelineRollsBackRuntimeBuildWhenRecordStoreFails(t *testing.T) {
	appDir := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", appDir)
	// The final runtime save succeeds, then the typed record store cannot open.
	if err := os.WriteFile(filepath.Join(appDir, "build-records"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	name := "rollback-runtime-build"
	artifact := buildplan.Artifact{Kind: "executable", Container: "pe", Target: buildplan.Target{OS: "windows", Arch: "amd64"}, GoBuildMode: "default"}
	plan := buildplan.Plan{
		Request:       buildplan.Request{Name: name, Target: artifact.Target, Mode: "session", FinalFormat: "exe"},
		Steps:         []buildplan.Step{{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "test/rollback", Version: "1.0.0"}, Provider: "server", Output: artifact}},
		FinalArtifact: artifact,
	}
	savedID := ""
	_, _, err := (&Server{}).executeResolvedPlan(t.Context(), plan,
		rollbackCoreAdapter{name: name, appDir: appDir, savedID: &savedID}, nil, nil)
	if err == nil || savedID == "" {
		t.Fatalf("expected record store failure after runtime save: buildID=%q err=%v", savedID, err)
	}
	build, lookupErr := db.ImplantBuildByID(savedID)
	if lookupErr == nil && build != nil && build.ID == savedID {
		t.Fatalf("saved runtime build %s still authenticates after record failure", savedID)
	}
	if resource, resourceErr := db.ResourceIDByName(name); resourceErr == nil && resource != nil && resource.Name == name {
		t.Fatalf("saved runtime build %s retained ResourceID %d", savedID, resource.Value)
	}
	if _, fileErr := os.Stat(filepath.Join(appDir, "builds", savedID)); !errors.Is(fileErr, os.ErrNotExist) {
		t.Fatalf("saved runtime build %s retained final file: %v", savedID, fileErr)
	}
}
