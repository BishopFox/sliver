package rpc

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
	"errors"
	"io/ioutil"
	"path"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/generate"

	"github.com/golang/protobuf/proto"
)

// Generate - Generate a new implant
func Generate(ctx context.Context, req *clientpb.GenerateReq) (*clientpb.Generate, error) {
	var fPath string
	var err error
	config := generate.SliverConfigFromProtobuf(req.Config)
	if config == nil {
		return nil, errors.New("Invalid implant config")
	}
	switch req.Config.Format {
	case clientpb.ImplantConfig_EXECUTABLE:
		fPath, err = generate.SliverExecutable(config)
	case clientpb.ImplantConfig_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(config)
	case clientpb.ImplantConfig_SHELLCODE:
		fPath, err = generate.SliverSharedLibrary(config)
		if err != nil {
			return nil, err
		}
		fPath, err = generate.ShellcodeRDIToFile(fPath, "")
		if err != nil {
			return nil, err
		}
	}

	filename := path.Base(fPath)
	filedata, err := ioutil.ReadFile(fPath)

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: filename,
			Data: filedata,
		},
	}, nil
}

// Regenerate - Regenerate a previously generated implant
func Regenerate(ctx context.Context, req *clientpb.RegenerateReq) (*clientpb.Generate, error) {

	config, err := generate.SliverConfigByName(req.ImplantName)
	if err != nil {
		return nil, err
	}

	fileData, err := generate.SliverFileByName(req.ImplantName)
	if err != nil {
		return nil, err
	}

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: config.FileName,
			Data: fileData,
		},
	}, nil
}

// ImplantBuilds - List existing implant builds
func ImplantBuilds(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	configs, err := generate.SliverConfigMap()
	if err != nil {
		return nil, err
	}
	builds := &clientpb.ImplantBuilds{
		Configs: map[string]*clientpb.ImplantConfig{},
	}
	for name, config := range configs {
		builds.Configs[name] = config.ToProtobuf()
	}
	return builds, nil
}

// Canaries - List existing canaries
func Canaries(ctx context.Context, _ *commonpb.Empty) (*clientpb.Canaries, error) {
	jsonCanaries, err := generate.ListCanaries()
	if err != nil {
		return nil, err
	}

	rpcLog.Infof("Found %d canaries", len(jsonCanaries))
	canaries := []*clientpb.DNSCanary{}
	for _, canary := range jsonCanaries {
		canaries = append(canaries, canary.ToProtobuf())
	}

	return &clientpb.Canaries{
		Canaries: canaries,
	}, nil
}

// Profiles - List profiles
func Profiles(ctx context.Context, _ *commonpb.Empty) (*clientpb.Profiles, error) {
	profiles := &clientpb.Profiles{List: []*clientpb.Profile{}}
	for name, config := range generate.Profiles() {
		profiles.List = append(profiles.List, &clientpb.Profile{
			Name:   name,
			Config: config.ToProtobuf(),
		})
	}
	return profiles, nil
}

func NewProfile(req []byte, timeout time.Duration, resp RPCResponse) {
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
