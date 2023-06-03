package export

import (
	"encoding/json"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/export"
)

func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	m, _ := json.Marshal(export.Export{
		Meta:   meta,
		Values: values,
	})
	return string(m)
}
