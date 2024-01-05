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
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// LootAddLocalCmd - Add a local file to the server as loot
func LootAddLocalCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	localPath := args[0]
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		con.PrintErrorf("Path '%s' not found\n", localPath)
		return
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = path.Base(localPath)
	}

	var lootFileType clientpb.FileType
	if isTextFile(localPath) {
		lootFileType = clientpb.FileType_TEXT
	} else {
		lootFileType = clientpb.FileType_BINARY
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		con.PrintErrorf("Failed to read file %s\n", err)
		return
	}

	loot := &clientpb.Loot{
		Name:     name,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: filepath.Base(localPath),
			Data: data,
		},
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Uploading loot from %s", localPath), ctrl)
	loot, err = con.Rpc.LootAdd(context.Background(), loot)
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}

	con.PrintInfof("Successfully added loot to server (%s)\n", loot.ID)
}
