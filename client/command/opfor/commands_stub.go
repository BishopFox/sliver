//go:build !client

package opfor

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// Commands is unavailable in builds which do not include the client tag.
func Commands(_ *console.SliverClient) []*cobra.Command { return nil }

// ServerCommands is unavailable in builds which do not include the client tag.
func ServerCommands(_ *console.SliverClient) []*cobra.Command { return nil }

// RegisterCommands is a no-op outside client builds.
func RegisterCommands(_ *cobra.Command, _ *console.SliverClient) {}
