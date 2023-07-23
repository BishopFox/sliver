package generate

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// GenerateInfoCmd - Display information about the Sliver server's compiler configuration.
func GenerateInfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to get compiler information: %s\n", err)
		return
	}
	con.Printf("%sServer:%s %s/%s\n", console.Bold, console.Normal, compiler.GOOS, compiler.GOARCH)
	con.Println()
	con.Printf("%sCross Compilers%s\n", console.Bold, console.Normal)
	for _, cc := range compiler.CrossCompilers {
		con.Printf("%s/%s - %s\n", cc.TargetGOOS, cc.TargetGOARCH, cc.GetCCPath())
	}
	con.Println()
	con.Printf("%sSupported Targets%s\n", console.Bold, console.Normal)
	for _, target := range compiler.Targets {
		con.Printf("%s/%s - %s\n", target.GOOS, target.GOARCH, nameOfOutputFormat(target.Format))
	}
	con.Println()
	con.Printf("%sDefault Builds Only%s\n", console.Bold, console.Normal)
	for _, target := range compiler.UnsupportedTargets {
		con.Printf("%s/%s - %s\n", target.GOOS, target.GOARCH, nameOfOutputFormat(target.Format))
	}
}
