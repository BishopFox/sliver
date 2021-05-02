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

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetEnv - Get the session's environment variables
type GetEnv struct {
	Positional struct {
		Vars []string `description:"environment variable name"`
	} `positional-args:"yes"`
}

// Execute - Get the session's environment variables
func (e *GetEnv) Execute(args []string) (err error) {

	// Get all variables if no arguments given
	if len(e.Positional.Vars) == 0 {
		e.Positional.Vars = []string{""}
	}

	for _, name := range e.Positional.Vars {
		envInfo, err := transport.RPC.GetEnv(context.Background(), &sliverpb.EnvReq{
			Name:    name,
			Request: core.ActiveSessionRequest(),
		})

		if err != nil {
			fmt.Printf(Error+"Error: %v", err)
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
		Key   string `description:"environment variable name" required:"1"`
		Value string `description:"environment variable value" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Set an environment variable on the target host
func (e *SetEnv) Execute(args []string) (err error) {

	envInfo, err := transport.RPC.SetEnv(context.Background(), &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{
			Key:   e.Positional.Key,
			Value: e.Positional.Value,
		},
		Request: core.ActiveSessionRequest(),
	})
	if err != nil {
		fmt.Printf(Warning+"Error: %v", err)
		return
	}
	if envInfo.Response != nil && envInfo.Response.Err != "" {
		fmt.Printf(Warning+"Error: %s", envInfo.Response.Err)
		return
	}
	fmt.Printf(Info+"set %s to %s\n", e.Positional.Key, e.Positional.Value)

	return
}
