package wireguard

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
