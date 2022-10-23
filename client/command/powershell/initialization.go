package powershell

import (
	"io/ioutil"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
)

var PSpath = ""

func init() {

	_ = ioutil.WriteFile(filepath.Join(assets.GetResourcesDir(), "PS.exe"), PSrunner, 0o600)
	PSpath = filepath.Join(assets.GetResourcesDir(), "PS.exe")
}
