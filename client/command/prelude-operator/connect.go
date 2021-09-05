package operator

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/prelude"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

func ConnectCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	url := ctx.Args.String("connection-string")
	aesKey := ctx.Flags.String("aes-key")
	agentRange := ctx.Flags.String("range")
	skipExisting := ctx.Flags.Bool("skip-existing")

	config := &prelude.OperatorConfig{
		Range:       agentRange,
		OperatorURL: url,
		RPC:         con.Rpc,
		AESKey:      aesKey,
	}

	sessionMapper := prelude.InitSessionMapper(config)

	con.PrintInfof("Connected to Operator at %s\n", url)
	if !skipExisting {
		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Could not get session list: %s", err)
			return
		}
		if len(sessions.Sessions) > 0 {
			con.PrintInfof("Adding existing sessions ...\n")
			for _, sess := range sessions.Sessions {
				err = sessionMapper.AddSession(sess)
				if err != nil {
					con.PrintErrorf("Could not add session %s to session mapper: %s", sess.Name, err)
				}
			}
			con.PrintInfof("Done !\n")

		}
	}
}
