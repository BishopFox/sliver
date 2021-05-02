package completion

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
	"fmt"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// ImplantProfiles - Returns the names of generated implant profiles, with a description
func ImplantProfiles() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "implant profiles",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}
	profiles := getSliverProfiles()
	for _, profile := range *profiles {
		conf := profile.Config
		comp.Suggestions = append(comp.Suggestions, profile.Name)
		desc := fmt.Sprintf(" %s [%s/%s] -> %d C2s%s", conf.Format.String(), conf.GOOS, conf.GOARCH, len(conf.GetC2()), readline.RESET)
		comp.Descriptions[profile.Name] = readline.DIM + desc
	}

	return []*readline.CompletionGroup{comp}
}

func getSliverProfiles() *map[string]*clientpb.ImplantProfile {
	pbProfiles, err := transport.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil
	}
	profiles := &map[string]*clientpb.ImplantProfile{}
	for _, profile := range pbProfiles.Profiles {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}

// ImplantNames - Returns the names of generated implants, with a description
func ImplantNames() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "implants",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}

	builds, err := transport.RPC.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}

	for name, implant := range builds.Configs {
		comp.Suggestions = append(comp.Suggestions, name)
		desc := fmt.Sprintf(" %s [%s/%s] -> %d C2s%s", implant.Format.String(), implant.GOOS, implant.GOARCH, len(implant.GetC2()), readline.RESET)
		comp.Descriptions[name] = readline.DIM + desc
	}

	return []*readline.CompletionGroup{comp}
}
