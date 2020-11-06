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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/desertbit/grumble"
)

func ls(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	ls, err := rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: ActiveSession.Request(ctx),
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
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file or directory name\n")
		return
	}

	rm, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
		Request:   ActiveSession.Request(ctx),
		Path:      ctx.Args[0],
		Recursive: ctx.Flags.Bool("recursive"),
		Force:     ctx.Flags.Bool("force"),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", rm.Path)
	}
}

func mkdir(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: directory name\n")
		return
	}

	mkdir, err := rpc.Mkdir(context.Background(), &sliverpb.MkdirReq{
		Request: ActiveSession.Request(ctx),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", mkdir.Path)
	}
}

func cd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	pwd, err := rpc.Cd(context.Background(), &sliverpb.CdReq{
		Request: ActiveSession.Request(ctx),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", pwd.Path)
	}
}

func pwd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pwd, err := rpc.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"%s\n", pwd.Path)
	}
}

func cat(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file name\n")
		return
	}

	download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: ActiveSession.Request(ctx),
		Path:    ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
	}
	if ctx.Flags.Bool("colorize-output") {
		if err = colorize(download); err != nil {
			fmt.Println(string(download.Data))
		}
	} else {
		fmt.Println(string(download.Data))
	}
}

func colorize(f *sliverpb.Download) error {
	lexer := lexers.Match(f.GetPath())
	if lexer == nil {
		lexer = lexers.Analyse(string(f.GetData()))
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	if lexer != nil {
		iterator, err := lexer.Tokenise(nil, string(f.GetData()))
		if err != nil {
			return err
		}
		err = formatter.Format(os.Stdout, style, iterator)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no lexer found")
	}
	return nil
}

func download(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "Missing parameter(s), see `help download`\n")
		return
	}
	if len(ctx.Args) == 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	src := ctx.Args[0]
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(ctx.Args[1])
	fi, err := os.Stat(dst)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	if fi.IsDir() {
		dst = path.Join(dst, fileName)
	}

	if _, err := os.Stat(dst); err == nil {
		overwrite := false
		prompt := &survey.Confirm{Message: "Overwrite local file?"}
		survey.AskOne(prompt, &overwrite, nil)
		if !overwrite {
			return
		}
	}

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
	download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: ActiveSession.Request(ctx),
		Path:    ctx.Args[0],
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			fmt.Printf(Warn+"Decoding failed %s", err)
			return
		}
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Printf(Warn+"Failed to open local file %s: %s\n", dst, err)
		return
	}
	defer dstFile.Close()
	n, err := dstFile.Write(download.Data)
	if err != nil {
		fmt.Printf(Warn+"Failed to write data %v\n", err)
	} else {
		fmt.Printf(Info+"Wrote %d bytes to %s\n", n, dstFile.Name())
	}
}

func upload(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf(Warn + "Missing parameter, see `help upload`\n")
		return
	}

	src, _ := filepath.Abs(ctx.Args[0])
	_, err := os.Stat(src)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	if len(ctx.Args) == 1 {
		fileName := filepath.Base(src)
		ctx.Args = append(ctx.Args, fileName)
	}
	dst := ctx.Args[1]

	fileBuf, err := ioutil.ReadFile(src)
	uploadGzip := new(encoders.Gzip).Encode(fileBuf)

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("%s -> %s", src, dst), ctrl)
	upload, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Request: ActiveSession.Request(ctx),
		Path:    dst,
		Data:    uploadGzip,
		Encoder: "gzip",
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(clearln+Info+"Wrote file to %s\n", upload.Path)
	}
}
