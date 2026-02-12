package exec

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// SSHCmd - A built-in SSH client command for the remote system (doesn't shell out).
// SSHCmd - A built__PH0__ SSH 远程系统的客户端命令（不 shell 输出）。
func SSHCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var (
		privKey []byte
		err     error
	)
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	username, _ := cmd.Flags().GetString("login")
	if username == "" {
		username = session.GetUsername()
	}

	port, _ := cmd.Flags().GetUint("port")
	privateKeyPath, _ := cmd.Flags().GetString("private-key")
	if privateKeyPath != "" {
		privKey, err = os.ReadFile(privateKeyPath)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	password, _ := cmd.Flags().GetString("password")

	hostname := args[0]
	command := args[1:]

	kerberosRealm, _ := cmd.Flags().GetString("kerberos-realm")
	kerberosConfig, _ := cmd.Flags().GetString("kerberos-config")
	kerberosKeytabFile, _ := cmd.Flags().GetString("kerberos-keytab")

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

	skipLoot, _ := cmd.Flags().GetBool("skip-loot")

	if password == "" && len(privKey) == 0 && !skipLoot {
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
		Request:  con.ActiveTarget.Request(cmd),
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

// PrintSSHCmd - Print the ssh command response.
// PrintSSHCmd - Print ssh 命令 response.
func PrintSSHCmd(sshCmd *sliverpb.SSHCommand, con *console.SliverClient) {
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

func tryCredsFromLoot(con *console.SliverClient) (string, string, []byte) {
	var (
		username string
		password string
		privKey  []byte
	)
	confirm := false
	_ = forms.Confirm("No credentials provided, use from loot?", &confirm)
	if confirm {
		// cred, err := loot.SelectCredentials(con)
		// 信用，错误：= loot.SelectCredentials(con)
		// if err != nil {
		// 如果错误！= nil {
		// 	con.PrintErrorf("Invalid loot data, will try to use the SSH agent")
		// 	con.PrintErrorf(__PH0__)
		// } else {
		// } 别的 {
		// 	switch cred.CredentialType {
		// 	开关 cred.CredentialType {
		// 	case clientpb.CredentialType_API_KEY:
		// 	案例 clientpb.CredentialType_API_KEY:
		// 		privKey = []byte(cred.Credential.APIKey)
		// 		privKey = []字节(cred.Credential.APIKey)
		// 	case clientpb.CredentialType_USER_PASSWORD:
		// 	案例 clientpb.CredentialType_USER_PASSWORD:
		// 		username = cred.Credential.User
		// 		用户名 = cred.Credential.User
		// 		password = cred.Credential.Password
		// 		密码 = cred.Credential.Password
		// 	}
		// }
	}
	return username, password, privKey
}
