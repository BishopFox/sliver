package hosts

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
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// HostsIOCCmd - Remove a host from the database
func HostsIOCCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	host, err := SelectHost(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if 0 < len(host.IOCs) {
		con.Printf("%s\n", hostIOCsTable(host, con))
	} else {
		con.Println()
		con.PrintInfof("No IOCs tracked on host\n")
	}
}

func hostIOCsTable(host *clientpb.Host, con *console.SliverConsoleClient) string {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleBold)
	tw.AppendHeader(table.Row{"File Path", "SHA-256"})
	for _, ioc := range host.IOCs {
		tw.AppendRow(table.Row{
			ioc.Path,
			ioc.FileHash,
		})
	}
	return tw.Render()
}

func SelectHostIOC(host *clientpb.Host, con *console.SliverConsoleClient) (*clientpb.IOC, error) {
	if len(host.IOCs) == 0 {
		return nil, ErrNoIOCs
	}

	// Sort the keys because maps have a randomized order, these keys must be ordered for the selection
	// to work properly since we rely on the index of the user's selection to find the session in the map
	var keys []string
	iocMap := make(map[string]*clientpb.IOC)
	for _, ioc := range host.IOCs {
		keys = append(keys, ioc.ID)
		iocMap[ioc.ID] = ioc
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		ioc := iocMap[key]
		fmt.Fprintf(table, "%s\t%s\t\n",
			ioc.Path,
			ioc.FileHash,
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.Select{
		Message: "Select an IOC:",
		Options: options,
	}
	selected := ""
	survey.AskOne(prompt, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> host
	for index, option := range options {
		if option == selected {
			return iocMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}
