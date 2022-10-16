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
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db/models"
)

// SliverExternal - Generates the cryptographic keys for the implant but compiles no code
func SliverExternal(name string, config *models.ImplantConfig) (*clientpb.ExternalImplantConfig, error) {
	err := GenerateConfig(name, config, true)
	if err != nil {
		return nil, err
	}
	otpSecret, err := cryptography.TOTPServerSecret()
	if err != nil {
		return nil, err
	}
	return &clientpb.ExternalImplantConfig{
		Config:    config.ToProtobuf(),
		OTPSecret: otpSecret,
	}, nil
}
