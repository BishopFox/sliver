package sessions

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// BackgroundCmd - Background the active session
func BackgroundCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	con.ActiveTarget.Background()
	con.PrintInfof("Background ...\n")
}
