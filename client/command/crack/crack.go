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

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// CrackCmd - GPU password cracking interface
func CrackCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if !AreCrackersOnline(con) {
		PrintNoCrackstations(con)
	} else {
		crackers, err := con.Rpc.Crackstations(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("%d crackstation(s) connected to server\n", len(crackers.Crackstations))
	}
	crackFiles, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(crackFiles.Files) == 0 {
		con.PrintInfof("No crack files uploaded to server\n")
	} else {
		con.Println()
		PrintCrackFilesByType(crackFiles, con)
	}
}

// CrackStationsCmd - Manage GPU cracking stations
func CrackStationsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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

func PrintNoCrackstations(con *console.SliverClient) {
	con.PrintInfof("No crackstations connected to server\n")
}

func AreCrackersOnline(con *console.SliverClient) bool {
	crackers, err := con.Rpc.Crackstations(context.Background(), &commonpb.Empty{})
	if err != nil {
		return false
	}
	return len(crackers.Crackstations) > 0
}

func PrintCrackers(crackers []*clientpb.Crackstation, con *console.SliverClient) {
	sort.Slice(crackers, func(i, j int) bool {
		return crackers[i].Name < crackers[j].Name
	})
	for index, cracker := range crackers {
		printCracker(cracker, index, con)
		if index < len(crackers)-1 {
			con.Println()
			con.Println()
		}
	}
}

func printCracker(cracker *clientpb.Crackstation, index int, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.Bold + console.Orange + fmt.Sprintf(">>> Crackstation %02d - %s (%s)", index+1, cracker.Name, cracker.OperatorName) + console.Normal + "\n")
	tw.AppendSeparator()
	tw.AppendRow(table.Row{console.Bold + "Operating System" + console.Normal, fmt.Sprintf("%s/%s", cracker.GOOS, cracker.GOARCH)})
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
	con.Println()
	printBenchmarks(cracker, con)
}

func printBenchmarks(cracker *clientpb.Crackstation, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.Bold + "Benchmarks" + console.Normal)
	tw.SortBy([]table.SortBy{{Name: "Hash Type"}})
	tw.AppendHeader(table.Row{"Hash Type", "Rate (H/s)"})
	for hashType, speed := range cracker.Benchmarks {
		tw.AppendRow(table.Row{clientpb.HashType(hashType), fmt.Sprintf("%d", speed)})
	}
	con.Printf("%s\n", tw.Render())
}
