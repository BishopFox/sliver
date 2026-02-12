package loot

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
)

// LootCmd - The loot root command
// LootCmd - The 掠夺根命令
func LootCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	allLoot, err := con.Rpc.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to fetch loot %s\n", err)
		return
	}
	PrintAllFileLootTable(allLoot, con)
}

// PrintAllFileLootTable - Displays a table of all file loot
// PrintAllFileLootTable - Displays 所有文件战利品表
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
// PrintLootFile - Display 一件战利品的内容
func PrintLootFile(loot *clientpb.Loot, con *console.SliverClient) {
	if loot.File == nil {
		return
	}
	if loot.File.Name != "" {
		con.PrintInfof("%s %s\n\n", console.StyleBold.Render("File Name:"), loot.File.Name)
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
// 带有 __PH0__ 的 Any 战利品可以保存到磁盘
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
		_ = forms.Confirm("Overwrite local file?", &overwrite)
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
// Taken 来自：__PH0__

// textExt[x] is true if the extension x indicates a text file, and false otherwise.
// 如果扩展名 x 表示文本文件，则 textExt[x] 为 true；如果 otherwise. 为 false，则 textExt[x] 为 true
var textExt = map[string]bool{
	".css": false, // Ignore as text
	".css": false, // Ignore 作为文本
	".js":  false, // Ignore as text
	".js":  false, // Ignore 作为文本
	".svg": false, // Ignore as text
	".svg": false, // Ignore 作为文本
}

// isTextFile reports whether the file has a known extension indicating
// isTextFile 报告文件是否具有已知的扩展名，指示
// a text file, or if a significant chunk of the specified file looks like
// 文本文件，或者指定文件的重要块看起来像
// correct UTF-8; that is, if it is likely that the file contains human-
// 正确的UTF__PH0__；也就是说，如果该文件可能包含人类
// readable text.
// 可读 text.
func isTextFile(filePath string) bool {
	// if the extension is known, use it for decision making
	// 如果扩展名已知，则将其用于决策
	if isText, found := textExt[path.Ext(filePath)]; found {
		return isText
	}

	// the extension is not known; read an initial chunk
	// 扩展名未知；读取初始块
	// of the file and check if it looks like text
	// 文件并检查它是否看起来像文本
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
// isText 报告 s 的重要前缀是否看起来像正确的 UTF__PH0__；
// that is, if it is likely that s is human-readable text.
// 也就是说，如果 s 很可能是 human__PH0__ text.
func isText(sample []byte) bool {
	const max = 1024 // at least utf8.UTFMax
	const max = 1024 // 至少 utf8.UTFMax
	if len(sample) > max {
		sample = sample[0:max]
	}
	for i, c := range string(sample) {
		if i+utf8.UTFMax > len(sample) {
			// last char may be incomplete - ignore
			// 最后一个字符可能不完整 - 忽略
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' && c != '\r' {
			// decoding error or control character - not a text file
			// 解码错误或控制字符 - 不是文本文件
			return false
		}
	}
	return true
}
