package registry

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// RegCreateKeyCmd - Create a new Windows registry key
func RegCreateKeyCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.Get()
	if session == nil {
		return
	}

	hostname := ctx.Flags.String("hostname")
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := ctx.Args.String("registry-path")
	if regPath == "" {
		con.PrintErrorf("You must provide a path\n")
		return
	}
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	createKeyResp, err := con.Rpc.RegistryCreateKey(context.Background(), &sliverpb.RegistryCreateKeyReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if createKeyResp.Response != nil && createKeyResp.Response.Err != "" {
		con.PrintErrorf("%s", createKeyResp.Response.Err)
		return
	}
	con.PrintInfof("Key created at %s\\%s", regPath, key)
}
