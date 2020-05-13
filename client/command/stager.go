package command

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
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

// stage-listener --url [tcp://ip:port | http://ip:port ] --profile name
func stageListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	var implantProfile *clientpb.ImplantProfile
	profileName := ctx.Flags.String("profile")
	listenerURL := ctx.Flags.String("url")

	if profileName == "" || listenerURL == "" {
		fmt.Println(Warn + "missing required flags, see `help stage-listener` for more info")
		return
	}

	// parse listener url
	stagingURL, err := url.Parse(listenerURL)
	if err != nil {
		fmt.Printf(Warn + "listener-url format not supported")
		return
	}
	stagingPort, err := strconv.ParseUint(stagingURL.Port(), 10, 32)
	if err != nil {
		fmt.Printf(Warn+"error parsing staging port: %v\n", err)
		return
	}

	// get profile
	profiles := getSliverProfiles(rpc)
	if profiles == nil {
		return
	}

	if len(*profiles) == 0 {
		fmt.Printf(Info+"No profiles, create one with `%s`\n", consts.NewProfileStr)
		return
	}

	for name, profile := range *profiles {
		if name == profileName {
			implantProfile = profile
		}
	}

	if implantProfile.GetName() == "" {
		fmt.Printf(Warn + "could not find the implant name from the profile\n")
		return
	}

	stage2, err := getSliverBinary(*implantProfile, rpc)
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}

	switch stagingURL.Scheme {
	case "http":
		ctrl := make(chan bool)
		go spin.Until("Starting HTTP staging listener...", ctrl)
		stageListener, err := rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_HTTP,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(Warn+"Error starting HTTP staging listener: %v\n", err)
			return
		}
		fmt.Printf(Info+"Job %d (http) started\n", stageListener.GetJobID())
	case "https":
		ctrl := make(chan bool)
		go spin.Until("Starting HTTPS staging listener...", ctrl)
		stageListener, err := rpc.StartHTTPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_HTTPS,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(Warn+"Error starting HTTPS staging listener: %v\n", err)
			return
		}
		fmt.Printf(Info+"Job %d (https) started\n", stageListener.GetJobID())
	case "tcp":
		ctrl := make(chan bool)
		go spin.Until("Starting TCP staging listener...", ctrl)
		stageListener, err := rpc.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
			Protocol: clientpb.StageProtocol_TCP,
			Data:     stage2,
			Host:     stagingURL.Hostname(),
			Port:     uint32(stagingPort),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(Warn+"Error starting TCP staging listener: %v\n", err)
			return
		}
		fmt.Printf(Info+"Job %d (tcp) started\n", stageListener.GetJobID())

	default:
		fmt.Printf(Warn+"Unsupported staging protocol: %s\n", stagingURL.Scheme)
		return
	}
}

func getSliverBinary(profile clientpb.ImplantProfile, rpc rpcpb.SliverRPCClient) ([]byte, error) {
	var data []byte
	// get implant builds
	builds, err := rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return data, err
	}

	implantName := buildImplantName(profile.GetConfig().GetName())
	_, ok := builds.GetConfigs()[implantName]
	if implantName == "" || !ok {
		// no built implant found for profile, generate a new one
		fmt.Printf(Info+"No builds found for profile %s, generating a new one\n", profile.GetName())
		ctrl := make(chan bool)
		go spin.Until("Compiling, please wait ...", ctrl)
		generated, err := rpc.Generate(context.Background(), &clientpb.GenerateReq{
			Config: profile.GetConfig(),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Println("Error generating implant")
			return data, err
		}
		data = generated.GetFile().GetData()
		profile.Config.Name = buildImplantName(generated.GetFile().GetName())
		_, err = rpc.SaveImplantProfile(context.Background(), &profile)
		if err != nil {
			fmt.Println("Error updating implant profile")
			return data, err
		}
	} else {
		// Found a build, reuse that one
		fmt.Printf(Info+"Sliver name for profile: %s\n", implantName)
		regenerate, err := rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
			ImplantName: profile.GetConfig().GetName(),
		})

		if err != nil {
			return data, err
		}
		data = regenerate.GetFile().GetData()
	}
	return data, err
}

func buildImplantName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
