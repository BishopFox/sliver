package command

import (
	"fmt"
	"os"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sliver/server/encoders"
	"sliver/server/util"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func ls(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	data, _ := proto.Marshal(&sliverpb.DirListReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := rpc(&pb.Envelope{
		Type: consts.LsStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	dirList := &sliverpb.DirList{}
	err := proto.Unmarshal(resp.Data, dirList)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	printDirList(dirList)
}

func printDirList(dirList *sliverpb.DirList) {
	fmt.Printf("%s\n", dirList.Path)
	fmt.Printf("%s\n", strings.Repeat("=", len(dirList.Path)))

	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t<dir>\t\n", fileInfo.Name)
		} else {
			fmt.Fprintf(table, "%s\t%s\t\n", fileInfo.Name, util.ByteCountBinary(fileInfo.Size))
		}
	}
	table.Flush()
}

func rm(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file or directory name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.RmReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := rpc(&pb.Envelope{
		Type: consts.RmStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	rm := &sliverpb.Rm{}
	err := proto.Unmarshal(resp.Data, rm)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if rm.Success {
		fmt.Printf(Info+"%s\n", rm.Path)
	} else {
		fmt.Printf(Warn+"%s\n", rm.Err)
	}

}

func mkdir(ctx *grumble.Context, rpc RPCServer) {

	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: directory name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.MkdirReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := rpc(&pb.Envelope{
		Type: consts.MkdirStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	mkdir := &sliverpb.Mkdir{}
	err := proto.Unmarshal(resp.Data, mkdir)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if mkdir.Success {
		fmt.Printf(Info+"%s\n", mkdir.Path)
	} else {
		fmt.Printf(Warn+"%s\n", mkdir.Err)
	}
}

func cd(ctx *grumble.Context, rpc RPCServer) {

	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	data, _ := proto.Marshal(&sliverpb.CdReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := rpc(&pb.Envelope{
		Type: consts.CdStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	pwd := &sliverpb.Pwd{}
	err := proto.Unmarshal(resp.Data, pwd)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	fmt.Printf(Info+"%s\n", pwd.Path)
}

func pwd(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.PwdReq{
		SliverID: ActiveSliver.Sliver.ID,
	})
	resp := rpc(&pb.Envelope{
		Type: consts.PwdStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	pwd := &sliverpb.Pwd{}
	err := proto.Unmarshal(resp.Data, pwd)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	fmt.Printf(Info+"%s\n", pwd.Path)
}

func cat(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.DownloadReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := rpc(&pb.Envelope{
		Type: consts.DownloadStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	download := &sliverpb.Download{}
	proto.Unmarshal(resp.Data, download)
	if download.Encoder == "gzip" {
		download.Data, _ = encoders.GzipRead(download.Data)
	}
	fmt.Printf(string(download.Data))
}

func download(ctx *grumble.Context, rpc RPCServer) {

}

func upload(ctx *grumble.Context, rpc RPCServer) {

}
