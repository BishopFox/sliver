package creds

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
	"context"
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// CredsCmd - Add new credentials
func CredsAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	username := ctx.Flags.String("username")
	plaintext := ctx.Flags.String("plaintext")
	hash := ctx.Flags.String("hash")
	hashType := parseHashType(ctx.Flags.String("hash-type"))
	if plaintext == "" && hash == "" {
		con.PrintErrorf("Either a plaintext or a hash must be provided")
		return
	}
	if hashType == clientpb.HashType_INVALID {
		con.PrintErrorf("Invalid hash type '%s'", ctx.Flags.String("hash-type"))
		return
	}
	_, err := con.Rpc.CredsAdd(context.Background(), &clientpb.Credentials{
		Credentials: []*clientpb.Credential{
			{
				Username:  username,
				Plaintext: plaintext,
				Hash:      hash,
				HashType:  hashType,
			},
		},
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
}

func parseHashType(raw string) clientpb.HashType {
	hashInt, err := strconv.Atoi(raw)
	if err == nil {
		return clientpb.HashType(hashInt)
	}
	return clientpb.HashType_INVALID
}
