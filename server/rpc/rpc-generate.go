package rpc

import (
	"errors"
	"io/ioutil"
	"path"
	clientpb "sliver/protobuf/client"
	"sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

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

func rpcRegenerate(req []byte, resp RPCResponse) {
	regenReq := &clientpb.Regenerate{}
	err := proto.Unmarshal(req, regenReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliverConfig, _ := generate.SliverConfigByName(regenReq.SliverName)
	sliverFileData, err := generate.SliverFileByName(regenReq.SliverName)
	regenerated := &clientpb.Regenerate{SliverName: regenReq.SliverName}
	if err != nil {
		resp([]byte{}, err)
		return
	}

	if sliverFileData != nil && sliverConfig != nil {
		regenerated.File = &clientpb.File{
			Name: sliverConfig.FileName,
			Data: sliverFileData,
		}
	}
	data, err := proto.Marshal(regenerated)
	resp(data, err)
}

func rpcListSliverBuilds(_ []byte, resp RPCResponse) {
	configs, err := generate.SliverConfigMap()
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliverBuilds := &clientpb.SliverBuilds{
		Configs: map[string]*clientpb.SliverConfig{},
	}
	for name, cfg := range configs {
		sliverBuilds.Configs[name] = cfg.ToProtobuf()
	}
	data, err := proto.Marshal(sliverBuilds)
	resp(data, err)
}

func rpcListCanaries(_ []byte, resp RPCResponse) {
	jsonCanaries, err := generate.ListCanaries()
	if err != nil {
		resp([]byte{}, err)
	}
	rpcLog.Infof("Found %d canaries", len(jsonCanaries))
	canaries := []*clientpb.DNSCanary{}
	for _, canary := range jsonCanaries {
		canaries = append(canaries, canary.ToProtobuf())
	}
	data, err := proto.Marshal(&clientpb.Canaries{
		Canaries: canaries,
	})
	resp(data, err)
}

func rpcProfiles(_ []byte, resp RPCResponse) {
	profiles := &clientpb.Profiles{List: []*clientpb.Profile{}}
	for name, config := range generate.Profiles() {
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
		err = generate.ProfileSave(profile.Name, config)
	} else {
		err = errors.New("Invalid profile name")
	}
	resp([]byte{}, err)
}
