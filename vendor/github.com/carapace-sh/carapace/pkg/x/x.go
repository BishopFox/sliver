// Package x contains experimental functions
package x

import (
	"github.com/carapace-sh/carapace/internal/export"
	"github.com/spf13/cobra"
)

var ClearStorage func()
var Complete func(cmd *cobra.Command, args ...string) (*export.Export, error)
