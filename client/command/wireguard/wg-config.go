package wireguard

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

var wgQuickTemplate = `[Interface]
Address = {{.ClientIP}}/16
ListenPort = 51902
PrivateKey = {{.PrivateKey}}
MTU = 1420

[Peer]
PublicKey = {{.ServerPublicKey}}
AllowedIPs = {{.AllowedSubnet}}
Endpoint = <configure yourself>`

type wgQuickConfig struct {
	ClientIP        string
	PrivateKey      string
	ServerPublicKey string
	AllowedSubnet   string
}

// WGConfigCmd - Generate a WireGuard client configuration.
// WGConfigCmd - Generate WireGuard 客户 configuration.
func WGConfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	wgConfig, err := con.Rpc.GenerateWGClientConfig(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	clientPrivKeyBytes, err := hex.DecodeString(wgConfig.ClientPrivateKey)
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	serverPubKeyBytes, err := hex.DecodeString(wgConfig.ServerPubKey)
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	tmpl, err := template.New("wgQuick").Parse(wgQuickTemplate)
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	clientIP, network, err := net.ParseCIDR(wgConfig.ClientIP + "/16")
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	output := bytes.Buffer{}
	tmpl.Execute(&output, wgQuickConfig{
		ClientIP:        clientIP.String(),
		PrivateKey:      base64.StdEncoding.EncodeToString(clientPrivKeyBytes),
		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPubKeyBytes),
		AllowedSubnet:   network.String(),
	})

	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		con.PrintInfof("New client config:\n")
		con.Println(output.String())
	} else {
		if !strings.HasSuffix(save, ".conf") {
			save += ".conf"
		}
		err = os.WriteFile(save, output.Bytes(), 0o600)
		if err != nil {
			con.PrintErrorf("Error: %s\n", err)
			return
		}
		con.PrintInfof("Wrote conf: %s\n", save)
	}
}
