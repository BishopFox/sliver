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
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// CrackCmd - GPU password cracking interface
func CrackCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	invocation, err := classifyCrackInvocation(cmd, args)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if invocation.mode == crackInvocationBackendInfo {
		ctx, cancel := crackCommandContext(cmd.Context(), cmd)
		defer cancel()
		crackers, err := connectedCrackstations(ctx, con.Rpc)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if len(crackers) == 0 {
			PrintNoCrackstations(con)
			return
		}
		printCrackersWithBackendInfoLevel(crackers, con, false, invocation.backendInfoLevel)
		return
	}

	if invocation.mode == crackInvocationJob || invocation.isSynchronousQuery() {
		crackCmd, err := buildCrackCommand(cmd, args)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		ctx, cancel := crackCommandContext(cmd.Context(), cmd)
		defer cancel()
		if err := resolveManagedCrackFileReferences(ctx, crackCmd, func(ctx context.Context, request *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
			return con.Rpc.CrackFilesList(ctx, request)
		}); err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		resp, err := con.Rpc.Crack(ctx, crackCmd)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if invocation.isSynchronousQuery() {
			value, stderr, err := crackQueryResponse(resp, invocation)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			printCrackQueryStderr(con, stderr)
			printCrackQueryValue(con, value)
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

func canonicalCrackKeyspace(response *clientpb.CrackResponse) (string, error) {
	return crackQueryResponseValue(response, crackInvocation{mode: crackInvocationKeyspace})
}

func (invocation crackInvocation) isSynchronousQuery() bool {
	switch invocation.mode {
	case crackInvocationKeyspace, crackInvocationTotalCandidates, crackInvocationLookup, crackInvocationIdentify, crackInvocationHashInfo:
		return true
	default:
		return false
	}
}

func (invocation crackInvocation) queryMode() clientpb.CrackQueryMode {
	switch invocation.mode {
	case crackInvocationKeyspace:
		return clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE
	case crackInvocationTotalCandidates:
		return clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES
	case crackInvocationLookup:
		return clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP
	case crackInvocationIdentify:
		return clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY
	case crackInvocationHashInfo:
		return clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO
	default:
		return clientpb.CrackQueryMode_CRACK_QUERY_UNSPECIFIED
	}
}

func crackQueryResponseValue(response *clientpb.CrackResponse, invocation crackInvocation) (string, error) {
	value, _, err := crackQueryResponse(response, invocation)
	return value, err
}

func crackQueryResponse(response *clientpb.CrackResponse, invocation crackInvocation) (string, string, error) {
	if response == nil {
		return "", "", fmt.Errorf("server returned an empty %s response", crackInvocationName(invocation.mode))
	}
	expectedMode := invocation.queryMode()
	if expectedMode == clientpb.CrackQueryMode_CRACK_QUERY_UNSPECIFIED {
		return "", "", fmt.Errorf("invalid synchronous crack query mode")
	}
	query := response.GetQuery()
	if query == nil {
		// Servers predating typed query results returned only Keyspace. Do not use
		// that legacy field for a selected station because it cannot prove which
		// station produced the result.
		if invocation.mode == crackInvocationKeyspace && invocation.crackstation == "" {
			value, err := canonicalCrackDecimal(response.GetKeyspace(), "keyspace")
			return value, "", err
		}
		return "", "", fmt.Errorf("server returned an empty %s query result", crackInvocationName(invocation.mode))
	}
	if query.GetMode() != expectedMode {
		return "", "", fmt.Errorf("server returned %s for %s query", query.GetMode().String(), crackInvocationName(invocation.mode))
	}
	value := query.GetValue()
	switch invocation.mode {
	case crackInvocationKeyspace, crackInvocationTotalCandidates:
		value, err := canonicalCrackDecimal(value, crackInvocationName(invocation.mode))
		return value, query.GetStderr(), err
	default:
		if strings.TrimSpace(value) == "" {
			return "", "", fmt.Errorf("server returned an empty %s query result", crackInvocationName(invocation.mode))
		}
		return value, query.GetStderr(), nil
	}
}

func canonicalCrackDecimal(value string, name string) (string, error) {
	raw := strings.TrimSpace(value)
	keyspace, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return "", fmt.Errorf("server returned invalid %s %q", name, value)
	}
	return strconv.FormatUint(keyspace, 10), nil
}

func printCrackQueryValue(con *console.SliverClient, value string) {
	con.Printf("%s", value)
	if !strings.HasSuffix(value, "\n") {
		con.Printf("\n")
	}
}

func printCrackQueryStderr(con *console.SliverClient, value string) {
	if value == "" {
		return
	}
	if !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	con.PrintWarnf("%s", value)
}

// CrackStationsCmd - Manage GPU cracking stations
func CrackStationsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	ctx, cancel := crackCommandContext(cmd.Context(), cmd)
	defer cancel()
	crackers, err := connectedCrackstations(ctx, con.Rpc)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	showBenchmarks, _ := cmd.Flags().GetBool("show-benchmarks")
	if len(crackers) == 0 {
		PrintNoCrackstations(con)
	} else {
		PrintCrackers(crackers, con, showBenchmarks)
	}
}

func connectedCrackstations(ctx context.Context, rpc rpcpb.SliverRPCClient) ([]*clientpb.Crackstation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	response, err := rpc.Crackstations(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	crackers := make([]*clientpb.Crackstation, 0, len(response.Crackstations))
	for _, cracker := range response.Crackstations {
		if cracker != nil {
			crackers = append(crackers, cracker)
		}
	}
	return crackers, nil
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
	printCrackersWithBackendInfoLevel(crackers, con, showBenchmarks, 1)
}

func printCrackersWithBackendInfoLevel(crackers []*clientpb.Crackstation, con *console.SliverClient, showBenchmarks bool, backendInfoLevel uint32) {
	ordered := make([]*clientpb.Crackstation, 0, len(crackers))
	for _, cracker := range crackers {
		if cracker != nil {
			ordered = append(ordered, cracker)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Name == ordered[j].Name {
			return ordered[i].HostUUID < ordered[j].HostUUID
		}
		return ordered[i].Name < ordered[j].Name
	})
	for index, cracker := range ordered {
		printCracker(cracker, index, con, showBenchmarks, backendInfoLevel)
		if index < len(ordered)-1 {
			con.Println()
			con.Println()
		}
	}
}

func printCracker(cracker *clientpb.Crackstation, index int, con *console.SliverClient, showBenchmarks bool, backendInfoLevel uint32) {
	con.Printf("%s\n", renderCracker(cracker, index, settings.GetTableStyle(con), backendInfoLevel))
	if showBenchmarks {
		con.Println()
		printBenchmarks(cracker, con)
	}
}

func renderCracker(cracker *clientpb.Crackstation, index int, style table.Style, backendInfoLevel uint32) string {
	if cracker == nil {
		return ""
	}
	tw := table.NewWriter()
	tw.SetStyle(style)
	tw.SetTitle(console.StyleBoldOrange.Render(fmt.Sprintf(">>> Crackstation %02d - %s (%s)", index+1, safeCrackCell(cracker.Name), safeCrackCell(cracker.OperatorName))) + "\n")
	tw.AppendSeparator()
	tw.AppendRow(table.Row{console.StyleBold.Render("Operating System"), fmt.Sprintf("%s/%s", valueOrDash(cracker.GOOS), valueOrDash(cracker.GOARCH))})
	tw.AppendRow(table.Row{console.StyleBold.Render("Hashcat Version"), valueOrDash(cracker.HashcatVersion)})
	if backendInfoLevel >= 2 {
		appendCrackstationDetailRows(tw, cracker)
	}
	for _, cuda := range cracker.CUDA {
		appendCUDABackendRows(tw, cuda, backendInfoLevel)
	}
	for _, hip := range cracker.HIP {
		appendHIPBackendRows(tw, hip, backendInfoLevel)
	}
	for _, metal := range cracker.Metal {
		appendMetalBackendRows(tw, metal, backendInfoLevel)
	}
	for _, openCL := range cracker.OpenCL {
		appendOpenCLBackendRows(tw, openCL, backendInfoLevel)
	}
	if !hasReportedBackendDevice(cracker) {
		tw.AppendSeparator()
		tw.AppendRow(table.Row{console.StyleBold.Render("Backend Devices"), "No backend devices reported"})
	}
	return tw.Render()
}

func formatCUDADevice(device *clientpb.CUDABackendInfo) string {
	return formatBackendDevice(device.GetName(), device.GetCUDAVersion(), device.GetVersion())
}

func formatHIPDevice(device *clientpb.HIPBackendInfo) string {
	return formatBackendDevice(device.GetName(), device.GetHIPVersion(), device.GetVersion())
}

func formatMetalDevice(device *clientpb.MetalBackendInfo) string {
	return formatBackendDevice(device.GetName(), device.GetMetalVersion(), device.GetVersion())
}

func formatOpenCLDevice(device *clientpb.OpenCLBackendInfo) string {
	return formatBackendDevice(device.GetName(), device.GetOpenCLVersion(), device.GetVersion())
}

func formatBackendDevice(name string, versions ...string) string {
	for _, version := range versions {
		if version != "" {
			return fmt.Sprintf("%s (%s)", name, version)
		}
	}
	return name
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
