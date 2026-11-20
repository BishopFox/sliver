package db

import (
	"errors"
	"testing"

	"github.com/bishopfox/sliver/server/db/models"
)

func TestDeleteProfilePreservesCompletedBuildReferences(t *testing.T) {
	name := "preserve-" + models.NewUUID().String()
	profile := &models.ImplantProfile{
		Name: name,
		ImplantConfig: &models.ImplantConfig{
			GOOS: "windows", GOARCH: "amd64", TemplateName: "sliver",
		},
	}
	if err := Session().Create(profile).Error; err != nil {
		t.Fatal(err)
	}
	configID := profile.ImplantConfig.ID
	build := &models.ImplantBuild{Name: "build-" + models.NewUUID().String(), ImplantConfigID: configID}
	if err := Session().Create(build).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = Session().Delete(build).Error
		_ = Session().Delete(profile.ImplantConfig).Error
		_ = Session().Delete(profile).Error
	})
	if err := DeleteProfile(name); err != nil {
		t.Fatalf("delete profile definition: %v", err)
	}
	var removed models.ImplantProfile
	if err := Session().Where("name = ?", name).First(&removed).Error; !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("profile definition remains: %v", err)
	}
	var savedConfig models.ImplantConfig
	if err := Session().Where("id = ?", configID).First(&savedConfig).Error; err != nil || savedConfig.ImplantProfileID != nil {
		t.Fatalf("linked configuration was removed or still points to profile: %+v, %v", savedConfig, err)
	}
	var savedBuild models.ImplantBuild
	if err := Session().Where("id = ?", build.ID).First(&savedBuild).Error; err != nil || savedBuild.ImplantConfigID != configID {
		t.Fatalf("completed build was removed or detached from its config: %+v, %v", savedBuild, err)
	}
}
