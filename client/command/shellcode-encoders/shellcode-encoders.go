package shellcodeencoders

import (
	"sort"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// ShellcodeEncodersCmd - Display supported shellcode encoders and architectures.
func ShellcodeEncodersCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()

	encoderMap, err := con.Rpc.ShellcodeEncoderMap(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	DisplayShellcodeEncoders(encoderMap, con)
}

// DisplayShellcodeEncoders - Display shellcode encoder map from server.
func DisplayShellcodeEncoders(encoderMap *clientpb.ShellcodeEncoderMap, con *console.SliverClient) {
	if encoderMap == nil || len(encoderMap.GetEncoders()) == 0 {
		con.PrintInfof("No shellcode encoders available\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Arch", "Encoder", "Description"})
	tw.SortBy([]table.SortBy{
		{Name: "Arch", Mode: table.Asc},
		{Name: "Encoder", Mode: table.Asc},
	})

	arches := make([]string, 0, len(encoderMap.GetEncoders()))
	for arch := range encoderMap.GetEncoders() {
		arches = append(arches, arch)
	}
	sort.Strings(arches)

	for _, arch := range arches {
		archMap := encoderMap.GetEncoders()[arch]
		if archMap == nil {
			continue
		}
		names := make([]string, 0, len(archMap.GetEncoders()))
		for name := range archMap.GetEncoders() {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			desc := archMap.GetDescriptions()[name]
			tw.AppendRow(table.Row{arch, name, desc})
		}
	}

	con.Println(tw.Render())
}
