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
	----------------------------------------------------------------------

	Loot helper functions for use by other commands

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
	ErrInvalidFileType = errors.New("invalid file type")
	// ErrInvalidLootType - Invalid loot type
	ErrInvalidLootType = errors.New("invalid loot type")
	// ErrNoLootFileData - No loot file data
	ErrNoLootFileData = errors.New("no loot file data")
)

// AddLootFile - Add a file as loot
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
func SelectLoot(cmd *cobra.Command, rpc rpcpb.SliverRPCClient) (*clientpb.Loot, error) {
	// Fetch data with optional filter
	var allLoot *clientpb.AllLoot
	var err error

	allLoot, err = rpc.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}

	// Render selection table
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
