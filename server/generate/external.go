package generate

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

// SliverExternal - Generates the cryptographic keys for the implant but compiles no code
func SliverExternal(name string, config *clientpb.ImplantConfig) (*clientpb.ExternalImplantConfig, error) {
	config.IncludeMTLS = models.IsC2Enabled([]string{"mtls"}, config.C2)
	config.IncludeWG = models.IsC2Enabled([]string{"wg"}, config.C2)
	config.IncludeHTTP = models.IsC2Enabled([]string{"http", "https"}, config.C2)
	config.IncludeDNS = models.IsC2Enabled([]string{"dns"}, config.C2)
	config.IncludeNamePipe = models.IsC2Enabled([]string{"namedpipe"}, config.C2)
	config.IncludeTCP = models.IsC2Enabled([]string{"tcppivot"}, config.C2)

	build, err := GenerateConfig(name, config)
	if err != nil {
		return nil, err
	}
	config, err = db.SaveImplantConfig(config)
	if err != nil {
		return nil, err
	}

	build.ImplantConfigID = config.ID
	implantBuild, err := db.SaveImplantBuild(build)
	if err != nil {
		return nil, err
	}

	return &clientpb.ExternalImplantConfig{
		Config: config,
		Build:  implantBuild,
	}, nil
}
