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
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetEnv - Get the session's environment variables
type GetEnv struct {
	Positional struct {
		Vars []string `description:"environment variable name" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Get the session's environment variables
func (e *GetEnv) Execute(args []string) (err error) {
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

// SetEnv - Set an environment variable on the target host
type SetEnv struct {
	Positional struct {
		Key string `description:"environment variable name" required:"1"`
		Value    string `description:"environment variable value" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Set an environment variable on the target host
func (e *SetEnv) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	envInfo, err := transport.RPC.SetEnv(context.Background(), &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{
			Key:   e.Positional.Key,
			Value: e.Positional.Value,
		},
		Request: cctx.Request(session),
	})
	if err != nil {
		fmt.Printf(util.Warn+"Error: %v", err)
		return
	}
	if envInfo.Response != nil && envInfo.Response.Err != "" {
		fmt.Printf(util.Warn+"Error: %s", envInfo.Response.Err)
		return
	}
	fmt.Printf(util.Info+"set %s to %s\n", e.Positional.Key, e.Positional.Value)

	return
}
