package socks

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// SocksStartCmd - Add a new tunneled port forward.
func SocksStartCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	// listener
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetString("port")
	bindAddr := fmt.Sprintf("%s:%s", host, port)
	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		con.PrintErrorf("Socks5 Listen %s \n", err.Error())
		return
	}
	username, _ := cmd.Flags().GetString("user")
	if err != nil {
		return
	}
	password := ""
	if username != "" {
		con.PrintWarnf("SOCKS proxy authentication credentials are tunneled to the implant\n")
		con.PrintWarnf("These credentials are recoverable from the implant's memory!\n\n")
		confirm := false
		prompt := &survey.Confirm{Message: "Do you understand the implication?"}
		survey.AskOne(prompt, &confirm, nil)
		if !confirm {
			return
		}
		password = randomPassword()
		con.Printf("\n")
	}

	channelProxy := &core.TcpProxy{
		Rpc:             con.Rpc,
		Session:         session,
		Listener:        ln,
		BindAddr:        bindAddr,
		Username:        username,
		Password:        password,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}

	go func(channelProxy *core.TcpProxy) {
		core.SocksProxies.Start(channelProxy)
	}(core.SocksProxies.Add(channelProxy).ChannelProxy)
	con.PrintInfof("Started SOCKS5 %s %s %s %s\n", host, port, username, password)
	con.PrintWarnf("In-band SOCKS proxies can be a little unstable depending on protocol\n")
}

func randomPassword() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return base64.RawStdEncoding.EncodeToString(buf)
}
