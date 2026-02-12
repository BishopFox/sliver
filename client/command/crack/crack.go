package crack

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// CrackCmd - GPU password cracking interface
// CrackCmd - GPU 密码破解接口
func CrackCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if shouldRunCrack(cmd, args) {
		crackCmd, err := buildCrackCommand(cmd, args)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		timeoutSeconds, _ := cmd.Flags().GetInt64("timeout")
		ctx := context.Background()
		if timeoutSeconds > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
			defer cancel()
		}

		resp, err := con.Rpc.Crack(ctx, crackCmd)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if resp == nil || resp.Job == nil {
			con.PrintInfof("Crack request submitted\n")
			return
		}
		con.PrintInfof("Crack job %s created (status: %s)\n", resp.Job.ID, resp.Job.Status.String())
		if resp.Job.Err != "" {
			con.PrintErrorf("Crack job error: %s\n", resp.Job.Err)
		}
		return
	}

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
// CrackStationsCmd - Manage GPU 裂解站
func CrackStationsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	crackers, err := con.Rpc.Crackstations(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	showBenchmarks, _ := cmd.Flags().GetBool("show-benchmarks")
	if len(crackers.Crackstations) == 0 {
		PrintNoCrackstations(con)
	} else {
		PrintCrackers(crackers.Crackstations, con, showBenchmarks)
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

func PrintCrackers(crackers []*clientpb.Crackstation, con *console.SliverClient, showBenchmarks bool) {
	sort.Slice(crackers, func(i, j int) bool {
		return crackers[i].Name < crackers[j].Name
	})
	for index, cracker := range crackers {
		printCracker(cracker, index, con, showBenchmarks)
		if index < len(crackers)-1 {
			con.Println()
			con.Println()
		}
	}
}

func printCracker(cracker *clientpb.Crackstation, index int, con *console.SliverClient, showBenchmarks bool) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.StyleBoldOrange.Render(fmt.Sprintf(">>> Crackstation %02d - %s (%s)", index+1, cracker.Name, cracker.OperatorName)) + "\n")
	tw.AppendSeparator()
	tw.AppendRow(table.Row{console.StyleBold.Render("Operating System"), fmt.Sprintf("%s/%s", cracker.GOOS, cracker.GOARCH)})
	tw.AppendRow(table.Row{console.StyleBold.Render("Hashcat Version"), cracker.HashcatVersion})
	if 0 < len(cracker.CUDA) {
		for _, cuda := range cracker.CUDA {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.StyleBold.Render("CUDA Device"), console.StyleBoldGreen.Render(fmt.Sprintf("%s (%s)", cuda.Name, cuda.Version))})
			tw.AppendRow(table.Row{console.StyleBold.Render("Memory"), fmt.Sprintf("%s free of %s", cuda.MemoryFree, cuda.MemoryTotal)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Clock"), fmt.Sprintf("%d", cuda.Clock)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Processors"), fmt.Sprintf("%d", cuda.Processors)})
		}
	}
	if 0 < len(cracker.Metal) {
		for _, metal := range cracker.Metal {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.StyleBold.Render("Metal Device"), console.StyleBoldGreen.Render(fmt.Sprintf("%s (%s)", metal.Name, metal.Version))})
			tw.AppendRow(table.Row{console.StyleBold.Render("Memory"), fmt.Sprintf("%s free of %s", metal.MemoryFree, metal.MemoryTotal)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Clock"), fmt.Sprintf("%d", metal.Clock)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Processors"), fmt.Sprintf("%d", metal.Processors)})
		}
	}
	if 0 < len(cracker.OpenCL) {
		for _, openCL := range cracker.OpenCL {
			tw.AppendSeparator()
			tw.AppendRow(table.Row{console.StyleBold.Render("OpenCL Device"), console.StyleBoldGreen.Render(fmt.Sprintf("%s (%s)", openCL.Name, openCL.Version))})
			tw.AppendRow(table.Row{console.StyleBold.Render("Memory"), fmt.Sprintf("%s free of %s", openCL.MemoryFree, openCL.MemoryTotal)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Clock"), fmt.Sprintf("%d", openCL.Clock)})
			tw.AppendRow(table.Row{console.StyleBold.Render("Processors"), fmt.Sprintf("%d", openCL.Processors)})
		}
	}
	con.Printf("%s\n", tw.Render())
	if showBenchmarks {
		con.Println()
		printBenchmarks(cracker, con)
	}
}

func printBenchmarks(cracker *clientpb.Crackstation, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.StyleBold.Render("Benchmarks"))
	tw.SortBy([]table.SortBy{{Name: "Hash Type"}})
	tw.AppendHeader(table.Row{"Hash Type", "Rate"})
	if len(cracker.Benchmarks) == 0 {
		tw.AppendRow(table.Row{"No benchmarks reported", "-"})
	} else {
		for hashType, speed := range cracker.Benchmarks {
			name, ok := hashcatHashTypeName(hashType)
			if !ok {
				name = clientpb.HashType(hashType).String()
			}
			tw.AppendRow(table.Row{name, humanizeHashRate(speed)})
		}
	}
	con.Printf("%s\n", tw.Render())
}
