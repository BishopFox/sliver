package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func websites(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if len(ctx.Args) < 1 {
		listWebsites(ctx, rpc)
		return
	}
	if ctx.Flags.String("website") == "" {
		fmt.Println(Warn + "Subcommand must specify a --website")
		return
	}
	switch strings.ToLower(ctx.Args[0]) {
	case "ls":
		listWebsiteContent(ctx, rpc)
	case "add":
		addWebsiteContent(ctx, rpc)
	case "rm":
		removeWebsiteContent(ctx, rpc)
	default:
		fmt.Println(Warn + "Invalid subcommand, see 'help websites'")
	}
}

func listWebsites(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	websites, err := rpc.Websites(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Failed to list websites %s", err)
		return
	}
	if len(websites.Websites) < 1 {
		fmt.Printf(Info + "No websites\n")
		return
	}
	fmt.Println("Websites")
	fmt.Println(strings.Repeat("=", len("Websites")))
	for _, site := range websites.Websites {
		fmt.Printf("%s%s%s - %d page(s)\n", bold, site.Name, normal, len(site.Contents))
	}
}

func listWebsiteContent(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("website")
	website, err := rpc.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		fmt.Printf(Warn+"Failed to list website content %s", err)
		return
	}
	if 0 < len(website.Contents) {
		displayWebsite(website)
	} else {
		fmt.Printf(Info+"No content for '%s'", name)
	}
}

func addWebsiteContent(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	websiteName := ctx.Flags.String("website")
	webPath := ctx.Flags.String("web-path")
	contentPath := ctx.Flags.String("content")
	if contentPath == "" {
		fmt.Println(Warn + "Must specify some --content")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType := ctx.Flags.String("content-type")
	recursive := ctx.Flags.Bool("recursive")

	fileInfo, err := os.Stat(contentPath)
	if err != nil {
		fmt.Printf(Warn+"Error adding content %s\n", err)
		return
	}

	addWeb := &clientpb.WebsiteAddContent{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}

	if fileInfo.IsDir() {
		if !recursive && !confirmAddDirectory() {
			return
		}
		webAddDirectory(addWeb, webPath, contentPath)
	} else {
		webAddFile(addWeb, webPath, contentType, contentPath)
	}

	web, err := rpc.WebsiteAddContent(context.Background(), addWeb)
	if err != nil {
		fmt.Printf(Warn+"%s", err)
		return
	}
	displayWebsite(web)
}

func removeWebsiteContent(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("website")
	webPath := ctx.Flags.String("web-path")
	recursive := ctx.Flags.Bool("recursive")
	website, err := rpc.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
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
	web, err := rpc.WebsiteRemoveContent(context.Background(), rmWebContent)
	if err != nil {
		fmt.Printf(Warn+"Failed to remove content %s", err)
		return
	}
	displayWebsite(web)
}

func displayWebsite(web *clientpb.Website) {
	fmt.Println(Info + web.Name)
	fmt.Println()
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Path\tContent-type\tSize\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Path")),
		strings.Repeat("=", len("Content-type")),
		strings.Repeat("=", len("Size")))
	sortedContents := []*clientpb.WebContent{}
	for _, content := range web.Contents {
		sortedContents = append(sortedContents, content)
	}
	sort.SliceStable(sortedContents, func(i, j int) bool {
		return sortedContents[i].Path < sortedContents[j].Path
	})
	for _, content := range sortedContents {
		fmt.Fprintf(table, "%s\t%s\t%d\t\n", content.Path, content.ContentType, content.Size)
	}
	table.Flush()
}

func webAddDirectory(web *clientpb.WebsiteAddContent, webpath string, contentPath string) {
	fullLocalPath, _ := filepath.Abs(contentPath)
	filepath.Walk(contentPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// localPath is the full absolute path to the file, so we cut it down
			fullWebpath := path.Join(webpath, localPath[len(fullLocalPath):])
			webAddFile(web, fullWebpath, "", localPath)
		}
		return nil
	})
}

func webAddFile(web *clientpb.WebsiteAddContent, webpath string, contentType string, contentPath string) error {

	fileInfo, err := os.Stat(contentPath)
	if os.IsNotExist(err) {
		return err // contentPath does not exist
	}
	if fileInfo.IsDir() {
		return errors.New("file content path is directory")
	}

	file, err := os.Open(contentPath)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if contentType == "" {
		contentType = sniffContentType(file)
	}

	web.Contents[webpath] = &clientpb.WebContent{
		Path:        webpath,
		ContentType: contentType,
		Content:     data,
	}
	return nil
}

func confirmAddDirectory() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "Recursively add entire directory?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

func sniffContentType(out *os.File) string {
	out.Seek(0, io.SeekStart)
	buffer := make([]byte, fileSampleSize)
	_, err := out.Read(buffer)
	if err != nil {
		return defaultMimeType
	}
	contentType := http.DetectContentType(buffer)
	return contentType
}
