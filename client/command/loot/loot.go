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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

// LootCmd - The loot root command
func LootCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	filter := ctx.Flags.String("filter")
	var allLoot *clientpb.AllLoot
	var err error
	if filter == "" {
		allLoot, err = con.Rpc.LootAll(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Failed to fetch loot %s\n", err)
			return
		}
	} else {
		lootType, err := lootTypeFromHumanStr(filter)
		if err != nil {
			con.PrintErrorf("Invalid loot type see --help")
			return
		}
		allLoot, err = con.Rpc.LootAllOf(context.Background(), &clientpb.Loot{Type: lootType})
		if err != nil {
			con.PrintErrorf("Failed to fetch loot %s\n", err)
			return
		}
	}
	if filter == "" {
		PrintAllLootTable(con.App.Stdout(), allLoot)
	} else {
		lootType, _ := lootTypeFromHumanStr(filter)
		switch lootType {
		case clientpb.LootType_LOOT_FILE:
			PrintAllFileLootTable(con.App.Stdout(), allLoot)
		case clientpb.LootType_LOOT_CREDENTIAL:
			PrintAllCredentialLootTable(con.App.Stdout(), allLoot)
		}
	}
}

// PrintLootFile - Display the contents of a piece of loot
func PrintLootFile(stdout io.Writer, loot *clientpb.Loot) {
	if loot.File == nil {
		return
	}
	fmt.Fprintln(stdout)

	if loot.File.Name != "" {
		fmt.Fprintf(stdout, "%sFile Name:%s %s\n\n", console.Bold, console.Normal, loot.File.Name)
	}
	if loot.File.Data != nil && 0 < len(loot.File.Data) {
		if loot.FileType == clientpb.FileType_TEXT || isText(loot.File.Data) {
			fmt.Fprintf(stdout, "%s", string(loot.File.Data))
		} else {
			fmt.Fprintf(stdout, "<%d bytes of binary data>\n", len(loot.File.Data))
		}
	} else {
		fmt.Fprintf(stdout, "No file data\n")
	}
}

func PrintLootCredential(stdout io.Writer, loot *clientpb.Loot) {
	stdout.Write([]byte("\n"))
	switch loot.CredentialType {
	case clientpb.CredentialType_USER_PASSWORD:
		if loot.Credential != nil {
			fmt.Fprintf(stdout, "%s    User:%s %s\n", console.Bold, console.Normal, loot.Credential.User)
			fmt.Fprintf(stdout, "%sPassword:%s %s\n", console.Bold, console.Normal, loot.Credential.Password)
		}
		if loot.File != nil {
			PrintLootFile(stdout, loot)
		}
	case clientpb.CredentialType_API_KEY:
		if loot.Credential != nil {
			fmt.Fprintf(stdout, "%sAPI Key:%s %s\n", console.Bold, console.Normal, loot.Credential.APIKey)
		}
		if loot.File != nil {
			PrintLootFile(stdout, loot)
		}
	case clientpb.CredentialType_FILE:
		if loot.File != nil {
			PrintLootFile(stdout, loot)
		}
	default:
		fmt.Fprintf(stdout, "%v\n", loot.Credential) // Well, let's give it our best
	}
}

// Any loot with a "File" can be saved to disk
func saveLootToDisk(ctx *grumble.Context, loot *clientpb.Loot) (string, error) {
	if loot.File == nil {
		return "", errors.New("Loot does not contain a file")
	}

	saveTo := ctx.Flags.String("save")
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
	err = ioutil.WriteFile(saveTo, loot.File.Data, 0600)
	return saveTo, err
}

// PrintAllLootTable - Displays a table of all loot
func PrintAllLootTable(stdout io.Writer, allLoot *clientpb.AllLoot) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		fmt.Fprintf(stdout, console.Info+"No loot 🙁\n")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Type\tName\tFile Name\tUUID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Type")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("File Name")),
		strings.Repeat("=", len("UUID")),
	)

	for _, loot := range allLoot.Loot {
		filename := ""
		if loot.File != nil {
			filename = loot.File.Name
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n", lootTypeToStr(loot.Type), loot.Name, filename, loot.LootID)
	}

	table.Flush()
	fmt.Fprintf(stdout, outputBuf.String())
}

// PrintAllFileLootTable - Displays a table of all file loot
func PrintAllFileLootTable(stdout io.Writer, allLoot *clientpb.AllLoot) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		fmt.Fprintf(stdout, console.Info+"No loot 🙁\n")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Type\tName\tFile Name\tSize\tUUID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Type")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("File Name")),
		strings.Repeat("=", len("Size")),
		strings.Repeat("=", len("UUID")),
	)
	for _, loot := range allLoot.Loot {
		if loot.Type != clientpb.LootType_LOOT_FILE {
			continue
		}
		size := 0
		name := ""
		if loot.File != nil {
			name = loot.File.Name
			size = len(loot.File.Data)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%d\t%s\t\n",
			fileTypeToStr(loot.FileType),
			loot.Name,
			name,
			size,
			loot.LootID,
		)
	}

	table.Flush()
	fmt.Fprintf(stdout, outputBuf.String())
}

// PrintAllCredentialLootTable - Displays a table of all credential loot
func PrintAllCredentialLootTable(stdout io.Writer, allLoot *clientpb.AllLoot) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		fmt.Fprintf(stdout, console.Info+"No loot 🙁\n")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Type\tName\tUser\tPassword\tAPI Key\tFile Name\tUUID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Type")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("User")),
		strings.Repeat("=", len("Password")),
		strings.Repeat("=", len("API Key")),
		strings.Repeat("=", len("File Name")),
		strings.Repeat("=", len("UUID")),
	)
	for _, loot := range allLoot.Loot {
		if loot.Type != clientpb.LootType_LOOT_CREDENTIAL {
			continue
		}
		fileName := ""
		if loot.File != nil {
			fileName = loot.File.Name
		}
		user := ""
		password := ""
		apiKey := ""
		if loot.Credential != nil {
			user = loot.Credential.User
			password = loot.Credential.Password
			apiKey = loot.Credential.APIKey
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			credentialTypeToString(loot.CredentialType),
			loot.Name,
			user,
			password,
			apiKey,
			fileName,
			loot.LootID,
		)
	}
	table.Flush()
	fmt.Fprintf(stdout, outputBuf.String())
}

func lootTypeToStr(value clientpb.LootType) string {
	switch value {
	case clientpb.LootType_LOOT_FILE:
		return "File"
	case clientpb.LootType_LOOT_CREDENTIAL:
		return "Credential"
	default:
		return ""
	}
}

func credentialTypeToString(value clientpb.CredentialType) string {
	switch value {
	case clientpb.CredentialType_API_KEY:
		return "API Key"
	case clientpb.CredentialType_USER_PASSWORD:
		return "User/Password"
	case clientpb.CredentialType_FILE:
		return "File"
	default:
		return ""
	}
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

func lootTypeFromHumanStr(value string) (clientpb.LootType, error) {
	switch strings.ToLower(value) {

	case "c":
		fallthrough
	case "cred":
		fallthrough
	case "creds":
		fallthrough
	case "credentials":
		fallthrough
	case "credential":
		return clientpb.LootType_LOOT_CREDENTIAL, nil

	case "f":
		fallthrough
	case "files":
		fallthrough
	case "file":
		return clientpb.LootType_LOOT_FILE, nil

	default:
		return -1, ErrInvalidLootType
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
