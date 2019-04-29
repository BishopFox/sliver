package generate

import (
	"testing"
)

func TestProfileByName(t *testing.T) {
	name := "foobar"
	config := &SliverConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
	}
	err := ProfileSave(name, config)
	if err != nil {
		t.Errorf("%v", err)
	}

	profile, err := ProfileByName(name)
	if err != nil {
		t.Errorf("%v", err)
	}

	if profile.GOOS != config.GOOS {
		t.Errorf("Fetched data does not match saved data")
	}
}
