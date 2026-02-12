//go:build !client

package serverctx

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// Commands is a no-op when building without the `client` build tag (e.g. sliver-server).
// 在没有 __PH0__ 构建标签 (e.g. sliver__PH2__) 的情况下构建时，Commands 是 no__PH1__。
func Commands(_ *console.SliverClient) []*cobra.Command {
	return nil
}
