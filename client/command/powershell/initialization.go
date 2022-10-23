package powershell

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
)

var PSpath = ""

func init() {

	if _, err := os.Stat(filepath.Join(assets.GetResourcesDir(), "PS.exe")); err != nil {
		_ = ioutil.WriteFile(filepath.Join(assets.GetResourcesDir(), "PS.exe"), PSrunner, 0o600)
	}
	PSpath = filepath.Join(assets.GetResourcesDir(), "PS.exe")
}
