//go:build server

package generate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

func TestProfileMutationDoesNotChangeStoredBuildSnapshotPolicy(t *testing.T) {
	suffix := time.Now().UnixNano()
	profileName := fmt.Sprintf("snapshot-profile-%d", suffix)
	buildName := fmt.Sprintf("snapshot-build-%d", suffix)

	profile, err := SaveImplantProfile(&clientpb.ImplantProfile{
		Name: profileName,
		Config: &clientpb.ImplantConfig{
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
			TemplateName:     SliverTemplateName,
		},
	})
	if err != nil {
		t.Fatalf("SaveImplantProfile() error = %v", err)
	}

	var snapshotID string
	var buildID string
	t.Cleanup(func() {
		if buildID != "" {
			db.Session().Where("id = ?", uuid.FromStringOrNil(buildID)).Delete(&models.ImplantBuild{})
		}
		if snapshotID != "" {
			configID := uuid.FromStringOrNil(snapshotID)
			db.Session().Where("implant_config_id = ?", configID).Delete(&models.ImplantC2{})
			db.Session().Where("implant_config_id = ?", configID).Delete(&models.EncoderAsset{})
			db.Session().Where("implant_config_id = ?", configID).Delete(&models.CanaryDomain{})
			db.Session().Where("id = ?", configID).Delete(&models.ImplantConfig{})
		}
		_ = db.DeleteProfile(profileName)
	})

	snapshot, sourceProfileID, err := NewImplantConfigSnapshot(profile.Config)
	if err != nil {
		t.Fatalf("NewImplantConfigSnapshot() error = %v", err)
	}
	snapshotID = snapshot.ID
	if _, err := db.CreateImplantConfigWithID(snapshot); err != nil {
		t.Fatalf("CreateImplantConfigWithID() error = %v", err)
	}

	build, err := db.SaveImplantBuildWithSourceProfile(&clientpb.ImplantBuild{
		Name:            buildName,
		ImplantConfigID: snapshot.ID,
	}, sourceProfileID)
	if err != nil {
		t.Fatalf("SaveImplantBuildWithSourceProfile() error = %v", err)
	}
	buildID = build.ID

	updatedProfile := proto.Clone(profile).(*clientpb.ImplantProfile)
	updatedProfile.Config.ControlFlow = clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED
	if _, err := SaveImplantProfile(updatedProfile); err != nil {
		t.Fatalf("update profile policy: %v", err)
	}

	persistedBuild, err := db.ImplantBuildByName(buildName)
	if err != nil {
		t.Fatalf("ImplantBuildByName() error = %v", err)
	}
	persistedConfig, err := db.ImplantConfigByID(persistedBuild.ImplantConfigID)
	if err != nil {
		t.Fatalf("ImplantConfigByID() error = %v", err)
	}
	if persistedConfig.ID != snapshot.ID {
		t.Fatalf("build config ID = %q, want snapshot %q", persistedConfig.ID, snapshot.ID)
	}
	if persistedConfig.ImplantProfileID != "" {
		t.Fatalf("build snapshot profile ID = %q, want detached config", persistedConfig.ImplantProfileID)
	}
	if persistedConfig.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatalf("stored build policy = %v after profile mutation, want balanced-v1", persistedConfig.ControlFlow)
	}

	if _, err := db.SaveImplantBuild(persistedBuild); err != nil {
		t.Fatalf("SaveImplantBuild() update error = %v", err)
	}
	storedBuild := &models.ImplantBuild{}
	if err := db.Session().Where("id = ?", uuid.FromStringOrNil(build.ID)).First(storedBuild).Error; err != nil {
		t.Fatalf("load build model: %v", err)
	}
	if storedBuild.SourceImplantProfileID == nil || storedBuild.SourceImplantProfileID.String() != profile.ID {
		t.Fatalf("source profile ID after build update = %v, want %q", storedBuild.SourceImplantProfileID, profile.ID)
	}

	listedProfile, err := db.ProfileByName(profileName)
	if err != nil {
		t.Fatalf("ProfileByName() error = %v", err)
	}
	if len(listedProfile.Config.ImplantBuilds) != 1 || listedProfile.Config.ImplantBuilds[0].Name != buildName {
		t.Fatalf("profile builds = %v, want detached snapshot build %q", listedProfile.Config.ImplantBuilds, buildName)
	}

	if err := db.Session().Where("id = ?", uuid.FromStringOrNil(build.ID)).Delete(&models.ImplantBuild{}).Error; err != nil {
		t.Fatalf("delete build record: %v", err)
	}
	buildID = ""
	if err := db.DeleteProfile(profileName); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	auditConfig, err := db.ImplantConfigByID(snapshot.ID)
	if err != nil {
		t.Fatalf("load origin-policy audit config after build/profile deletion: %v", err)
	}
	if auditConfig.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatalf("retained audit policy = %v, want balanced-v1", auditConfig.ControlFlow)
	}
}

func TestDeleteFailedImplantConfigSnapshotOnlyDeletesUnreferencedSnapshot(t *testing.T) {
	newSnapshot := func(t *testing.T) *clientpb.ImplantConfig {
		t.Helper()
		config, err := db.CreateImplantConfigWithID(&clientpb.ImplantConfig{
			ID: uuid.Must(uuid.NewV4()).String(),
			C2: []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
		})
		if err != nil {
			t.Fatalf("CreateImplantConfigWithID() error = %v", err)
		}
		return config
	}
	deleteConfig := func(configID string) {
		id := uuid.FromStringOrNil(configID)
		db.Session().Where("implant_config_id = ?", id).Delete(&models.ImplantC2{})
		db.Session().Where("id = ?", id).Delete(&models.ImplantConfig{})
	}

	unreferenced := newSnapshot(t)
	t.Cleanup(func() { deleteConfig(unreferenced.ID) })
	if err := db.DeleteFailedImplantConfigSnapshot(unreferenced.ID); err != nil {
		t.Fatalf("DeleteFailedImplantConfigSnapshot() error = %v", err)
	}
	if _, err := db.ImplantConfigByID(unreferenced.ID); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("unreferenced snapshot lookup error = %v, want record not found", err)
	}
	var c2Count int64
	if err := db.Session().Model(&models.ImplantC2{}).Where("implant_config_id = ?", uuid.FromStringOrNil(unreferenced.ID)).Count(&c2Count).Error; err != nil {
		t.Fatalf("count rolled-back C2 rows: %v", err)
	}
	if c2Count != 0 {
		t.Fatalf("rolled-back snapshot C2 rows = %d, want 0", c2Count)
	}

	referenced := newSnapshot(t)
	buildName := fmt.Sprintf("snapshot-rollback-guard-%d", time.Now().UnixNano())
	build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{
		Name:            buildName,
		ImplantConfigID: referenced.ID,
	})
	if err != nil {
		t.Fatalf("SaveImplantBuild() error = %v", err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", uuid.FromStringOrNil(build.ID)).Delete(&models.ImplantBuild{})
		deleteConfig(referenced.ID)
	})
	if err := db.DeleteFailedImplantConfigSnapshot(referenced.ID); err != nil {
		t.Fatalf("guard referenced snapshot: %v", err)
	}
	if _, err := db.ImplantConfigByID(referenced.ID); err != nil {
		t.Fatalf("referenced snapshot was deleted: %v", err)
	}
}
