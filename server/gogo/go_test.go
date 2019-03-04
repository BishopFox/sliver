package gogo

import (
	"fmt"
	"sliver/server/assets"
	"testing"
)

func TestGoGoVersion(t *testing.T) {
	appDir := assets.GetRootAppDir()
	winConfig := GoConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		GOROOT: GetGoRootDir(appDir),
	}
	_, err := GoVersion(winConfig)
	if err != nil {
		t.Errorf(fmt.Sprintf("version cmd failed %v", err))
	}
}
