package generate

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

func TestImplantBuildSavePublicationFailureRevokesBuild(t *testing.T) {
	for _, preSaved := range []bool{false, true} {
		name := "new build"
		if preSaved {
			name = "pre-saved build"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("SLIVER_ROOT_DIR", root)
			artifact := filepath.Join(root, "artifact")
			if err := os.WriteFile(artifact, []byte("test binary"), 0600); err != nil {
				t.Fatal(err)
			}
			identity := "save-failure-" + models.NewUUID().String()
			endpoint := "mtls://" + identity + ".invalid:8888"
			config := &clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity,
				C2: []*clientpb.ImplantC2{{URL: endpoint, Priority: 1}}}
			build := &clientpb.ImplantBuild{Name: identity, PeerPublicKeyDigest: identity}
			if preSaved {
				var err error
				config, err = db.SaveImplantConfig(config)
				if err != nil {
					t.Fatal(err)
				}
				build.ImplantConfigID = config.ID
				build, err = db.SaveImplantBuild(build)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					db.Session().Where("id = ?", config.ID).Delete(&models.ImplantConfig{})
				})
			}
			publishFailure := errors.New("injected file publication failure")
			called := false
			err := implantBuildSave(build, config, artifact, func(staged, final string) error {
				called = true
				if info, statErr := os.Stat(staged); statErr != nil || !info.Mode().IsRegular() {
					return errors.New("staged file is absent")
				}
				if filepath.Dir(final) != filepath.Join(root, "builds") {
					return errors.New("unexpected final build directory")
				}
				return publishFailure
			})
			if !called || !errors.Is(err, publishFailure) {
				t.Fatalf("publish failure = %v, called=%v", err, called)
			}
			var builds, resources, configs, c2s int64
			if err := db.Session().Model(&models.ImplantBuild{}).Where("name = ?", identity).Count(&builds).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Session().Model(&models.ResourceID{}).Where("name = ?", identity).Count(&resources).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Session().Model(&models.ImplantConfig{}).Where("limit_hostname = ?", identity).Count(&configs).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Session().Model(&models.ImplantC2{}).Where("url = ?", endpoint).Count(&c2s).Error; err != nil {
				t.Fatal(err)
			}
			wantConfigs := int64(0)
			if preSaved {
				wantConfigs = 1 // A pre-existing config may be shared by another build.
			}
			if builds != 0 || resources != 0 || configs != wantConfigs || c2s != wantConfigs {
				t.Fatalf("unfinished save persisted builds=%d resources=%d configs=%d C2=%d", builds, resources, configs, c2s)
			}
			entries, err := os.ReadDir(filepath.Join(root, "builds"))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("unfinished save left %d build files", len(entries))
			}
		})
	}
}

func TestImplantBuildSavePreSavedBuildPublishesFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	artifact := filepath.Join(root, "artifact")
	data := []byte("test binary")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	identity := "save-success-" + models.NewUUID().String()
	endpoint := "mtls://" + identity + ".invalid:8888"
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity,
		C2: []*clientpb.ImplantC2{{URL: endpoint, Priority: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: identity, ImplantConfigID: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{})
		db.Session().Where("id = ?", config.ID).Delete(&models.ImplantConfig{})
		db.Session().Where("name = ?", identity).Delete(&models.ResourceID{})
	})
	if err := ImplantBuildSave(build, config, artifact); err != nil {
		t.Fatalf("save pre-saved build: %v", err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "builds", build.ID))
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("stored file differs: %v", err)
	}
	if build.ImplantID == 0 || build.SHA256 == "" {
		t.Fatalf("saved build missing ID or hash: %+v", build)
	}
	var c2s int64
	if err := db.Session().Model(&models.ImplantC2{}).Where("url = ? AND implant_config_id = ?", endpoint, config.ID).Count(&c2s).Error; err != nil {
		t.Fatal(err)
	}
	if c2s != 1 {
		t.Fatalf("pre-saved config C2 rows = %d, want 1", c2s)
	}
}

func TestImplantBuildSaveDoesNotReplaceConcurrentFinalFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	artifact := filepath.Join(root, "artifact")
	if err := os.WriteFile(artifact, []byte("losing binary"), 0600); err != nil {
		t.Fatal(err)
	}
	identity := "save-race-" + models.NewUUID().String()
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity})
	if err != nil {
		t.Fatal(err)
	}
	build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: identity, ImplantConfigID: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{})
		db.Session().Where("id = ?", config.ID).Delete(&models.ImplantConfig{})
	})
	winner := []byte("winning binary")
	err = implantBuildSave(build, config, artifact, func(staged, final string) error {
		if err := os.WriteFile(final, winner, 0600); err != nil {
			return err
		}
		return publishNewImplantFile(staged, final)
	})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("competing publication error = %v, want file exists", err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "builds", build.ID))
	if err != nil || !bytes.Equal(stored, winner) {
		t.Fatalf("competing final file was replaced: %v", err)
	}
	var persisted models.ImplantBuild
	if err := db.Session().Where("id = ?", build.ID).First(&persisted).Error; err != nil || persisted.SHA256 != "" {
		t.Fatalf("losing transaction persisted build hash=%q, err=%v", persisted.SHA256, err)
	}
}

func TestImplantBuildSaveRejectsCompletedBuildWithoutFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	artifact := filepath.Join(root, "artifact")
	if err := os.WriteFile(artifact, []byte("new binary"), 0600); err != nil {
		t.Fatal(err)
	}
	identity := "save-completed-" + models.NewUUID().String()
	config, err := db.SaveImplantConfig(&clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity})
	if err != nil {
		t.Fatal(err)
	}
	build, err := db.SaveImplantBuild(&clientpb.ImplantBuild{Name: identity, ImplantConfigID: config.ID, SHA256: "previous-hash"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{})
		db.Session().Where("id = ?", config.ID).Delete(&models.ImplantConfig{})
	})
	called := false
	if err := implantBuildSave(build, config, artifact, func(string, string) error { called = true; return nil }); err == nil || called {
		t.Fatalf("completed build save = %v, publish called=%v", err, called)
	}
	var stored models.ImplantBuild
	if err := db.Session().Where("id = ?", build.ID).First(&stored).Error; err != nil {
		t.Fatalf("completed build was revoked: %v", err)
	}
	if stored.SHA256 != "previous-hash" {
		t.Fatalf("completed build changed on rollback: SHA256=%q", stored.SHA256)
	}
}

func TestImplantBuildSaveStagePublicationFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	identity := "stage-failure-" + models.NewUUID().String()
	build := &clientpb.ImplantBuild{Name: identity, Stage: true}
	config := &clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity}
	failure := errors.New("stage publication failed")
	called := false
	err := saveStage(build, config, []byte("stage bytes"), "stage-type", func(staged, final string) error {
		called = true
		if _, statErr := os.Stat(staged); statErr != nil {
			return statErr
		}
		return failure
	})
	if !called || !errors.Is(err, failure) {
		t.Fatalf("stage publication = %v, called=%v", err, called)
	}
	var builds, resources, configs int64
	if err := db.Session().Model(&models.ImplantBuild{}).Where("name = ?", identity).Count(&builds).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Session().Model(&models.ResourceID{}).Where("name = ?", identity).Count(&resources).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Session().Model(&models.ImplantConfig{}).Where("limit_hostname = ?", identity).Count(&configs).Error; err != nil {
		t.Fatal(err)
	}
	if builds != 0 || resources != 0 || configs != 0 {
		t.Fatalf("failed stage persisted builds=%d resources=%d configs=%d", builds, resources, configs)
	}
	entries, err := os.ReadDir(filepath.Join(root, "builds"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed stage left files: %v, entries=%d", err, len(entries))
	}
}

func TestImplantBuildSaveStageTypeAndFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	identity := "stage-success-" + models.NewUUID().String()
	build := &clientpb.ImplantBuild{Name: identity, Stage: true}
	config := &clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity}
	data := []byte("stage bytes")
	if err := SaveStage(build, config, data, "stage-type"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{})
		db.Session().Where("id = ?", build.ImplantConfigID).Delete(&models.ImplantConfig{})
		db.Session().Where("name = ?", identity).Delete(&models.ResourceID{})
	})
	resource, err := db.ResourceIDByName(identity)
	if err != nil || resource.Type != "stage-type" || !build.Stage {
		t.Fatalf("stage resource = %+v, build.Stage=%v, err=%v", resource, build.Stage, err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "builds", build.ID))
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("stored stage differs: %v", err)
	}
}

func TestImplantBuildSaveUpdateFileRollbackAndPublish(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	identity := "update-file-" + models.NewUUID().String()
	artifact := filepath.Join(root, "artifact")
	oldData, newData := []byte("old binary"), []byte("new binary")
	if err := os.WriteFile(artifact, oldData, 0600); err != nil {
		t.Fatal(err)
	}
	build := &clientpb.ImplantBuild{Name: identity}
	if err := ImplantBuildSave(build, &clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", LimitHostname: identity}, artifact); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Session().Where("id = ?", build.ID).Delete(&models.ImplantBuild{})
		db.Session().Where("id = ?", build.ImplantConfigID).Delete(&models.ImplantConfig{})
		db.Session().Where("name = ?", identity).Delete(&models.ResourceID{})
	})
	oldHash := build.SHA256
	failure := errors.New("update publication failed")
	if err := implantBuildUpdateFile(build, newData, func(staged, final string) error {
		if _, err := os.Stat(staged); err != nil {
			return err
		}
		if _, err := os.Stat(final); !os.IsNotExist(err) {
			return errors.New("original file was not moved before publishing")
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatalf("update failure = %v", err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "builds", build.ID))
	if err != nil || !bytes.Equal(stored, oldData) || build.SHA256 != oldHash {
		t.Fatalf("failed update changed caller or file: %v", err)
	}
	var persisted models.ImplantBuild
	if err := db.Session().Where("id = ?", build.ID).First(&persisted).Error; err != nil || persisted.SHA256 != oldHash {
		t.Fatalf("failed update changed database SHA256=%q, err=%v", persisted.SHA256, err)
	}
	build.SHA256 = "stale-hash"
	if err := ImplantBuildUpdateFile(build, newData); err == nil {
		t.Fatal("stale build metadata overwrote a newer build")
	}
	build.SHA256 = oldHash
	if err := ImplantBuildUpdateFile(build, newData); err != nil {
		t.Fatalf("publish update: %v", err)
	}
	stored, err = os.ReadFile(filepath.Join(root, "builds", build.ID))
	if err != nil || !bytes.Equal(stored, newData) || build.SHA256 == oldHash {
		t.Fatalf("successful update did not publish: %v", err)
	}
}

func TestImplantBuildSaveRejectsSiblingPrefixPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	sibling := root + "-sibling"
	if err := os.Mkdir(sibling, 0700); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sibling)
	artifact := filepath.Join(sibling, "artifact")
	if err := os.WriteFile(artifact, []byte("test binary"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ImplantBuildSave(&clientpb.ImplantBuild{Name: "invalid-path"}, &clientpb.ImplantConfig{}, artifact); err == nil {
		t.Fatal("sibling prefix path accepted")
	}
}
