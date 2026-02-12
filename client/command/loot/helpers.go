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
	----------------------------------------------------------------------

	Loot helper functions for use by other commands
	Loot 供其他命令使用的辅助函数

*/

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// ErrInvalidFileType - Invalid file type
	// ErrInvalidFileType - Invalid 文件类型
	ErrInvalidFileType = errors.New("invalid file type")
	// ErrInvalidLootType - Invalid loot type
	// ErrInvalidLootType - Invalid 战利品类型
	ErrInvalidLootType = errors.New("invalid loot type")
	// ErrNoLootFileData - No loot file data
	// ErrNoLootFileData - No 战利品文件数据
	ErrNoLootFileData = errors.New("no loot file data")
)

// AddLootFile - Add a file as loot
// AddLootFile - Add 作为战利品的文件
func AddLootFile(rpc rpcpb.SliverRPCClient, name string, fileName string, data []byte, isCredential bool) error {
	if len(data) < 1 {
		return ErrNoLootFileData
	}

	var lootFileType clientpb.FileType
	if isText(data) || strings.HasSuffix(fileName, ".txt") {
		lootFileType = clientpb.FileType_TEXT
	} else {
		lootFileType = clientpb.FileType_BINARY
	}
	loot := &clientpb.Loot{
		Name:     name,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: filepath.Base(fileName),
			Data: data,
		},
	}

	_, err := rpc.LootAdd(context.Background(), loot)
	return err
}

// SelectLoot - Interactive menu for the user to select a piece loot (all types)
// SelectLoot - Interactive 菜单供用户选择一件战利品（所有类型）
func SelectLoot(cmd *cobra.Command, rpc rpcpb.SliverRPCClient) (*clientpb.Loot, error) {
	// Fetch data with optional filter
	// 带有可选过滤器的 Fetch 数据
	var allLoot *clientpb.AllLoot
	var err error

	allLoot, err = rpc.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}

	// Render selection table
	// Render选型表
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, loot := range allLoot.Loot {
		filename := ""
		if loot.File != nil {
			filename = loot.File.Name
		}
		if loot.Name == filename {
			fmt.Fprintf(table, "%s\t%s\t\n", loot.Name, loot.ID)
		} else {
			fmt.Fprintf(table, "%s\t(File name: %s)\t%s\t\n", loot.Name, filename, loot.ID)
		}
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no loot to select from")
	}

	selected := ""
	err = forms.Select("Select a piece of loot:", options, &selected)
	if err != nil {
		return nil, err
	}
	for index, value := range options {
		if value == selected {
			return allLoot.Loot[index], nil
		}
	}
	return nil, errors.New("loot not found")
}
