package websites

import (
	"context"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// WebsitesRmContent - Remove static content from a website
func WebsitesRmContent(ctx *grumble.Context, con *console.SliverConsoleClient) {
	name := ctx.Flags.String("website")
	webPath := ctx.Flags.String("web-path")
	recursive := ctx.Flags.Bool("recursive")

	if name == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	if webPath == "" {
		con.PrintErrorf("Must specify a web path via --web-path, see --help\n")
		return
	}

	website, err := con.Rpc.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	rmWebContent := &clientpb.WebsiteRemoveContent{
		Name:  name,
		Paths: []string{},
	}
	if recursive {
		for contentPath := range website.Contents {
			if strings.HasPrefix(contentPath, webPath) {
				rmWebContent.Paths = append(rmWebContent.Paths, contentPath)
			}
		}
	} else {
		rmWebContent.Paths = append(rmWebContent.Paths, webPath)
	}
	web, err := con.Rpc.WebsiteRemoveContent(context.Background(), rmWebContent)
	if err != nil {
		con.PrintErrorf("Failed to remove content %s", err)
		return
	}
	PrintWebsite(web, con)
}
