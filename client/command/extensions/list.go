package extensions

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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// ExtensionMatch holds the details of a matched extension
// ExtensionMatch 保存匹配扩展的详细信息

type ExtensionMatch struct {
	CommandName string
	Hash        string
	BinPath     string
}

// FindExtensionMatches searches through loaded extensions for matching hashes
// FindExtensionMatches 在加载的扩展中搜索匹配的哈希值
// Returns a map of hash to ExtensionMatch (match will be nil if hash wasn't found)
// Returns 到 ExtensionMatch 的哈希映射（如果未找到哈希，则匹配为零）
func FindExtensionMatches(targetHashes []string) map[string]*ExtensionMatch {
	results := make(map[string]*ExtensionMatch)
	pathCache := make(map[string]*ExtensionMatch)

	// Initialize results map with all target hashes
	// Initialize 结果与所有目标哈希值映射
	for _, hash := range targetHashes {
		results[hash] = nil
	}

	// Search for matches
	// Search 用于匹配
	for _, extCmd := range loadedExtensions {
		if extCmd == nil || len(extCmd.Files) == 0 {
			continue
		}

		for _, file := range extCmd.Files {
			fullPath := filepath.Join(extCmd.Manifest.RootPath, file.Path)

			// Check cache first
			// Check 缓存优先
			var match *ExtensionMatch
			if cached, exists := pathCache[fullPath]; exists {
				match = cached
			} else {
				// Calculate hash if not cached
				// Calculate 哈希（如果未缓存）
				fileData, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				hashBytes := sha256.Sum256(fileData)
				fileHash := hex.EncodeToString(hashBytes[:])

				match = &ExtensionMatch{
					CommandName: extCmd.CommandName,
					Hash:        fileHash,
					BinPath:     fullPath,
				}
				pathCache[fullPath] = match
			}

			if slices.Contains(targetHashes, match.Hash) {
				results[match.Hash] = match
			}
		}
	}


	return results
}

// PrintExtensionMatches prints the extension matches in a formatted table
// PrintExtensionMatches 在格式化表中打印扩展名匹配项
func PrintExtensionMatches(matches map[string]*ExtensionMatch, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Command Name",
		"Sha256 Hash",
		"Bin Path",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Command Name", Mode: table.Asc},
	})

	for hash, match := range matches {
		if match != nil {
			tw.AppendRow(table.Row{
				match.CommandName,
				hash,
				match.BinPath,
			})
		} else {
			tw.AppendRow(table.Row{
				"",
				hash,
				"",
			})
		}
	}

	con.Println(tw.Render())
}

// ExtensionsListCmd - List all extension loaded on the active session/beacon.
// ExtensionsListCmd - List 所有扩展加载到活动 session/beacon. 上
func ExtensionsListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if extList.Response != nil && extList.Response.Err != "" {
		con.PrintErrorf("%s\n", extList.Response.Err)
		return
	}

	if len(extList.Names) == 0 {
		return
	}

	con.PrintInfof("Loaded extensions:\n\n")
	matches := FindExtensionMatches(extList.Names)
	PrintExtensionMatches(matches, con)
}
