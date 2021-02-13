package sliver

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

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// SessionEnv - Get the session's environment variables
type SessionEnv struct {
	Positional struct {
		Vars []string `description:"environment variable name" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Get the session's environment variables
func (e *SessionEnv) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	for _, name := range e.Positional.Vars {
		envInfo, err := transport.RPC.GetEnv(context.Background(), &sliverpb.EnvReq{
			Name:    name,
			Request: cctx.Request(session),
		})

		if err != nil {
			fmt.Printf(util.Error+"Error: %v", err)
			continue
		}

		for _, envVar := range envInfo.Variables {
			fmt.Printf(" %s=%s\n", envVar.Key, envVar.Value)
		}
	}
	return
}
