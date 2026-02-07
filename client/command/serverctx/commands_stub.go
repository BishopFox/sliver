//go:build !client

package serverctx

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// Commands is a no-op when building without the `client` build tag (e.g. sliver-server).
func Commands(_ *console.SliverClient) []*cobra.Command {
	return nil
}
