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

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/desertbit/grumble"
)

var (
	ErrInvalidFileType = errors.New("invalid file type")
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
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		fmt.Printf(Warn+"Path '%s' not found\n", localPath)
		return
	}

	name := ctx.Flags.String("name")
	if name == "" {
		name = path.Base(localPath)
	}

	var lootType clientpb.LootType
	var err error
	lootTypeStr := ctx.Flags.String("type")
	if lootTypeStr != "" {
		lootType, err = lootTypeFromHumanStr(lootTypeStr)
		if err == ErrInvalidLootType {
			fmt.Printf(Warn+"Invalid loot type %s", lootTypeStr)
			return
		}
	} else {
		lootType = clientpb.LootType_LOOT_FILE
	}

	lootFileTypeStr := ctx.Flags.String("file-type")
	var lootFileType clientpb.FileType
	if lootFileTypeStr != "" {
		lootFileType, err = lootFileTypeFromHumanStr(lootFileTypeStr)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
	} else {
		if isTextFile(localPath) {
			lootFileType = clientpb.FileType_TEXT
		} else {
			lootFileType = clientpb.FileType_BINARY
		}
	}
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		fmt.Printf(Warn+"Failed to read file %s\n", err)
		return
	}

	loot := &clientpb.Loot{
		Name:     name,
		Type:     lootType,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: path.Base(localPath),
			Data: data,
		},
	}

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Uploading loot from %s", localPath), ctrl)
	loot, err = rpc.LootAdd(context.Background(), loot)
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
			fmt.Printf(Warn+"Invalid loot type %s", lootTypeStr)
			return
		}
	} else {
		lootType = clientpb.LootType_LOOT_FILE
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

	loot, err = rpc.LootAdd(context.Background(), loot)
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Successfully added loot to server (%s)\n", loot.LootID)
}

func lootRm(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	loot, err := selectLoot(ctx, rpc)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	_, err = rpc.LootRm(context.Background(), loot)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	fmt.Printf(Info + "Removed loot from server\n")
}

func lootFetch(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	loot, err := selectLoot(ctx, rpc)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	loot, err = rpc.LootContent(context.Background(), loot)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	// Handle loot based on its type
	switch loot.Type {
	case clientpb.LootType_LOOT_FILE:
		displayLootFile(loot)
	case clientpb.LootType_LOOT_CREDENTIAL:
		displayLootCredential(loot)
	}

	if ctx.Flags.String("save") != "" {
		savedTo, err := saveLootToDisk(ctx, loot)
		if err != nil {
			fmt.Printf("Failed to save loot %s\n", err)
		}
		if savedTo != "" {
			fmt.Printf(Info+"Saved loot to %s\n", savedTo)
		}
	}
}

func lootAddCredential(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	prompt := &survey.Select{
		Message: "Choose a credential type:",
		Options: []string{
			clientpb.CredentialType_API_KEY.String(),
			clientpb.CredentialType_USER_PASSWORD.String(),
		},
	}
	credType := ""
	survey.AskOne(prompt, &credType, survey.WithValidator(survey.Required))
	name := ctx.Flags.String("name")
	if name == "" {
		namePrompt := &survey.Input{Message: "Credential Name: "}
		survey.AskOne(namePrompt, &name)
	}

	loot := &clientpb.Loot{
		Type:       clientpb.LootType_LOOT_CREDENTIAL,
		Name:       name,
		Credential: &clientpb.Credential{},
	}

	switch credType {
	case clientpb.CredentialType_USER_PASSWORD.String():
		loot.CredentialType = clientpb.CredentialType_USER_PASSWORD
		usernamePrompt := &survey.Input{Message: "Username: "}
		survey.AskOne(usernamePrompt, &loot.Credential.User)
		passwordPrompt := &survey.Input{Message: "Password: "}
		survey.AskOne(passwordPrompt, &loot.Credential.Password)
	case clientpb.CredentialType_API_KEY.String():
		loot.CredentialType = clientpb.CredentialType_API_KEY
		usernamePrompt := &survey.Input{Message: "API Key: "}
		survey.AskOne(usernamePrompt, &loot.Credential.APIKey)
	}

	loot, err := rpc.LootAdd(context.Background(), loot)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Successfully added loot to server (%s)\n", loot.LootID)
}

func displayLootFile(loot *clientpb.Loot) {
	if loot.File == nil {
		return
	}
	fmt.Println()

	if loot.File.Name != "" {
		fmt.Printf("%sFile Name:%s %s\n\n", bold, normal, loot.File.Name)
	}
	if isText(loot.File.Data) {
		fmt.Printf(string(loot.File.Data))
	} else {
		fmt.Printf("<%d bytes of binary data>\n", len(loot.File.Data))
	}
}

func displayLootCredential(loot *clientpb.Loot) {
	fmt.Println()
	switch loot.CredentialType {
	case clientpb.CredentialType_USER_PASSWORD:
		if loot.Credential != nil {
			fmt.Printf("%s    User:%s %s\n", bold, normal, loot.Credential.User)
			fmt.Printf("%sPassword:%s %s\n", bold, normal, loot.Credential.Password)
		}
		if loot.File != nil {
			displayLootFile(loot)
		}
	case clientpb.CredentialType_API_KEY:
		if loot.Credential != nil {
			fmt.Printf("%sAPI Key:%s %s\n", bold, normal, loot.Credential.APIKey)
		}
		if loot.File != nil {
			displayLootFile(loot)
		}
	default:
		fmt.Printf("%v\n", loot.Credential) // Well, let's give it our best
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

func displayLootTable(allLoot *clientpb.AllLoot) {
	if allLoot == nil || len(allLoot.Loot) == 0 {
		fmt.Printf(Info + "No loot 🙁\n")
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Type\tName\t")
	fmt.Fprintf(table, "%s\t%s\t\n",
		strings.Repeat("=", len("Type")),
		strings.Repeat("=", len("Name")),
	)
	for _, loot := range allLoot.Loot {
		fmt.Fprintf(table, "%s\t%s\t\n", loot.Type, loot.Name)
	}

	table.Flush()
	fmt.Printf(outputBuf.String())
}

func selectLoot(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) (*clientpb.Loot, error) {

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
		fmt.Fprintf(table, "%s\t%s\t%s\t\n", loot.Name, loot.Type, loot.LootID)
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

func lootFileTypeToStr(value clientpb.FileType) string {
	switch value {
	case clientpb.FileType_BINARY:
		return "binary file"
	case clientpb.FileType_TEXT:
		return "text"
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
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}
