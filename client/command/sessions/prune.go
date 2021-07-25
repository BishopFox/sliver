package sessions

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

// SessionsPruneCmd - Forcefully kill stale sessions
func SessionsPruneCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(sessions.GetSessions()) == 0 {
		con.PrintInfof("No sessions to prune\n")
		return
	}
	for _, session := range sessions.GetSessions() {
		if session.IsDead {
			con.Printf("Pruning session #%d ...", session.ID)
			err = killSession(session, true, con)
			if err != nil {
				con.Printf("failed!\n")
				con.PrintErrorf("%s\n", err)
			} else {
				con.Printf("done!\n")
			}
		}
	}
}
