package command

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"text/template"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
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

func getWGClientConfig(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	wgConfig, err := rpc.GenerateWGClientConfig(context.Background(), &commonpb.Empty{})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	clientPrivKeyBytes, err := hex.DecodeString(wgConfig.ClientPrivateKey)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	serverPubKeyBytes, err := hex.DecodeString(wgConfig.ServerPubKey)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	tmpl, err := template.New("wgQuick").Parse(wgQuickTemplate)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	clientIP, network, err := net.ParseCIDR(wgConfig.ClientIP + "/16")
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	output := bytes.Buffer{}
	tmpl.Execute(&output, wgQuickConfig{
		ClientIP:        clientIP.String(),
		PrivateKey:      base64.StdEncoding.EncodeToString(clientPrivKeyBytes),
		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPubKeyBytes),
		AllowedSubnet:   network.String(),
	})
	fmt.Println(Info + "New client config:")
	fmt.Println(output.String())
}
