package filesystem

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
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// CatCmd - Display the contents of a remote file.
func CatCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var filePath string
	if len(args) > 0 {
		filePath = args[0]
	}
	if filePath == "" {
		con.PrintErrorf("Missing parameter: file name\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Downloading %s ...", filePath), ctrl)
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request:          con.ActiveTarget.Request(cmd),
		RestrictedToFile: true,
		Path:             filePath,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Response != nil && download.Response.Async {
		con.AddBeaconCallback(download.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, download)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintCat(filePath, download, cmd, con)
		})
		con.PrintAsyncResponse(download.Response)
	} else {
		PrintCat(filePath, download, cmd, con)
	}
}

// PrintCat - Print the download to stdout.
func PrintCat(originalFileName string, download *sliverpb.Download, cmd *cobra.Command, con *console.SliverClient) {
	var (
		lootDownload bool = true
		err          error
	)
	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("name")
	userLootFileType, _ := cmd.Flags().GetString("file-type")
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
		}
	}

	if saveLoot {
		fileType := loot.ValidateLootFileType(userLootFileType, download.Data)
		if lootDownload {
			loot.LootDownload(download, lootName, fileType, cmd, con)
			con.Printf("\n")
		}
	}
	if !strings.Contains(download.Path, originalFileName) {
		con.PrintInfof("Supplied pattern %s matched file %s\n\n", originalFileName, download.Path)
	}
	if color, _ := cmd.Flags().GetBool("colorize-output"); color {
		if err = colorize(download); err != nil {
			con.Println(string(download.Data))
		}
	} else {
		if phex, _ := cmd.Flags().GetBool("hex"); phex {
			con.Println(hex.Dump(download.Data))
		} else {
			con.Println(string(download.Data))
		}
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
