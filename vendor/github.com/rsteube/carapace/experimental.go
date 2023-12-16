package carapace

import (
	"encoding/json"

	"github.com/rsteube/carapace/internal/config"
	"github.com/rsteube/carapace/internal/export"
	"github.com/rsteube/carapace/pkg/x"
	"github.com/spf13/cobra"
)

func init() {
	x.ClearStorage = func() {
		storage = make(_storage)
	}

	x.Complete = func(cmd *cobra.Command, args ...string) (*export.Export, error) {
		initHelpCompletion(cmd)
		action, context := traverse(cmd, args[2:])

		if err := config.Load(); err != nil {
			return nil, err
		}

		output := action.Invoke(context).value("export", "")
		var e export.Export
		if err := json.Unmarshal([]byte(output), &e); err != nil {
			return nil, err
		}
		return &e, nil
	}
}
