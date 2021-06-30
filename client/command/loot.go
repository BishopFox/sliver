package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
)

var (
	ErrInvalidLootType = errors.New("invalid loot type")
)

func lootRoot(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	filter := ctx.Flags.String("filter")
	var allLoot *clientpb.AllLoot
	var err error
	if filter == "" {
		allLoot, err = rpc.LootAll(context.Background(), &commonpb.Empty{})
		if err != nil {
			fmt.Printf(Warn+"Failed to fetch loot %s\n", err)
			return
		}
	} else {
		lootType, err := lootTypeFromHumanStr(filter)
		if err != nil {
			fmt.Printf(Warn + "Invalid loot type see --help")
			return
		}
		allLoot, err = rpc.LootAllOf(context.Background(), &clientpb.Loot{Type: lootType})
		if err != nil {
			fmt.Printf(Warn+"Failed to fetch loot %s\n", err)
			return
		}
	}
	displayLootTable(allLoot)
}

func lootAddLocal(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	localPath := ctx.Args.String("path")
	if localPath == "" {
		fmt.Printf(Warn + "Missing local path argument, see --help\n")
		return
	}
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		fmt.Printf(Warn+"Path '%s' not found\n", localPath)
		return
	}

	name := ctx.Flags.String("name")
	if name == "" {
		name = path.Base(localPath)
	}
	lootTypeStr := ctx.Flags.String("type")
	var lootType clientpb.LootType
	var err error
	if lootTypeStr != "" {
		lootType, err = lootTypeFromHumanStr(lootTypeStr)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
	} else {
		if IsTextFile(localPath) {
			lootType = clientpb.LootType_TEXT
		} else {
			lootType = clientpb.LootType_BINARY
		}
	}
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		fmt.Printf(Warn+"Failed to read file %s\n", err)
		return
	}

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Uploading loot from %s", localPath), ctrl)
	loot, err := rpc.LootAdd(context.Background(), &clientpb.Loot{
		Name: name,
		Type: lootType,
		File: &commonpb.File{
			Name: path.Base(localPath),
			Data: data,
		},
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	}

	fmt.Printf(Info+"Successfully added loot to server (%s)\n", loot.LootID)
}

func lootAddRemote(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	remotePath := ctx.Args.String("path")
	if remotePath == "" {
		fmt.Printf(Warn + "Missing remote path argument, see --help\n")
		return
	}
	name := ctx.Flags.String("name")
	if name == "" {
		name = path.Base(remotePath)
	}

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Looting remote file %s", remotePath), ctrl)

	download, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: ActiveSession.Request(ctx),
		Path:    remotePath,
	})
	if err != nil {
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(Warn+"%s\n", err) // Download failed
			return
		}
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			fmt.Printf(Warn+"Decoding failed %s", err)
			return
		}
	}

	// Determine type based on download buffer
	var lootType clientpb.LootType
	if IsText(download.Data) {
		lootType = clientpb.LootType_TEXT
	} else {
		lootType = clientpb.LootType_BINARY
	}

	loot, err := rpc.LootAdd(context.Background(), &clientpb.Loot{
		Name: name,
		Type: lootType,
		File: &commonpb.File{
			Name: path.Base(remotePath),
			Data: download.Data,
		},
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Successfully added loot to server (%s)\n", loot.LootID)
}

func lootRm(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

}

func lootFetch(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

}

func displayLootTable(allLoot *clientpb.AllLoot) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		fmt.Printf(Info + "No loot 🙁\n")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Loot Type\tName\t")
	fmt.Fprintf(table, "%s\t%s\t\n",
		strings.Repeat("=", len("Loot Type")),
		strings.Repeat("=", len("Name")),
	)
	for _, loot := range allLoot.Loot {
		fmt.Fprintf(table, "%s\t%s\t\n", loot.Type, loot.Name)
	}

	table.Flush()
	fmt.Printf(outputBuf.String())
}

func lootTypeToStr(value clientpb.LootType) string {
	switch value {
	case clientpb.LootType_BINARY:
		return "binary file"
	case clientpb.LootType_TEXT:
		return "text"
	case clientpb.LootType_CREDENTIAL:
		return "credential"
	default:
		return ""
	}
}

func lootTypeFromHumanStr(value string) (clientpb.LootType, error) {
	switch strings.ToLower(value) {

	case "binary":
		return clientpb.LootType_BINARY, nil
	case "bin":
		return clientpb.LootType_BINARY, nil

	case "text":
		return clientpb.LootType_TEXT, nil
	case "utf8":
		return clientpb.LootType_TEXT, nil
	case "utf-8":
		return clientpb.LootType_TEXT, nil

	case "credential":
		return clientpb.LootType_CREDENTIAL, nil
	case "cred":
		return clientpb.LootType_CREDENTIAL, nil
	case "creds":
		return clientpb.LootType_CREDENTIAL, nil

	default:
		return -1, ErrInvalidLootType
	}
}

// Taken from: https://cs.opensource.google/go/x/tools/+/refs/tags/v0.1.4:godoc/util/util.go;l=69

// textExt[x] is true if the extension x indicates a text file, and false otherwise.
var textExt = map[string]bool{
	".css": false, // must be served raw
	".js":  false, // must be served raw
	".svg": false, // must be served raw
}

// IsTextFile reports whether the file has a known extension indicating
// a text file, or if a significant chunk of the specified file looks like
// correct UTF-8; that is, if it is likely that the file contains human-
// readable text.
func IsTextFile(filePath string) bool {
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

	return IsText(buf[0:n])
}

// IsText reports whether a significant prefix of s looks like correct UTF-8;
// that is, if it is likely that s is human-readable text.
func IsText(sample []byte) bool {
	const max = 1024 // at least utf8.UTFMax
	if len(sample) > max {
		sample = sample[0:max]
	}
	for i, c := range string(sample) {
		if i+utf8.UTFMax > len(sample) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}
