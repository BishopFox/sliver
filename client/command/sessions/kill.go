package sessions

import (
	"context"
	"errors"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"gopkg.in/AlecAivazis/survey.v1"
)

// KillCmd - Kill the active session (not to be confused with TerminateCmd)
func KillCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		con.PrintWarnf("No active session\n")
		return
	}
	// Confirm with the user, just in case they confused kill with terminate
	confirm := false
	prompt := &survey.Confirm{Message: "Kill the active session?"}
	survey.AskOne(prompt, &confirm, nil)
	if !confirm {
		return
	}

	err := killSession(session, ctx.Flags.Bool("force"), con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Killed %s (%d)\n", session.Name, session.ID)
	con.ActiveSession.Background()
}

func killSession(session *clientpb.Session, force bool, con *console.SliverConsoleClient) error {
	if session == nil {
		return errors.New("session does not exist")
	}
	_, err := con.Rpc.KillSession(context.Background(), &sliverpb.KillSessionReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
		Force: force,
	})
	return err
}
