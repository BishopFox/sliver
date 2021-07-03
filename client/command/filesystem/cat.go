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
	"fmt"
	"os"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"

	"github.com/desertbit/grumble"
)

func CatCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	filePath := ctx.Args.String("path")
	if filePath == "" {
		con.PrintErrorf("Missing parameter: file name\n")
		return
	}

	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if ctx.Flags.Bool("colorize-output") {
		if err = colorize(download); err != nil {
			con.Println(string(download.Data))
		}
	} else {
		con.Println(string(download.Data))
	}
	// if ctx.Flags.Bool("loot") && 0 < len(download.Data) {
	// 	err = AddLootFile(rpc, fmt.Sprintf("[cat] %s", filepath.Base(filePath)), filePath, download.Data, false)
	// 	if err != nil {
	// 		con.PrintErrorf("Failed to save output as loot: %s", err)
	// 	} else {
	// 		fmt.Printf(clearln + Info + "Output saved as loot\n")
	// 	}
	// }
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
