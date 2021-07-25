package exec

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// SSHCmd - A built-in SSH client command for the remote system (doesn't shell out)
func SSHCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var (
		privKey []byte
		err     error
	)
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	username := ctx.Flags.String("login")
	if username == "" {
		username = session.GetUsername()
	}

	port := ctx.Flags.Uint("port")
	privateKeypath := ctx.Flags.String("private-key")
	if privateKeypath != "" {
		privKey, err = ioutil.ReadFile(privateKeypath)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	password := ctx.Flags.String("password")

	hostname := ctx.Args.String("hostname")
	command := ctx.Args.StringList("command")

	if password == "" && len(privKey) == 0 && !ctx.Flags.Bool("skip-loot") {
		oldUsername := username
		username, password, privKey = tryCredsFromLoot(con)
		if username == "" {
			username = oldUsername
		}
	}

	commandResp, err := con.Rpc.RunSSHCommand(context.Background(), &sliverpb.SSHCommandReq{
		Username: username,
		Hostname: hostname,
		Port:     uint32(port),
		PrivKey:  privKey,
		Password: password,
		Command:  strings.Join(command, " "),
		Request:  con.ActiveSession.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if commandResp.Response != nil && commandResp.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", commandResp.Response.Err)
		if commandResp.StdErr != "" {
			con.PrintErrorf("StdErr: %s\n", commandResp.StdErr)
		}
		return
	}
	if commandResp.StdOut != "" {
		con.PrintInfof("Output:")
		con.Println(commandResp.StdOut)
		if commandResp.StdErr != "" {
			con.PrintInfof("StdErr")
			con.Println(commandResp.StdErr)
		}
	}
}

func tryCredsFromLoot(con *console.SliverConsoleClient) (string, string, []byte) {
	var (
		username string
		password string
		privKey  []byte
	)
	confirm := false
	prompt := &survey.Confirm{Message: "No credentials provided, use from loot?"}
	survey.AskOne(prompt, &confirm, nil)
	if confirm {
		cred, err := loot.SelectCredentials(con)
		if err != nil {
			con.PrintErrorf("Invalid loot data, will try to use the SSH agent")
		} else {
			switch cred.CredentialType {
			case clientpb.CredentialType_API_KEY:
				privKey = []byte(cred.Credential.APIKey)
			case clientpb.CredentialType_USER_PASSWORD:
				username = cred.Credential.User
				password = cred.Credential.Password
			}
		}
	}
	return username, password, privKey
}
