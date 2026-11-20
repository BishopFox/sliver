package rpc

import (
	"testing"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func externalBuildForCleanupTest(t *testing.T, completed bool) (*clientpb.ImplantBuild, string, string) {
	t.Helper()
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	suffix := models.NewUUID().String()
	builderName := "external-cleanup-builder-" + suffix
	operatorName := "external-cleanup-operator-" + suffix
	if err := core.AddBuilder(&clientpb.Builder{Name: builderName, OperatorName: operatorName}); err != nil {
		t.Fatal(err)
	}
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	build := &clientpb.ImplantBuild{Name: "external-cleanup-build-" + suffix, ImplantConfigID: config.ID}
	if completed {
		build.SHA256 = "completed-checksum"
	}
	build, err = db.SaveImplantBuild(build)
	if err != nil {
		t.Fatal(err)
	}
	core.TrackExternalBuildAssignment(build.ID, builderName, operatorName)
	t.Cleanup(func() {
		core.RemoveExternalBuildAssignment(build.ID)
		core.RemoveBuilder(builderName)
		_ = db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{}).Error
		_ = db.Session().Where("id = ?", config.ID).Delete(&models.ImplantConfig{}).Error
	})
	return build, builderName, operatorName
}

func TestExternalBuildFailureCleanup(t *testing.T) {
	build, _, operator := externalBuildForCleanupTest(t, false)
	_, err := (&Server{}).BuilderTrigger(contextWithCommonName(operator), &clientpb.Event{
		EventType: consts.ExternalBuildFailedEvent, Data: []byte(build.ID + ":compile failed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved, err := db.ImplantBuildByID(build.ID); err == nil && saved != nil && saved.ID == build.ID {
		t.Fatal("failed external build retained its authentication row")
	}
	if core.GetExternalBuildAssignment(build.ID) != nil {
		t.Fatal("failed external build retained its assignment")
	}
}

func TestExternalBuildSaveErrorCleanup(t *testing.T) {
	build, builderName, operator := externalBuildForCleanupTest(t, false)
	_, err := (&Server{}).GenerateExternalSaveBuild(contextWithCommonName(operator), &clientpb.ExternalImplantBinary{
		Name: builderName, ImplantBuildID: build.ID,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing upload error = %v", err)
	}
	if saved, err := db.ImplantBuildByID(build.ID); err == nil && saved != nil && saved.ID == build.ID {
		t.Fatal("failed external upload retained its authentication row")
	}
	if core.GetExternalBuildAssignment(build.ID) != nil {
		t.Fatal("failed external upload retained its assignment")
	}
}

func TestExternalBuildLateFailurePreservesCompleted(t *testing.T) {
	build, _, operator := externalBuildForCleanupTest(t, true)
	_, err := (&Server{}).BuilderTrigger(contextWithCommonName(operator), &clientpb.Event{
		EventType: consts.ExternalBuildFailedEvent, Data: []byte(build.ID + ":upload response lost"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved, err := db.ImplantBuildByID(build.ID); err != nil || saved == nil || saved.ID != build.ID {
		t.Fatalf("late failure removed completed build: %+v, %v", saved, err)
	}
}

func TestExternalBuildUnauthorizedFailurePreservesPending(t *testing.T) {
	build, _, _ := externalBuildForCleanupTest(t, false)
	_, err := (&Server{}).BuilderTrigger(contextWithCommonName("other-operator"), &clientpb.Event{
		EventType: consts.ExternalBuildFailedEvent, Data: []byte(build.ID + ":forged failure"),
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("foreign failure error = %v", err)
	}
	if saved, err := db.ImplantBuildByID(build.ID); err != nil || saved == nil || saved.ID != build.ID {
		t.Fatalf("foreign failure removed assigned build: %+v, %v", saved, err)
	}
}

func TestExternalBuildDisconnectCleanup(t *testing.T) {
	build, builderName, _ := externalBuildForCleanupTest(t, false)
	(&Server{}).discardDisconnectedExternalBuilder(core.GetBuilder(builderName))
	if saved, err := db.ImplantBuildByID(build.ID); err == nil && saved != nil && saved.ID == build.ID {
		t.Fatal("disconnected external builder retained pending authentication")
	}
	if core.GetExternalBuildAssignment(build.ID) != nil || core.GetBuilder(builderName) != nil {
		t.Fatal("disconnected external builder retained registry state")
	}
}

func TestExternalBuildDisconnectPreservesCompleted(t *testing.T) {
	build, builderName, _ := externalBuildForCleanupTest(t, true)
	(&Server{}).discardDisconnectedExternalBuilder(core.GetBuilder(builderName))
	if saved, err := db.ImplantBuildByID(build.ID); err != nil || saved == nil || saved.ID != build.ID {
		t.Fatalf("disconnect removed completed external build: %+v, %v", saved, err)
	}
	if core.GetExternalBuildAssignment(build.ID) != nil || core.GetBuilder(builderName) != nil {
		t.Fatal("disconnected external builder retained registry state")
	}
}
