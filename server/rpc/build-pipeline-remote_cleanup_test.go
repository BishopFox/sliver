package rpc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

//nolint:gocyclo // The test compares build records and cleanup state through one injected failure.
func TestRemotePreparedSliverBuildDiscardedOnFailure(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: "failed-remote-sliver", ImplantConfigID: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	run := pipelineRun{err: errors.New("assigned worker failed")}
	discardFailedPreparedRemoteSliver(&run, prepared)
	if run.err == nil || run.err.Error() != "assigned worker failed" {
		t.Fatalf("cleanup changed the original failure: %v", run.err)
	}
	if build, err := db.ImplantBuildByID(prepared.ID); err == nil && build.ID == prepared.ID {
		t.Fatal("failed remote Sliver build retained its authentication row")
	}

	// The final save can create a ResourceID and file before returning an
	// error. Cleanup removes these exact build-owned resources too.
	partial, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: "partial-remote-sliver", ImplantConfigID: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	partial.ImplantID = 0x7fac3101
	if err := db.SaveResourceID(&clientpb.ResourceID{Name: partial.Name, Value: partial.ImplantID}); err != nil {
		t.Fatal(err)
	}
	buildsDir := filepath.Join(assets.GetRootAppDir(), "builds")
	if err := os.MkdirAll(buildsDir, 0700); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildsDir, partial.ID)
	if err := os.WriteFile(buildPath, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	partialRun := pipelineRun{err: errors.New("final save failed")}
	discardFailedPreparedRemoteSliver(&partialRun, partial)
	if partialRun.err == nil || partialRun.err.Error() != "final save failed" {
		t.Fatalf("partial cleanup changed the original failure: %v", partialRun.err)
	}
	if build, err := db.ImplantBuildByID(partial.ID); err == nil && build.ID == partial.ID {
		t.Fatal("partial remote save retained its authentication row")
	}
	if resource, err := db.ResourceIDByValue(partial.ImplantID); err == nil && resource != nil && resource.Value == partial.ImplantID {
		t.Fatal("partial remote save retained its ResourceID")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err := os.Stat(buildPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial remote save retained its build file: %v", err)
	}
}

func TestRemotePreparedSliverSuccessfulBuildIsPreserved(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: "successful-remote-sliver", ImplantConfigID: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	run := pipelineRun{savedFinal: true}
	discardFailedPreparedRemoteSliver(&run, prepared)
	if run.err != nil {
		t.Fatal(run.err)
	}
	if build, err := db.ImplantBuildByID(prepared.ID); err != nil || build.ID != prepared.ID {
		t.Fatalf("successful remote build was discarded: %+v, %v", build, err)
	}
	other := proto.Clone(prepared).(*clientpb.ImplantBuild)
	other.Name = "different-build"
	if err := discardRuntimeBuildExact(other); err == nil {
		t.Fatal("cleanup accepted another build identity")
	}
	if build, err := db.ImplantBuildByID(prepared.ID); err != nil || build.ID != prepared.ID {
		t.Fatalf("identity mismatch removed a successful build: %+v, %v", build, err)
	}
}
