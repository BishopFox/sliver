package cdp

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
	"fmt"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

var (
	curses = &sync.Map{}
)

// CursedChromeCmd - Execute a .NET assembly in-memory
func CursedChromeCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	chromeProcess, err := getChromeProcess(session, ctx, con)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	fmt.Printf("Found chrome process: %d (ppid: %d)\n", chromeProcess.GetPid(), chromeProcess.GetPpid())

	// overlord.ExecuteJS("", "", "", "")
}

// Check process: 'Google Chrome' (67807)
// 'Google Chrome' does not have suffix 'Google Chrome'

func isChromeProcess(executable string) bool {
	var chromeProcessNames = []string{
		// "chrome",
		// "chrome.exe",
		"Google Chrome",
	}
	for _, suffix := range chromeProcessNames {
		if strings.HasSuffix(executable, suffix) {
			return true
		}
	}
	return false
}

func getChromeProcess(session *clientpb.Session, ctx *grumble.Context, con *console.SliverConsoleClient) (*commonpb.Process, error) {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return nil, err
	}
	for _, process := range ps.Processes {
		executable := process.GetExecutable()
		if isChromeProcess(executable) {
			return process, nil
		}
	}
	return nil, nil
}
