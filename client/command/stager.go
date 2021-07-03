package command

// /*
// 	Sliver Implant Framework
// 	Copyright (C) 2019  Bishop Fox

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
// 	"context"
// 	"fmt"
// 	"net/url"
// 	"path/filepath"
// 	"strconv"
// 	"strings"

// 	"github.com/bishopfox/sliver/client/spin"
// 	"github.com/bishopfox/sliver/protobuf/clientpb"
// 	"github.com/bishopfox/sliver/protobuf/commonpb"
// 	"github.com/bishopfox/sliver/protobuf/rpcpb"
// 	"github.com/desertbit/grumble"
// )

// // stage-listener --url [tcp://ip:port | http://ip:port ] --profile name
// func stageListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	profileName := ctx.Flags.String("profile")
// 	listenerURL := ctx.Flags.String("url")

// 	if profileName == "" || listenerURL == "" {
// 		fmt.Println(Warn + "missing required flags, see `help stage-listener` for more info")
// 		return
// 	}

// 	// parse listener url
// 	stagingURL, err := url.Parse(listenerURL)
// 	if err != nil {
// 		fmt.Printf(Warn + "listener-url format not supported")
// 		return
// 	}
// 	stagingPort, err := strconv.ParseUint(stagingURL.Port(), 10, 32)
// 	if err != nil {
// 		fmt.Printf(Warn+"error parsing staging port: %v\n", err)
// 		return
// 	}

// 	profile := getImplantProfileByName(rpc, profileName)
// 	if profile != nil {

// 	}
// 	stage2, err := getSliverBinary(profile, rpc)
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %v\n", err)
// 		return
// 	}

// 	switch stagingURL.Scheme {
// 	case "http":
// 		ctrl := make(chan bool)
// 		go spin.Until("Starting HTTP staging listener...", ctrl)
// 		stageListener, err := rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
// 			Protocol: clientpb.StageProtocol_HTTP,
// 			Data:     stage2,
// 			Host:     stagingURL.Hostname(),
// 			Port:     uint32(stagingPort),
// 		})
// 		ctrl <- true
// 		<-ctrl
// 		if err != nil {
// 			fmt.Printf(Warn+"Error starting HTTP staging listener: %v\n", err)
// 			return
// 		}
// 		fmt.Printf(Info+"Job %d (http) started\n", stageListener.GetJobID())
// 	case "https":
// 		cert, key, err := getLocalCertificatePair(ctx)
// 		if err != nil {
// 			fmt.Printf("\n"+Warn+"Failed to load local certificate %v", err)
// 			return
// 		}
// 		ctrl := make(chan bool)
// 		go spin.Until("Starting HTTPS staging listener...", ctrl)
// 		stageListener, err := rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
// 			Protocol: clientpb.StageProtocol_HTTPS,
// 			Data:     stage2,
// 			Host:     stagingURL.Hostname(),
// 			Port:     uint32(stagingPort),
// 			Cert:     cert,
// 			Key:      key,
// 			ACME:     ctx.Flags.Bool("lets-encrypt"),
// 		})
// 		ctrl <- true
// 		<-ctrl
// 		if err != nil {
// 			fmt.Printf(Warn+"Error starting HTTPS staging listener: %v\n", err)
// 			return
// 		}
// 		fmt.Printf(Info+"Job %d (https) started\n", stageListener.GetJobID())
// 	case "tcp":
// 		ctrl := make(chan bool)
// 		go spin.Until("Starting TCP staging listener...", ctrl)
// 		stageListener, err := rpc.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
// 			Protocol: clientpb.StageProtocol_TCP,
// 			Data:     stage2,
// 			Host:     stagingURL.Hostname(),
// 			Port:     uint32(stagingPort),
// 		})
// 		ctrl <- true
// 		<-ctrl
// 		if err != nil {
// 			fmt.Printf(Warn+"Error starting TCP staging listener: %v\n", err)
// 			return
// 		}
// 		fmt.Printf(Info+"Job %d (tcp) started\n", stageListener.GetJobID())

// 	default:
// 		fmt.Printf(Warn+"Unsupported staging protocol: %s\n", stagingURL.Scheme)
// 		return
// 	}
// }
