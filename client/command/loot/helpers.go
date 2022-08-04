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
	"path"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
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
	var lootType clientpb.LootType
	if isCredential {
		lootType = clientpb.LootType_LOOT_CREDENTIAL
	} else {
		lootType = clientpb.LootType_LOOT_FILE
	}
	var lootFileType clientpb.FileType
	if isText(data) || strings.HasSuffix(fileName, ".txt") {
		lootFileType = clientpb.FileType_TEXT
	} else {
		lootFileType = clientpb.FileType_BINARY
	}
	loot := &clientpb.Loot{
		Name:     name,
		Type:     lootType,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: path.Base(fileName),
			Data: data,
		},
	}
	if lootType == clientpb.LootType_LOOT_CREDENTIAL {
		loot.CredentialType = clientpb.CredentialType_FILE
	}
	_, err := rpc.LootAdd(context.Background(), loot)
	return err
}

// AddLootUserPassword - Add user/password as loot
func AddLootUserPassword(rpc rpcpb.SliverRPCClient, name string, user string, password string) error {
	loot := &clientpb.Loot{
		Name:           name,
		Type:           clientpb.LootType_LOOT_CREDENTIAL,
		CredentialType: clientpb.CredentialType_USER_PASSWORD,
		Credential: &clientpb.Credential{
			User:     user,
			Password: password,
		},
	}
	_, err := rpc.LootAdd(context.Background(), loot)
	return err
}

// AddLootAPIKey - Add a api key as loot
func AddLootAPIKey(rpc rpcpb.SliverRPCClient, name string, apiKey string) error {
	loot := &clientpb.Loot{
		Name:           name,
		Type:           clientpb.LootType_LOOT_CREDENTIAL,
		CredentialType: clientpb.CredentialType_API_KEY,
		Credential: &clientpb.Credential{
			APIKey: apiKey,
		},
	}
	_, err := rpc.LootAdd(context.Background(), loot)
	return err
}

// SelectCredentials - An interactive menu for the user to select a piece of loot
func SelectCredentials(con *console.SliverConsoleClient) (*clientpb.Loot, error) {
	allLoot, err := con.Rpc.LootAllOf(context.Background(), &clientpb.Loot{
		Type: clientpb.LootType_LOOT_CREDENTIAL,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}

	// Render selection table
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, loot := range allLoot.Loot {
		fmt.Fprintf(table, "%s\t%s\t%s\t\n", loot.Name, loot.CredentialType, loot.LootID)
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no loot to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a piece of credentials:",
		Options: options,
	}
	err = survey.AskOne(prompt, &selected)
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

// SelectLoot - Interactive menu for the user to select a piece loot (all types)
func SelectLoot(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) (*clientpb.Loot, error) {

	// Fetch data with optional filter
	filter := ctx.Flags.String("filter")
	var allLoot *clientpb.AllLoot
	var err error
	if filter == "" {
		allLoot, err = rpc.LootAll(context.Background(), &commonpb.Empty{})
		if err != nil {
			return nil, err
		}
	} else {
		lootType, err := lootTypeFromHumanStr(filter)
		if err != nil {
			return nil, ErrInvalidFileType
		}
		allLoot, err = rpc.LootAllOf(context.Background(), &clientpb.Loot{Type: lootType})
		if err != nil {
			return nil, err
		}
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
			fmt.Fprintf(table, "%s\t%s\t%s\t\n", loot.Name, loot.Type, loot.LootID)
		} else {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n", loot.Name, filename, loot.Type, loot.LootID)
		}
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no loot to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a piece of loot:",
		Options: options,
	}
	err = survey.AskOne(prompt, &selected)
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
