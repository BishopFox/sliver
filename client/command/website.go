package command

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/golang/protobuf/proto"

	"github.com/desertbit/grumble"
)

func websites(ctx *grumble.Context, rpc RPCServer) {
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

func listWebsites(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)

	if len(websites.Sites) < 1 {
		fmt.Printf(Info + "No websites\n")
		return
	}

	for index, site := range websites.Sites {
		fmt.Printf("%d. %s\n", index+1, site.Name)
	}
}

func listWebsiteContent(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)

}

func addWebsiteContent(ctx *grumble.Context, rpc RPCServer) {

	websiteName := ctx.Flags.String("website")
	webPath := ctx.Flags.String("web-path")
	contentPath := ctx.Flags.String("content")
	if contentPath == "" {
		fmt.Printf(Warn + "Must specify some --content")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType := ctx.Flags.String("content-type")
	recursive := ctx.Flags.Bool("recursive")

	addWebsite := &clientpb.Website{Name: websiteName}

	fileInfo, _ := os.Stat(contentPath)
	if fileInfo.IsDir() {
		if !recursive && !confirmAddDirectory() {
			return
		}
		webAddDirectory(addWebsite, webPath, contentPath)
	} else {
		webAddFile(addWebsite, webPath, contentType, contentPath)
	}

	data, err := proto.Marshal(addWebsite)
	if err != nil {
		fmt.Printf(Warn+"Failed to marshal data %s\n", err)
		return
	}
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteAddContent,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

}

func removeWebsiteContent(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteRemoveContent,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

}

func displayWebsite(web *clientpb.Website) {

}

func webAddDirectory(web *clientpb.Website, path string, contentPath string) {

}

func webAddFile(web *clientpb.Website, path string, contentType string, contentPath string) error {

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

	web.Content[path] = &clientpb.WebContent{
		Path:        path,
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
