package websites

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// WebsitesUpdateContentCmd - Update metadata about static website content
func WebsitesUpdateContentCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	websiteName := ctx.Flags.String("website")
	if websiteName == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	webPath := ctx.Flags.String("web-path")
	if webPath == "" {
		con.PrintErrorf("Must specify a web path via --web-path, see --help\n")
		return
	}
	contentType := ctx.Flags.String("content-type")
	if contentType == "" {
		con.PrintErrorf("Must specify a new --content-type, see --help\n")
		return
	}

	updateWeb := &clientpb.WebsiteAddContent{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}
	updateWeb.Contents[webPath] = &clientpb.WebContent{
		ContentType: contentType,
	}

	web, err := con.Rpc.WebsiteUpdateContent(context.Background(), updateWeb)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	PrintWebsite(web, con)
}
