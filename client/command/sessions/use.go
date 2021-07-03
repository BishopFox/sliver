package sessions

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

func UseCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.GetSession(ctx.Args.String("session"))
	if session != nil {
		con.ActiveSession.Set(session)
		con.PrintInfof("Active session %s (%d)\n", session.Name, session.ID)
	} else {
		con.PrintErrorf("Invalid session name or session number '%s'\n", ctx.Args.String("session"))
	}
}
