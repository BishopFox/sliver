package server

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
	"sort"
	"strings"

	"github.com/evilsocket/islazy/tui"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Builds - List saved implant builds (binaries)
type Builds struct{}

// Execute - List saved implant builds (binaries)
func (b *Builds) Execute(args []string) (err error) {

	builds, err := transport.RPC.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return
	}

	if 0 < len(builds.Configs) {
		printImplantBuilds(builds.Configs)
	} else {
		fmt.Printf(util.Info + "No implant builds\n")
	}

	return
}

func printImplantBuilds(configs map[string]*clientpb.ImplantConfig) {

	// Sort keys
	var keys []string
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	table := util.NewTable(tui.Bold(tui.Yellow("Implant Builds")))
	headers := []string{"Name", "OS/Arch", "Format", "C2 Transports", "Debug/Obfsc/Evasion", "Limits", "Errs/Timeout"}
	headLen := []int{0, 0, 0, 15, 15, 7, 0}
	table.SetColumns(headers, headLen)

	// Populate the table with builds
	for _, k := range keys {
		config := configs[k]

		osArch := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)

		// Get a formated C2s string
		var c2s string
		if 0 < len(config.C2) {
			for index, c2 := range config.C2 {
				// for index, c2 := range config.C2[0:] {
				endpoint := fmt.Sprintf("[%d] %s \n", index+1, c2.URL)
				c2s += endpoint
			}
		}
		c2s = strings.TrimSuffix(c2s, "\n")

		// Security
		var debug, obfs, evas string
		if config.Debug {
			debug = tui.Yellow(" yes ")
		} else {
			debug = tui.Dim(" no ")
		}
		if config.ObfuscateSymbols {
			obfs = tui.Green(" yes ")
		} else {
			obfs = tui.Yellow(" no ")
		}
		if config.Evasion {
			evas = tui.Green("  yes ")
		} else {
			evas = tui.Yellow("  no ")
		}
		sec := fmt.Sprintf("%s %s %s", debug, obfs, evas)

		// Limits
		var user, domainJoin, dateTime, hostname, file string
		if config.LimitUsername != "" {
			user = tui.Bold("User: ") + config.LimitUsername + "\n"
		}
		if config.LimitHostname != "" {
			hostname = tui.Bold("Hostname: ") + config.LimitHostname + "\n"
		}
		if config.LimitFileExists != "" {
			file = tui.Bold("File: ") + config.LimitFileExists + "\n"
		}
		if config.LimitDatetime != "" {
			dateTime = tui.Bold("DateTime: ") + config.LimitDatetime + "\n"
		}
		if config.LimitDomainJoined == true {
			domainJoin = tui.Bold("Domain joined: ") + config.LimitDatetime + "\n"
		}
		limits := user + hostname + file + domainJoin + dateTime

		// Timeouts
		timeouts := fmt.Sprintf("%d / %ds", config.MaxConnectionErrors, config.ReconnectInterval)

		// Add row
		table.AppendRow([]string{k, osArch, config.Format.String(), c2s, sec, limits, timeouts})
	}

	// Print table
	table.Output()
}
