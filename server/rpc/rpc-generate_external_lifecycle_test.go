package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExternalBuildFailedDeletesPendingBuildAndDetachedSnapshot(t *testing.T) {
	fixture := newPendingExternalBuildFixture(t, "external-failed")

	builds, err := db.ImplantBuilds()
	if err != nil {
		t.Fatalf("ImplantBuilds() with a pending build: %v", err)
	}
	if builds.Configs[fixture.name] != nil {
		t.Fatalf("ImplantBuilds() exposed pending build %q as completed", fixture.name)
	}
	if builds.ResourceIDs[fixture.name] != nil {
		t.Fatalf("pending build %q unexpectedly has a resource ID", fixture.name)
	}

	const operatorName = "external-failed-operator"
	core.TrackExternalBuildAssignment(fixture.buildID, "external-failed-builder", operatorName)
	t.Cleanup(func() { core.RemoveExternalBuildAssignment(fixture.buildID) })

	_, err = (&Server{}).BuilderTrigger(contextWithCommonName(operatorName), &clientpb.Event{
		EventType: consts.ExternalBuildFailedEvent,
		Data:      []byte(fixture.buildID + ":compiler failed"),
	})
	if err != nil {
		t.Fatalf("BuilderTrigger(ExternalBuildFailedEvent) error = %v", err)
	}
	if assignment := core.GetExternalBuildAssignment(fixture.buildID); assignment != nil {
		t.Fatalf("external build assignment still exists after failure: %+v", assignment)
	}

	assertExternalBuildFixtureCounts(t, fixture, 0, 0, 0)
}

func TestExternalBuildFailedProtectsCompletedBuild(t *testing.T) {
	fixture := newPendingExternalBuildFixture(t, "external-completed")
	resourceID := uint64(time.Now().UnixNano())
	if err := db.SaveResourceID(&clientpb.ResourceID{Name: fixture.name, Value: resourceID}); err != nil {
		t.Fatalf("SaveResourceID() error = %v", err)
	}
	build, err := db.ImplantBuildByName(fixture.name)
	if err != nil {
		t.Fatalf("ImplantBuildByName() error = %v", err)
	}
	build.ImplantID = resourceID
	build.MD5 = "completed-md5"
	build.SHA1 = "completed-sha1"
	build.SHA256 = "completed-sha256"
	if _, err := db.SaveImplantBuild(build); err != nil {
		t.Fatalf("SaveImplantBuild() completed update error = %v", err)
	}

	const operatorName = "external-completed-operator"
	core.TrackExternalBuildAssignment(fixture.buildID, "external-completed-builder", operatorName)
	t.Cleanup(func() { core.RemoveExternalBuildAssignment(fixture.buildID) })

	_, err = (&Server{}).BuilderTrigger(contextWithCommonName(operatorName), &clientpb.Event{
		EventType: consts.ExternalBuildFailedEvent,
		Data:      []byte(fixture.buildID + ":late failure"),
	})
	if err != nil {
		t.Fatalf("BuilderTrigger(ExternalBuildFailedEvent) error = %v", err)
	}
	if assignment := core.GetExternalBuildAssignment(fixture.buildID); assignment != nil {
		t.Fatalf("external build assignment still exists after terminal failure: %+v", assignment)
	}

	assertExternalBuildFixtureCounts(t, fixture, 1, 1, 1)
}

func TestRemoveBuildByNameDeletesPendingBuildAndDetachedSnapshot(t *testing.T) {
	fixture := newPendingExternalBuildFixture(t, "external-remove-pending")
	core.TrackExternalBuildAssignment(fixture.buildID, "external-remove-builder", "external-remove-operator")
	t.Cleanup(func() { core.RemoveExternalBuildAssignment(fixture.buildID) })
	if err := RemoveBuildByName(fixture.name); err != nil {
		t.Fatalf("RemoveBuildByName() pending build error = %v", err)
	}
	if assignment := core.GetExternalBuildAssignment(fixture.buildID); assignment != nil {
		t.Fatalf("external build assignment still exists after removal: %+v", assignment)
	}
	assertExternalBuildFixtureCounts(t, fixture, 0, 0, 0)
}

func TestGenerateExternalRejectsUnpersistedProfileAssociation(t *testing.T) {
	builderName := fmt.Sprintf("external-profile-trust-builder-%d", time.Now().UnixNano())
	if err := core.AddBuilder(&clientpb.Builder{Name: builderName, OperatorName: "external-profile-trust-operator"}); err != nil {
		t.Fatalf("AddBuilder() error = %v", err)
	}
	t.Cleanup(func() { core.RemoveBuilder(builderName) })

	_, err := (&Server{}).GenerateExternal(context.Background(), &clientpb.ExternalGenerateReq{
		Name:        fmt.Sprintf("external-profile-trust-%d", time.Now().UnixNano()),
		BuilderName: builderName,
		Config: &clientpb.ImplantConfig{
			ImplantProfileID: uuid.Must(uuid.NewV4()).String(),
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GenerateExternal() error = %v, code = %s; want %s", err, status.Code(err), codes.InvalidArgument)
	}
}

func TestResolveExternalBuildConfigUsesPersistedProfileAssociation(t *testing.T) {
	profileName := fmt.Sprintf("external-profile-source-%d", time.Now().UnixNano())
	profile, err := generate.SaveImplantProfile(&clientpb.ImplantProfile{
		Name: profileName,
		Config: &clientpb.ImplantConfig{
			GOOS:         "linux",
			GOARCH:       "amd64",
			TemplateName: generate.SliverTemplateName,
		},
	})
	if err != nil {
		t.Fatalf("SaveImplantProfile() error = %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteProfile(profileName) })

	resolved, err := resolveExternalBuildConfig(&clientpb.ImplantConfig{
		ID:               profile.Config.ID,
		ImplantProfileID: uuid.Must(uuid.NewV4()).String(),
		GOOS:             "forged",
	})
	if err != nil {
		t.Fatalf("resolveExternalBuildConfig() error = %v", err)
	}
	if resolved.GOOS != "linux" {
		t.Fatalf("resolved GOOS = %q, want persisted value %q", resolved.GOOS, "linux")
	}
	if resolved.ImplantProfileID != profile.ID {
		t.Fatalf("resolved profile ID = %q, want persisted association %q", resolved.ImplantProfileID, profile.ID)
	}
}

type pendingExternalBuildFixture struct {
	name     string
	buildID  string
	configID uuid.UUID
}

func newPendingExternalBuildFixture(t *testing.T, prefix string) pendingExternalBuildFixture {
	t.Helper()
	configID := uuid.Must(uuid.NewV4())
	name := fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	config, err := db.CreateImplantConfigWithID(&clientpb.ImplantConfig{
		ID:            configID.String(),
		GOOS:          "linux",
		GOARCH:        "amd64",
		TemplateName:  generate.SliverTemplateName,
		C2:            []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
		CanaryDomains: []string{"pending.example"},
		Assets:        []*commonpb.File{{Name: "pending-asset"}},
	})
	if err != nil {
		t.Fatalf("CreateImplantConfigWithID() error = %v", err)
	}
	build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{
		Name:            name,
		ImplantConfigID: config.ID,
	})
	if err != nil {
		t.Fatalf("SaveImplantBuild() pending build error = %v", err)
	}

	fixture := pendingExternalBuildFixture{name: name, buildID: build.ID, configID: configID}
	t.Cleanup(func() { cleanupExternalBuildFixture(fixture) })
	return fixture
}

func cleanupExternalBuildFixture(fixture pendingExternalBuildFixture) {
	buildID := uuid.FromStringOrNil(fixture.buildID)
	db.Session().Where("id = ?", buildID).Delete(&models.ImplantBuild{})
	db.Session().Where("name = ?", fixture.name).Delete(&models.ResourceID{})
	db.Session().Where("implant_config_id = ?", fixture.configID).Delete(&models.ImplantC2{})
	db.Session().Where("implant_config_id = ?", fixture.configID).Delete(&models.EncoderAsset{})
	db.Session().Where("implant_config_id = ?", fixture.configID).Delete(&models.CanaryDomain{})
	db.Session().Where("id = ?", fixture.configID).Delete(&models.ImplantConfig{})
}

func assertExternalBuildFixtureCounts(t *testing.T, fixture pendingExternalBuildFixture, wantBuilds int64, wantConfigs int64, wantResources int64) {
	t.Helper()
	var buildCount int64
	if err := db.Session().Model(&models.ImplantBuild{}).Where("id = ?", uuid.FromStringOrNil(fixture.buildID)).Count(&buildCount).Error; err != nil {
		t.Fatalf("count implant builds: %v", err)
	}
	var configCount int64
	if err := db.Session().Model(&models.ImplantConfig{}).Where("id = ?", fixture.configID).Count(&configCount).Error; err != nil {
		t.Fatalf("count implant configs: %v", err)
	}
	var resourceCount int64
	if err := db.Session().Model(&models.ResourceID{}).Where("name = ?", fixture.name).Count(&resourceCount).Error; err != nil {
		t.Fatalf("count resource IDs: %v", err)
	}
	if buildCount != wantBuilds || configCount != wantConfigs || resourceCount != wantResources {
		t.Fatalf(
			"fixture counts = builds:%d configs:%d resources:%d, want builds:%d configs:%d resources:%d",
			buildCount, configCount, resourceCount, wantBuilds, wantConfigs, wantResources,
		)
	}

	if wantConfigs == 0 {
		for name, model := range map[string]interface{}{
			"C2":       &models.ImplantC2{},
			"assets":   &models.EncoderAsset{},
			"canaries": &models.CanaryDomain{},
		} {
			var count int64
			if err := db.Session().Model(model).Where("implant_config_id = ?", fixture.configID).Count(&count).Error; err != nil {
				t.Fatalf("count %s associations: %v", name, err)
			}
			if count != 0 {
				t.Fatalf("%s association count = %d, want 0", name, count)
			}
		}
	}
}
