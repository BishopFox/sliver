package sessions

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

func BackgroundCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	con.ActiveSession.Background()
	con.PrintInfof("Background ...\n")
}
