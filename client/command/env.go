package command

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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func getEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	name := ctx.Args.String("name")
	envInfo, err := rpc.GetEnv(context.Background(), &sliverpb.EnvReq{
		Name:    name,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	for _, envVar := range envInfo.Variables {
		fmt.Printf("%s=%s\n", envVar.Key, envVar.Value)
	}
}

func setEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	name := ctx.Args.String("name")
	value := ctx.Args.String("value")
	if name == "" || value == "" {
		fmt.Printf(Warn + "Usage: setenv KEY VALUE\n")
		return
	}

	envInfo, err := rpc.SetEnv(context.Background(), &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{
			Key:   name,
			Value: value,
		},
		Request: ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if envInfo.Response != nil && envInfo.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", envInfo.Response.Err)
		return
	}
	fmt.Printf(Info+"set %s to %s\n", name, value)
}

func unsetEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	name := ctx.Args.String("name")
	if name == "" {
		fmt.Printf(Warn + "Usage: setenv NAME\n")
		return
	}

	unsetResp, err := rpc.UnsetEnv(context.Background(), &sliverpb.UnsetEnvReq{
		Name:    name,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if unsetResp.Response != nil && unsetResp.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", unsetResp.Response.Err)
		return
	}
	fmt.Printf(Info+"Successfully unset %s\n", name)
}
