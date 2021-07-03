package command

// /*
// 	Sliver Implant Framework
// 	Copyright (C) 2021  Bishop Fox

// 	This program is free software: you can redistribute it and/or modify
// 	it under the terms of the GNU General Public License as published by
// 	the Free Software Foundation, either version 3 of the License, or
// 	(at your option) any later version.

// 	This program is distributed in the hope that it will be useful,
// 	but WITHOUT ANY WARRANTY; without even the implied warranty of
// 	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 	GNU General Public License for more details.

// 	You should have received a copy of the GNU General Public License
// 	along with this program.  If not, see <https://www.gnu.org/licenses/>.
// */

// import (
// 	"bytes"
// 	"context"
// 	"encoding/base64"
// 	"encoding/hex"
// 	"fmt"
// 	"io/ioutil"
// 	"net"
// 	"strings"
// 	"text/template"

// 	"github.com/bishopfox/sliver/protobuf/commonpb"
// 	"github.com/bishopfox/sliver/protobuf/rpcpb"
// 	"github.com/desertbit/grumble"
// )

// var wgQuickTemplate = `[Interface]
// Address = {{.ClientIP}}/16
// ListenPort = 51902
// PrivateKey = {{.PrivateKey}}
// MTU = 1420

// [Peer]
// PublicKey = {{.ServerPublicKey}}
// AllowedIPs = {{.AllowedSubnet}}
// Endpoint = <configure yourself>`

// type wgQuickConfig struct {
// 	ClientIP        string
// 	PrivateKey      string
// 	ServerPublicKey string
// 	AllowedSubnet   string
// }

// func getWGClientConfig(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	wgConfig, err := rpc.GenerateWGClientConfig(context.Background(), &commonpb.Empty{})
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %s\n", err)
// 		return
// 	}
// 	clientPrivKeyBytes, err := hex.DecodeString(wgConfig.ClientPrivateKey)
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %s\n", err)
// 		return
// 	}
// 	serverPubKeyBytes, err := hex.DecodeString(wgConfig.ServerPubKey)
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %s\n", err)
// 		return
// 	}
// 	tmpl, err := template.New("wgQuick").Parse(wgQuickTemplate)
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %s\n", err)
// 		return
// 	}
// 	clientIP, network, err := net.ParseCIDR(wgConfig.ClientIP + "/16")
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %s\n", err)
// 		return
// 	}
// 	output := bytes.Buffer{}
// 	tmpl.Execute(&output, wgQuickConfig{
// 		ClientIP:        clientIP.String(),
// 		PrivateKey:      base64.StdEncoding.EncodeToString(clientPrivKeyBytes),
// 		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPubKeyBytes),
// 		AllowedSubnet:   network.String(),
// 	})

// 	save := ctx.Flags.String("save")
// 	if save == "" {
// 		fmt.Println(Info + "New client config:")
// 		fmt.Println(output.String())
// 	} else {
// 		if !strings.HasSuffix(save, ".conf") {
// 			save += ".conf"
// 		}
// 		err = ioutil.WriteFile(save, []byte(output.String()), 0600)
// 		if err != nil {
// 			fmt.Printf(Warn+"Error: %s\n", err)
// 			return
// 		}
// 		fmt.Printf(Info+"Wrote conf: %s\n", save)
// 	}
// }
