package windows

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
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RunAs - Run a new process in the context of the designated user (Windows Only)
type RunAs struct {
	Positional struct {
		Args []string `description:"(optional) arguments to pass to --process when executing"`
	} `positional-args:"yes"`

	Options struct {
		Username   string `long:"username" short:"u" description:"user to impersonate" default:"NT AUTHORITY\\SYSTEM"`
		RemotePath string `long:"process" short:"p" description:"process to start" required:"yes"`
	} `group:"run-as options"`
}

// Execute - Run a new process in the context of the designated user (Windows Only)
func (ra *RunAs) Execute(args []string) (err error) {

	username := ra.Options.Username
	process := ra.Options.RemotePath
	arguments := strings.Join(ra.Positional.Args, " ")

	runAsResp, err := transport.RPC.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     core.ActiveSessionRequest(),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v\n", err)
		return
	}

	if runAsResp.GetResponse().GetErr() != "" {
		fmt.Printf(Error+"Error: %s\n", runAsResp.GetResponse().GetErr())
		return
	}

	fmt.Printf(Info+"Sucessfully ran %s %s on %s\n", process, arguments, core.ActiveSession.GetName())

	return
}

// Impersonate - Impersonate a logged in user
type Impersonate struct {
	Positional struct {
		Username string `description:"user to impersonate" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Impersonate a logged in user
func (i *Impersonate) Execute(args []string) (err error) {

	username := i.Positional.Username
	impResp, err := transport.RPC.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
		Request:  core.ActiveSessionRequest(),
		Username: username,
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}
	if impResp.GetResponse().GetErr() != "" {
		fmt.Printf(Error+"Error: %s\n", impResp.GetResponse().GetErr())
		return
	}
	fmt.Printf(Info+"Successfully impersonated %s\n", username)

	return
}

// Rev2Self - Revert to self: lose stolen Windows token
type Rev2Self struct{}

// Execute - Revert to self: lose stolen Windows token
func (rs *Rev2Self) Execute(args []string) (err error) {

	_, err = transport.RPC.RevToSelf(context.Background(), &sliverpb.RevToSelfReq{
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v\n", err)
		return
	}
	fmt.Printf(Info + "Back to self...\n")
	return nil
}

// GetSystem - Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user
type GetSystem struct {
	Options struct {
		RemotePath string `long:"process" short:"p" description:"SYSTEM process to inject into" default:"spoolsv.exe"`
	} `group:"getsystem options"`
}

// Execute - Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user
func (gs *GetSystem) Execute(args []string) (err error) {

	process := gs.Options.RemotePath
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	go spin.Until("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getsystemResp, err := transport.RPC.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        core.ActiveSessionRequest(),
		Config:         config,
		HostingProcess: process,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Error+"Error: %v\n", err)
		return
	}
	if getsystemResp.GetResponse().GetErr() != "" {
		fmt.Printf(Error+"Error: %s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n" + Info + "A new SYSTEM session should pop soon...\n")

	return
}

// MakeToken - Create a new Logon Session with the specified credentials
type MakeToken struct {
	Options struct {
		Username string `long:"username" short:"u" description:"user to impersonate" required:"yes"`
		Password string `long:"password" short:"p" description:"password of user to impersonate" required:"yes"`
		Domain   string `long:"domain" short:"d" description:"domain of the user to impersonate"`
	} `group:"token options"`
}

// Execute - Create a new Logon Session with the specified credentials
func (mt *MakeToken) Execute(args []string) (err error) {

	username := mt.Options.Username
	password := mt.Options.Password
	domain := mt.Options.Domain

	if username == "" || password == "" {
		fmt.Printf(Error + "You must provide a username and password\n")
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Creating new logon session ...", ctrl)

	makeToken, err := transport.RPC.MakeToken(context.Background(), &sliverpb.MakeTokenReq{
		Request:  core.ActiveSessionRequest(),
		Username: username,
		Domain:   domain,
		Password: password,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Error+"Error: %v\n", err)
		return
	}

	if makeToken.GetResponse().GetErr() != "" {

		fmt.Printf(Error+"Error: %s\n", makeToken.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n"+Info+"Successfully impersonated %s\\%s. Use `rev2self` to revert to your previous token.\n", domain, username)
	return
}
