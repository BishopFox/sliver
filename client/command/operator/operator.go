package operator

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/prelude"
	"github.com/desertbit/grumble"
)

func OperatorCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	if prelude.SessionMapper != nil {
		con.PrintInfof("Connected to Operator at %s\n", prelude.SessionMapper.GetConfig().OperatorURL)
		return
	}
	con.PrintInfof("Not connected to any Operator server. Use `operator connect` to connect to one.")
}
