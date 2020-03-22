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
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"

	"github.com/desertbit/grumble"
)

func ls(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	ls, err := rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: ActiveSession.Request(),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		printDirList(ls)
	}
}

func printDirList(dirList *sliverpb.Ls) {
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

func rm(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file or directory name\n")
		return
	}

	rm, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
		Request: ActiveSession.Request(),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", rm.Path)
	}
}

func mkdir(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: directory name\n")
		return
	}

	mkdir, err := rpc.Mkdir(context.Background(), &sliverpb.MkdirReq{
		Request: ActiveSession.Request(),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", mkdir.Path)
	}
}

func cd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	pwd, err := rpc.Cd(context.Background(), &sliverpb.CdReq{
		Request: ActiveSession.Request(),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", pwd.Path)
	}
}

func pwd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	pwd, err := rpc.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: ActiveSession.Request(),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", pwd.Path)
	}
}

// func cat(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	if ActiveSession.Session == nil {
// 		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
// 		return
// 	}

// 	if len(ctx.Args) == 0 {
// 		fmt.Printf(Warn + "Missing parameter: file name\n")
// 		return
// 	}

// 	data, _ := proto.Marshal(&sliverpb.DownloadReq{
// 		SliverID: ActiveSession.Session.ID,
// 		Path:     ctx.Args[0],
// 	})
// 	resp := <-rpc(&sliverpb.Envelope{
// 		Type: sliverpb.MsgDownloadReq,
// 		Data: data,
// 	}, defaultTimeout)
// 	if resp.Err != "" {
// 		fmt.Printf(Warn+"Error: %s", resp.Err)
// 		return
// 	}

// 	download := &sliverpb.Download{}
// 	proto.Unmarshal(resp.Data, download)
// 	if download.Encoder == "gzip" {
// 		download.Data, _ = new(util.Gzip).Decode(download.Data)
// 	}
// 	fmt.Printf(string(download.Data))
// }

// func download(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	if ActiveSession.Session == nil {
// 		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
// 		return
// 	}

// 	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

// 	if len(ctx.Args) < 1 {
// 		fmt.Println(Warn + "Missing parameter(s), see `help download`\n")
// 		return
// 	}
// 	if len(ctx.Args) == 1 {
// 		ctx.Args = append(ctx.Args, ".")
// 	}

// 	src := ctx.Args[0]
// 	fileName := filepath.Base(src)
// 	dst, _ := filepath.Abs(ctx.Args[1])
// 	fi, err := os.Stat(dst)
// 	if err != nil {
// 		fmt.Printf(Warn+"%v\n", err)
// 		return
// 	}
// 	if fi.IsDir() {
// 		dst = path.Join(dst, fileName)
// 	}

// 	if _, err := os.Stat(dst); err == nil {
// 		overwrite := false
// 		prompt := &survey.Confirm{Message: "Overwrite local file?"}
// 		survey.AskOne(prompt, &overwrite, nil)
// 		if !overwrite {
// 			return
// 		}
// 	}

// 	ctrl := make(chan bool)
// 	go spin.Until(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
// 	data, _ := proto.Marshal(&sliverpb.DownloadReq{
// 		SliverID: ActiveSession.Session.ID,
// 		Path:     ctx.Args[0],
// 	})
// 	resp := <-rpc(&sliverpb.Envelope{
// 		Type: sliverpb.MsgDownloadReq,
// 		Data: data,
// 	}, cmdTimeout)
// 	ctrl <- true
// 	<-ctrl
// 	if resp.Err != "" {
// 		fmt.Printf(Warn+"Error: %s", resp.Err)
// 		return
// 	}

// 	download := &sliverpb.Download{}
// 	proto.Unmarshal(resp.Data, download)
// 	if download.Encoder == "gzip" {
// 		download.Data, _ = new(util.Gzip).Decode(download.Data)
// 	}
// 	f, err := os.Create(dst)
// 	if err != nil {
// 		fmt.Printf(Warn+"Failed to open local file %s: %v\n", dst, err)
// 	}
// 	defer f.Close()
// 	n, err := f.Write(download.Data)
// 	if err != nil {
// 		fmt.Printf(Warn+"Failed to write data %v\n", err)
// 	} else {
// 		fmt.Printf(Info+"Wrote %d bytes to %s\n", n, dst)
// 	}
// }

// func upload(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

// 	if ActiveSession.Session == nil {
// 		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
// 		return
// 	}
// 	if len(ctx.Args) < 1 {
// 		fmt.Println(Warn + "Missing parameter, see `help upload`\n")
// 		return
// 	}

// 	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

// 	src, _ := filepath.Abs(ctx.Args[0])
// 	_, err := os.Stat(src)
// 	if err != nil {
// 		fmt.Printf(Warn+"%v\n", err)
// 		return
// 	}

// 	if len(ctx.Args) == 1 {
// 		fileName := filepath.Base(src)
// 		ctx.Args = append(ctx.Args, fileName)
// 	}
// 	dst := ctx.Args[1]

// 	fileBuf, err := ioutil.ReadFile(src)
// 	uploadGzip := bytes.NewBuffer([]byte{})
// 	new(util.Gzip).Encode(uploadGzip, fileBuf)

// 	ctrl := make(chan bool)
// 	go spin.Until(fmt.Sprintf("%s -> %s", src, dst), ctrl)
// 	data, _ := proto.Marshal(&sliverpb.UploadReq{
// 		SliverID: ActiveSession.Session.ID,
// 		Path:     dst,
// 		Data:     uploadGzip.Bytes(),
// 		Encoder:  "gzip",
// 	})
// 	resp := <-rpc(&sliverpb.Envelope{
// 		Type: sliverpb.MsgUploadReq,
// 		Data: data,
// 	}, cmdTimeout)
// 	ctrl <- true
// 	<-ctrl
// 	if resp.Err != "" {
// 		fmt.Printf(Warn+"Error: %s", resp.Err)
// 		return
// 	}

// 	upload := &sliverpb.Upload{}
// 	err = proto.Unmarshal(resp.Data, upload)
// 	if err != nil {
// 		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
// 		return
// 	}
// 	if upload.Success {
// 		fmt.Printf(clearln+Info+"Written to %s\n", upload.Path)
// 	} else {
// 		fmt.Printf(Warn+"Error %s\n", upload.Err)
// 	}

// }
