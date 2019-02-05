package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	consts "sliver/client/constants"
	"sliver/client/spin"
	pb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}
)

func sessions(ctx *grumble.Context, rpc RPCServer) {
	resp := rpc(&pb.Envelope{
		Type: consts.SessionsStr,
		Data: []byte{},
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "Command timeout\n")
		return
	}
	sessions := &pb.Sessions{}
	proto.Unmarshal(resp.Data, sessions)

	slivers := map[int32]*pb.Sliver{}
	for _, sliver := range sessions.Slivers {
		slivers[sliver.ID] = sliver
	}
	if 0 < len(slivers) {
		printSlivers(slivers)
	} else {
		fmt.Printf(Info + "No slivers connected\n")
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
func printSlivers(sessions map[int32]*pb.Sliver) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "ID\tName\tTransport\tRemote Address\tUsername\tOperating System\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Transport")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")))

	// Sort the keys becuase maps have a randomized order
	var keys []int
	for _, sliver := range sessions {
		keys = append(keys, int(sliver.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	activeIndex := -1
	for index, key := range keys {
		sliver := sessions[int32(key)]
		if ActiveSliver.Sliver != nil && ActiveSliver.Sliver.ID == sliver.ID {
			activeIndex = index + 2 // Two lines for the headers
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\t\n",
			sliver.ID, sliver.Name, sliver.Transport, sliver.RemoteAddress, sliver.Username,
			fmt.Sprintf("%s/%s", sliver.OS, sliver.Arch))
	}
	table.Flush()

	if activeIndex != -1 {
		lines := strings.Split(outputBuf.String(), "\n")
		for lineNumber, line := range lines {
			if len(line) == 0 {
				continue
			}
			if lineNumber == activeIndex {
				fmt.Printf("%s%s%s\n", green, line, normal)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		fmt.Printf(outputBuf.String())
	}
}

func use(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing sliver name or session number, see `help use`\n")
		return
	}
	resp := rpc(&pb.Envelope{
		Type: consts.SessionsStr,
		Data: []byte{},
	}, defaultTimeout)
	sessions := &pb.Sessions{}
	proto.Unmarshal(resp.Data, sessions)

	for _, sliver := range sessions.Slivers {
		if strconv.Itoa(int(sliver.ID)) == ctx.Args[0] || sliver.Name == ctx.Args[0] {
			ActiveSliver.SetActiveSliver(sliver)
			fmt.Printf(Info+"Active sliver: %s (%d)\n", sliver.Name, sliver.ID)
			return
		}
	}
	fmt.Printf(Warn+"Invalid sliver name or session number '%s'\n", ctx.Args[0])
}

func background(ctx *grumble.Context, rpc RPCServer) {
	ActiveSliver.SetActiveSliver(nil)
	fmt.Printf(Info + "Background ...\n")
}

func kill(ctx *grumble.Context, rpc RPCServer) {

}

func info(ctx *grumble.Context, rpc RPCServer) {

}

func generate(ctx *grumble.Context, rpc RPCServer) {
	targetOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	debug := ctx.Flags.Bool("debug")
	dnsParent := ctx.Flags.String("dns")
	save := ctx.Flags.String("save")

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s'\n", lhost)
		return
	}
	if save == "" {
		fmt.Printf(Warn + "Save path required (--save)\n")
		return
	}

	// Make sure we have the FQDN
	if dnsParent != "" && !strings.HasSuffix(dnsParent, ".") {
		dnsParent += "."
	}
	if dnsParent != "" && strings.HasPrefix(dnsParent, ".") {
		dnsParent = dnsParent[1:]
	}

	fmt.Printf(Info+"Generating new %s/%s sliver binary \n", targetOS, arch)
	ctrl := make(chan bool)
	go spin.Until("Compiling ...", ctrl)
	generateReq, _ := proto.Marshal(&pb.GenerateReq{
		OS:        targetOS,
		Arch:      arch,
		LHost:     lhost,
		LPort:     int32(lport),
		Debug:     debug,
		DNSParent: dnsParent,
	})

	resp := rpc(&pb.Envelope{
		Type: consts.GenerateStr,
		Data: generateReq,
	}, defaultTimeout)
	ctrl <- true
	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
		return
	}

	generated := &pb.Generate{}
	proto.Unmarshal(resp.Data, generated)

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate sliver %v\n\n", err)
		return
	}
	if fi.IsDir() {
		saveTo = filepath.Join(saveTo, generated.File.Name)
	}
	err = ioutil.WriteFile(saveTo, generated.File.Data, os.ModePerm)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
		return
	}
	fmt.Printf(Info+"Sliver binary saved to: %s\n", saveTo)
}

func ping(ctx *grumble.Context, rpc RPCServer) {

}

func getPID(ctx *grumble.Context, rpc RPCServer) {

}

func getUID(ctx *grumble.Context, rpc RPCServer) {

}

func getGID(ctx *grumble.Context, rpc RPCServer) {

}

func whoami(ctx *grumble.Context, rpc RPCServer) {

}

func ps(ctx *grumble.Context, rpc RPCServer) {
	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	if ActiveSliver.Sliver == nil {
		fmt.Println(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.PsReq{SliverID: ActiveSliver.Sliver.ID})
	resp := rpc(&pb.Envelope{
		Type: consts.PsStr,
		Data: data,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}
	ps := &sliverpb.Ps{}
	err := proto.Unmarshal(resp.Data, ps)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
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
	for _, proc := range ps.Processes {
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

	for index, line := range strings.Split(outputBuf.String(), "\n") {
		if len(line) == 0 {
			continue
		}
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}

}

// printProcInfo - Stylizes the process information
func printProcInfo(table *tabwriter.Writer, proc *sliverpb.Process) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if ActiveSliver.Sliver != nil && proc.Pid == ActiveSliver.Sliver.PID {
		color = green
	}
	fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner)
	return color
}

func procdump(ctx *grumble.Context, rpc RPCServer) {

}
