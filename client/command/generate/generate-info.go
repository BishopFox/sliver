package generate

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// GenerateInfoCmd - Display information about the Sliver server's compiler configuration.
// GenerateInfoCmd - Display 有关 Sliver 服务器编译器的信息 configuration.
func GenerateInfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to get compiler information: %s\n", err)
		return
	}
	con.Printf("%s %s/%s\n", console.StyleBold.Render("Server:"), compiler.GOOS, compiler.GOARCH)
	con.Println()
	con.Printf("%s\n", console.StyleBold.Render("Cross Compilers"))
	for _, cc := range compiler.CrossCompilers {
		con.Printf("%s/%s - %s\n", cc.TargetGOOS, cc.TargetGOARCH, cc.GetCCPath())
	}
	con.Println()
	con.Printf("%s\n", console.StyleBold.Render("Supported Targets"))
	for _, target := range compiler.Targets {
		con.Printf("%s/%s - %s\n", target.GOOS, target.GOARCH, nameOfOutputFormat(target.Format))
	}
	con.Println()
	con.Printf("%s\n", console.StyleBold.Render("Default Builds Only"))
	for _, target := range compiler.UnsupportedTargets {
		con.Printf("%s/%s - %s\n", target.GOOS, target.GOARCH, nameOfOutputFormat(target.Format))
	}
}
