package export

import (
	"encoding/json"
	"runtime/debug"
	"sort"

	"github.com/rsteube/carapace/internal/common"
)

type Export struct {
	Version string `json:"version"`
	common.Meta
	Values common.RawValues `json:"values"`
}

func (e Export) MarshalJSON() ([]byte, error) {
	sort.Sort(common.ByValue(e.Values))
	return json.Marshal(&struct {
		Version string `json:"version"`
		common.Meta
		Values common.RawValues `json:"values"`
	}{
		Version: version(),
		Meta:    e.Meta,
		Values:  e.Values,
	})
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/rsteube/carapace" {
				return dep.Version
			}
		}
	}
	return "unknown"
}
