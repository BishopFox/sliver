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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/generate"
)

// Generate - Generate a new implant
func (rpc *Server) Generate(ctx context.Context, req *clientpb.GenerateReq) (*clientpb.Generate, error) {
	var fPath string
	var err error
	config := generate.ImplantConfigFromProtobuf(req.Config)
	if config == nil {
		return nil, errors.New("Invalid implant config")
	}
	switch req.Config.Format {
	case clientpb.ImplantConfig_SERVICE:
		fallthrough
	case clientpb.ImplantConfig_EXECUTABLE:
		fPath, err = generate.SliverExecutable(config)
		break
	case clientpb.ImplantConfig_SHARED_LIB:
		fPath, err = generate.SliverSharedLibrary(config)
	case clientpb.ImplantConfig_SHELLCODE:
		fPath, err = generate.SliverShellcode(config)
	}

	filename := path.Base(fPath)
	filedata, err := ioutil.ReadFile(fPath)
	if err != nil {
		return nil, err
	}

	return &clientpb.Generate{
		File: &commonpb.File{
			Name: filename,
			Data: filedata,
		},
	}, err
}

// Regenerate - Regenerate a previously generated implant
func (rpc *Server) Regenerate(ctx context.Context, req *clientpb.RegenerateReq) (*clientpb.Generate, error) {

	config, err := generate.ImplantConfigByName(req.ImplantName)
	if err != nil {
		return nil, err
	}

	fileData, err := generate.ImplantFileByName(req.ImplantName)
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
func (rpc *Server) ImplantBuilds(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	configs, err := generate.ImplantConfigMap()
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
func (rpc *Server) Canaries(ctx context.Context, _ *commonpb.Empty) (*clientpb.Canaries, error) {
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

// ImplantProfiles - List profiles
func (rpc *Server) ImplantProfiles(ctx context.Context, _ *commonpb.Empty) (*clientpb.ImplantProfiles, error) {
	implantProfiles := &clientpb.ImplantProfiles{
		Profiles: []*clientpb.ImplantProfile{},
	}
	for name, config := range generate.Profiles() {
		implantProfiles.Profiles = append(implantProfiles.Profiles, &clientpb.ImplantProfile{
			Name:   name,
			Config: config.ToProtobuf(),
		})
	}
	return implantProfiles, nil
}

// SaveImplantProfile - Save a new profile
func (rpc *Server) SaveImplantProfile(ctx context.Context, profile *clientpb.ImplantProfile) (*clientpb.ImplantProfile, error) {
	config := generate.ImplantConfigFromProtobuf(profile.Config)
	profile.Name = path.Base(profile.Name)
	if 0 < len(profile.Name) && profile.Name != "." {
		rpcLog.Infof("Saving new profile with name %#v", profile.Name)
		err := generate.ProfileSave(profile.Name, config)
		if err != nil {
			return nil, err
		}
		return profile, nil
	}
	return nil, errors.New("Invalid profile name")
}

// ShellcodeRDI - Generates a RDI shellcode from a given DLL
func (rpc *Server) ShellcodeRDI(ctx context.Context, req *clientpb.ShellcodeRDIReq) (*clientpb.ShellcodeRDI, error) {
	shellcode, err := generate.ShellcodeRDIFromBytes(req.GetData(), req.GetFunctionName(), req.GetArguments())
	return &clientpb.ShellcodeRDI{Data: shellcode}, err
}
