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
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

// CatCmd - Display the contents of a remote file
func CatCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := ctx.Args.String("path")
	if filePath == "" {
		con.PrintErrorf("Missing parameter: file name\n")
		return
	}

	saveLoot := ctx.Flags.Bool("loot")
	lootName := ctx.Flags.String("name")
	userLootType := ctx.Flags.String("type")
	userLootFileType := ctx.Flags.String("file-type")
	var lootDownload bool = true

	download, err := PerformDownload(filePath, filepath.Base(filePath), "console", ctx, con)

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if saveLoot {
		lootType, err := loot.ValidateLootType(userLootType)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			// Even if the loot type is bad, we can still print the result to the screen
			// We will not loot it though
			lootDownload = false
		}
		fileType := loot.ValidateLootFileType(userLootFileType, download.Data)

		if lootDownload {
			loot.LootDownload(download, lootName, lootType, fileType, ctx, con)
			con.Printf("\n")
		}
	}

	PrintCat(download, ctx, con)
}

// PrintCat - Print the download to stdout
func PrintCat(download *sliverpb.Download, ctx *grumble.Context, con *console.SliverConsoleClient) {
	var err error

	if ctx.Flags.Bool("colorize-output") {
		if err = colorize(download); err != nil {
			con.Println(string(download.Data))
		}
	} else {
		if ctx.Flags.Bool("hex") {
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
