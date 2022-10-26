package crack

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
	"sort"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// CrackCmd - GPU password cracking interface
func CrackCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	if !AreCrackersOnline(con) {
		PrintNoCrackstations(con)
		return
	}

	// do stuff
}

// CrackStationsCmd - Manage GPU cracking stations
func CrackStationsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	crackers, err := con.Rpc.Crackstations(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(crackers.Crackstations) == 0 {
		PrintNoCrackstations(con)
	} else {
		PrintCrackers(crackers.Crackstations, con)
	}
}

func PrintNoCrackstations(con *console.SliverConsoleClient) {
	con.PrintInfof("No crackstations connected to server\n")
}

func AreCrackersOnline(con *console.SliverConsoleClient) bool {
	crackers, err := con.Rpc.Crackstations(context.Background(), &commonpb.Empty{})
	if err != nil {
		return false
	}
	return len(crackers.Crackstations) > 0
}

func PrintCrackers(crackers []*clientpb.Crackstation, con *console.SliverConsoleClient) {
	sort.Slice(crackers, func(i, j int) bool {
		return crackers[i].Name < crackers[j].Name
	})
	for index, cracker := range crackers {
		printCracker(cracker, index, con)
		if index < len(crackers)-1 {
			con.Println()
		}
	}
}

func printCracker(cracker *clientpb.Crackstation, index int, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.Bold + fmt.Sprintf("[Crackstation %02d] %s (%s)", index+1, cracker.Name, cracker.OperatorName) + console.Normal + "\n")
	tw.AppendSeparator()
	tw.AppendRow(table.Row{console.Bold + "Hashcat Version" + console.Normal, cracker.HashcatVersion})
	if 0 < len(cracker.CUDA) {
		for _, cuda := range cracker.CUDA {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.Bold + "CUDA Device" + console.Normal, fmt.Sprintf(console.Bold+console.Green+"%s (%s)"+console.Normal, cuda.Name, cuda.Version)})
			tw.AppendRow(table.Row{console.Bold + "Memory" + console.Normal, fmt.Sprintf("%s free of %s", cuda.MemoryFree, cuda.MemoryTotal)})
			tw.AppendRow(table.Row{console.Bold + "Clock" + console.Normal, fmt.Sprintf("%d", cuda.Clock)})
			tw.AppendRow(table.Row{console.Bold + "Processors" + console.Normal, fmt.Sprintf("%d", cuda.Processors)})
		}
	}
	if 0 < len(cracker.Metal) {
		for _, metal := range cracker.Metal {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.Bold + "Metal Device" + console.Normal, fmt.Sprintf(console.Bold+console.Green+"%s (%s)"+console.Normal, metal.Name, metal.Version)})
			tw.AppendRow(table.Row{console.Bold + "Memory" + console.Normal, fmt.Sprintf("%s free of %s", metal.MemoryFree, metal.MemoryTotal)})
			tw.AppendRow(table.Row{console.Bold + "Clock" + console.Normal, fmt.Sprintf("%d", metal.Clock)})
			tw.AppendRow(table.Row{console.Bold + "Processors" + console.Normal, fmt.Sprintf("%d", metal.Processors)})
		}
	}
	if 0 < len(cracker.OpenCL) {
		for _, openCL := range cracker.OpenCL {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.Bold + "OpenCL Device" + console.Normal, fmt.Sprintf(console.Bold+console.Green+console.Bold+"%s (%s)"+console.Normal, openCL.Name, openCL.Version)})
			tw.AppendRow(table.Row{console.Bold + "Memory" + console.Normal, fmt.Sprintf("%s free of %s", openCL.MemoryFree, openCL.MemoryTotal)})
			tw.AppendRow(table.Row{console.Bold + "Clock" + console.Normal, fmt.Sprintf("%d", openCL.Clock)})
			tw.AppendRow(table.Row{console.Bold + "Processors" + console.Normal, fmt.Sprintf("%d", openCL.Processors)})
		}
	}
	con.Printf("%s\n", tw.Render())
}
