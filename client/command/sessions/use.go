package sessions

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func UseCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	sessionArg := ctx.Args.String("session")
	if sessionArg != "" {
		session := con.GetSession(sessionArg)
		if session == nil {
			con.PrintErrorf("Invalid session name or session number '%s'\n", ctx.Args.String("session"))
		} else {
			activate(session, con)
		}
	} else {
		session, err := SelectSession(false, con)
		if session != nil {
			activate(session, con)
		} else if err != nil {
			switch err {
			case ErrNoSessions:
				con.PrintErrorf("No sessions available\n")
			case ErrNoSelection:
				con.PrintErrorf("No session selected\n")
			default:
				con.PrintErrorf("%s\n", err)
			}
		}
	}
}

func activate(session *clientpb.Session, con *console.SliverConsoleClient) {
	con.ActiveSession.Set(session)
	con.PrintInfof("Active session %s (%d)\n", session.Name, session.ID)
}
