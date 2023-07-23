package creds

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

const (
	UserColonHashNewlineFormat = "user:hash" // username:hash\n
	HashNewlineFormat          = "hash"      // hash\n
	CSVFormat                  = "csv"       // username,hash\n
)

// CredsCmd - Add new credentials.
func CredsAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	collection, _ := cmd.Flags().GetString("collection")
	username, _ := cmd.Flags().GetString("username")
	plaintext, _ := cmd.Flags().GetString("plaintext")
	hash, _ := cmd.Flags().GetString("hash")
	hashTypeF, _ := cmd.Flags().GetString("hash-type")
	hashType := parseHashTypeString(hashTypeF)
	if plaintext == "" && hash == "" {
		con.PrintErrorf("Either a plaintext or a hash must be provided\n")
		return
	}
	if hashType == clientpb.HashType_INVALID {
		con.PrintErrorf("Invalid hash type '%s'\n", hashTypeF)
		return
	}
	_, err := con.Rpc.CredsAdd(context.Background(), &clientpb.Credentials{
		Credentials: []*clientpb.Credential{
			{
				Collection: collection,
				Username:   username,
				Plaintext:  plaintext,
				Hash:       hash,
				HashType:   hashType,
			},
		},
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	creds, err := con.Rpc.Creds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintCreds(creds.Credentials, con)
}

// CredsCmd - Add new credentials.
func CredsAddHashFileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	collection, _ := cmd.Flags().GetString("collection")
	filePath := args[0]
	fileFormat, _ := cmd.Flags().GetString("file-format")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		con.PrintErrorf("File '%s' does not exist\n", filePath)
		return
	}
	hashTypeF, _ := cmd.Flags().GetString("hash-type")
	hashType := parseHashTypeString(hashTypeF)
	if hashType == clientpb.HashType_INVALID {
		con.PrintErrorf("Invalid hash type '%s'\n", hashTypeF)
		return
	}

	con.PrintInfof("Parsing file '%s' as '%s' format ...\n", filePath, fileFormat)
	var creds *clientpb.Credentials
	var err error
	switch fileFormat {
	case UserColonHashNewlineFormat:
		creds, err = parseUserColonHashNewline(filePath, hashType)
	case HashNewlineFormat:
		creds, err = parseHashNewline(filePath, hashType)
	case CSVFormat:
		creds, err = parseCSV(filePath, hashType)
	default:
		con.PrintErrorf("Invalid file format '%s', see 'creds add file --help'\n", fileFormat)
		return
	}
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	for _, cred := range creds.Credentials {
		cred.Collection = collection
	}
	con.PrintInfof("Adding %d credential(s) ...\n", len(creds.Credentials))
	_, err = con.Rpc.CredsAdd(context.Background(), creds)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	creds, err = con.Rpc.Creds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintCreds(creds.Credentials, con)
}

func parseHashType(raw string) clientpb.HashType {
	hashInt, err := strconv.Atoi(raw)
	if err == nil {
		return clientpb.HashType(hashInt)
	}
	return clientpb.HashType_INVALID
}

// same as parseHashType, but use the string representation.
func parseHashTypeString(raw string) clientpb.HashType {
	if hashType, valid := clientpb.HashType_value[raw]; valid {
		return clientpb.HashType(hashType)
	}

	return clientpb.HashType_INVALID
}

func parseUserColonHashNewline(filePath string, hashType clientpb.HashType) (*clientpb.Credentials, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	creds := &clientpb.Credentials{}
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		creds.Credentials = append(creds.Credentials, &clientpb.Credential{
			Username: parts[0],
			Hash:     parts[1],
			HashType: hashType,
		})
	}
	return creds, nil
}

func parseHashNewline(filePath string, hashType clientpb.HashType) (*clientpb.Credentials, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	creds := &clientpb.Credentials{}
	for scanner.Scan() {
		line := scanner.Text()
		creds.Credentials = append(creds.Credentials, &clientpb.Credential{
			Hash:     line,
			HashType: hashType,
		})
	}
	return creds, nil
}

func parseCSV(filePath string, hashType clientpb.HashType) (*clientpb.Credentials, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	creds := &clientpb.Credentials{}
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		creds.Credentials = append(creds.Credentials, &clientpb.Credential{
			Username: parts[0],
			Hash:     parts[1],
			HashType: hashType,
		})
	}
	return creds, nil
}
