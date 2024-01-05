package loot

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
)

// LootCmd - The loot root command
func LootCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	allLoot, err := con.Rpc.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to fetch loot %s\n", err)
		return
	}
	PrintAllFileLootTable(allLoot, con)
}

// PrintAllFileLootTable - Displays a table of all file loot
func PrintAllFileLootTable(allLoot *clientpb.AllLoot, con *console.SliverClient) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		con.PrintInfof("No loot 🙁\n")
		return
	}
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Name",
		"File Name",
		"Type",
		"Size",
	})
	for _, loot := range allLoot.Loot {
		if loot.File != nil {
			tw.AppendRow(table.Row{
				strings.Split(loot.ID, "-")[0],
				loot.Name,
				loot.File.Name,
				fileTypeToStr(loot.FileType),
				util.ByteCountBinary(loot.Size),
			})
		}
	}
	con.Printf("%s\n", tw.Render())
}

// PrintLootFile - Display the contents of a piece of loot
func PrintLootFile(loot *clientpb.Loot, con *console.SliverClient) {
	if loot.File == nil {
		return
	}
	if loot.File.Name != "" {
		con.PrintInfof("%sFile Name:%s %s\n\n", console.Bold, console.Normal, loot.File.Name)
	}
	if loot.File.Data != nil && 0 < len(loot.File.Data) {
		if loot.FileType == clientpb.FileType_TEXT || isText(loot.File.Data) {
			con.Printf("%s\n", string(loot.File.Data))
		} else {
			con.PrintInfof("<%d bytes of binary data>\n", len(loot.File.Data))
		}
	} else {
		con.PrintInfof("No file data\n")
	}
}

// Any loot with a "File" can be saved to disk
func saveLootToDisk(cmd *cobra.Command, loot *clientpb.Loot) (string, error) {
	if loot.File == nil {
		return "", errors.New("loot does not contain a file")
	}

	saveTo, _ := cmd.Flags().GetString("save")
	fi, err := os.Stat(saveTo)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil && fi.IsDir() {
		saveTo = path.Join(saveTo, path.Base(loot.File.Name))
	}
	if _, err := os.Stat(saveTo); err == nil {
		overwrite := false
		prompt := &survey.Confirm{Message: "Overwrite local file?"}
		survey.AskOne(prompt, &overwrite, nil)
		if !overwrite {
			return "", nil
		}
	}
	err = os.WriteFile(saveTo, loot.File.Data, 0o600)
	return saveTo, err
}

func fileTypeToStr(value clientpb.FileType) string {
	switch value {
	case clientpb.FileType_BINARY:
		return "Binary"
	case clientpb.FileType_TEXT:
		return "Text"
	default:
		return ""
	}
}

func lootFileTypeFromHumanStr(value string) (clientpb.FileType, error) {
	switch strings.ToLower(value) {

	case "b":
		fallthrough
	case "bin":
		fallthrough
	case "binary":
		return clientpb.FileType_BINARY, nil

	case "t":
		fallthrough
	case "utf-8":
		fallthrough
	case "utf8":
		fallthrough
	case "txt":
		fallthrough
	case "text":
		return clientpb.FileType_TEXT, nil

	default:
		return -1, ErrInvalidFileType
	}
}

// Taken from: https://cs.opensource.google/go/x/tools/+/refs/tags/v0.1.4:godoc/util/util.go;l=69

// textExt[x] is true if the extension x indicates a text file, and false otherwise.
var textExt = map[string]bool{
	".css": false, // Ignore as text
	".js":  false, // Ignore as text
	".svg": false, // Ignore as text
}

// isTextFile reports whether the file has a known extension indicating
// a text file, or if a significant chunk of the specified file looks like
// correct UTF-8; that is, if it is likely that the file contains human-
// readable text.
func isTextFile(filePath string) bool {
	// if the extension is known, use it for decision making
	if isText, found := textExt[path.Ext(filePath)]; found {
		return isText
	}

	// the extension is not known; read an initial chunk
	// of the file and check if it looks like text
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	var buf [1024]byte
	n, err := f.Read(buf[0:])
	if err != nil {
		return false
	}

	return isText(buf[0:n])
}

// isText reports whether a significant prefix of s looks like correct UTF-8;
// that is, if it is likely that s is human-readable text.
func isText(sample []byte) bool {
	const max = 1024 // at least utf8.UTFMax
	if len(sample) > max {
		sample = sample[0:max]
	}
	for i, c := range string(sample) {
		if i+utf8.UTFMax > len(sample) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' && c != '\r' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}
