package commands

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

	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// NewProfile - Configure and save a new implant profile.
type NewProfile struct {
	StageOptions // This commands works the same as generate, and needs full options.
}

// Execute - Configure and save a new implant profile.
func (p *NewProfile) Execute(args []string) (err error) {

	name := p.CoreOptions.Profile
	if name == "" {
		fmt.Printf(util.Error + "Invalid profile name\n")
		return
	}

	config := parseCompileFlags(p.StageOptions)
	if config == nil {
		return
	}

	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := connection.RPC.SaveImplantProfile(context.Background(), profile)

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"Saved new profile %s\n", resp.Name)
	}

	return
}

func getSliverProfiles() *map[string]*clientpb.ImplantProfile {
	pbProfiles, err := connection.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"Error %s", err)
		return nil
	}
	profiles := &map[string]*clientpb.ImplantProfile{}
	for _, profile := range pbProfiles.Profiles {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}
