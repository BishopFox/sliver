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
	"strings"
	"text/tabwriter"
	"time"

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
	sliver := getSliver(ctx.Args[0], rpc)
	if sliver != nil {
		ActiveSliver.SetActiveSliver(sliver)
		fmt.Printf(Info+"Active sliver %s (%d)\n", sliver.Name, sliver.ID)
	} else {
		fmt.Printf(Warn+"Invalid sliver name or session number '%s'\n", ctx.Args[0])
	}
}

func background(ctx *grumble.Context, rpc RPCServer) {
	ActiveSliver.SetActiveSliver(nil)
	fmt.Printf(Info + "Background ...\n")
}

func kill(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	sliver := ActiveSliver.Sliver
	data, _ := proto.Marshal(&sliverpb.KillReq{
		SliverID: sliver.ID,
	})
	resp := rpc(&pb.Envelope{
		Type: consts.KillStr,
		Data: data,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}

	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
	} else {
		fmt.Printf(Info+"Killed %s (%d)\n", sliver.Name, sliver.ID)
	}
}

func info(ctx *grumble.Context, rpc RPCServer) {

	var sliver *pb.Sliver
	if ActiveSliver.Sliver != nil {
		sliver = ActiveSliver.Sliver
	} else if 0 < len(ctx.Args) {
		sliver = getSliver(ctx.Args[0], rpc)
	}

	if sliver != nil {
		fmt.Printf(bold+"            ID: %s%d\n", normal, sliver.ID)
		fmt.Printf(bold+"          Name: %s%s\n", normal, sliver.Name)
		fmt.Printf(bold+"      Hostname: %s%s\n", normal, sliver.Hostname)
		fmt.Printf(bold+"      Username: %s%s\n", normal, sliver.Username)
		fmt.Printf(bold+"           UID: %s%s\n", normal, sliver.UID)
		fmt.Printf(bold+"           GID: %s%s\n", normal, sliver.GID)
		fmt.Printf(bold+"           PID: %s%d\n", normal, sliver.PID)
		fmt.Printf(bold+"            OS: %s%s\n", normal, sliver.OS)
		fmt.Printf(bold+"          Arch: %s%s\n", normal, sliver.Arch)
		fmt.Printf(bold+"Remote Address: %s%s\n", normal, sliver.RemoteAddress)
	} else {
		fmt.Printf(Warn+"No target sliver, see `help %s`\n", consts.InfoStr)
	}
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
	}, 1200*time.Second)
	ctrl <- true
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}
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

func profileGenerate(ctx *grumble.Context, rpc RPCServer) {

}

func profiles(ctx *grumble.Context, rpc RPCServer) {
	profiles := getSliverProfiles(rpc)
	if profiles == nil {
		return
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Name\tPlatform\tmTLS\tDNS\tDebug\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("mTLS")),
		strings.Repeat("=", len("DNS")),
		strings.Repeat("=", len("Debug")))
	for name, profile := range *profiles {
		config := profile.Config
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			name,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			fmt.Sprintf("%s:%d", config.MTLSServer, config.MTLSLPort),
			config.DNSParent,
			fmt.Sprintf("%v", config.Debug),
		)
	}
	table.Flush()
}

func newProfile(ctx *grumble.Context, rpc RPCServer) {
	name := ctx.Flags.String("name")

	targetOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	debug := ctx.Flags.Bool("debug")
	dnsParent := ctx.Flags.String("dns")

	data, _ := proto.Marshal(&pb.Profile{
		Name: name,
		Config: &pb.SliverConfig{
			GOOS:       targetOS,
			GOARCH:     arch,
			MTLSServer: lhost,
			MTLSLPort:  int32(lport),
			Debug:      debug,
			DNSParent:  dnsParent,
		},
	})

	resp := rpc(&pb.Envelope{
		Type: consts.NewProfileStr,
		Data: data,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}
	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
	} else {
		fmt.Printf(Info + "Saved new profile\n")
	}
}

func getSliverProfiles(rpc RPCServer) *map[string]*pb.Profile {
	resp := rpc(&pb.Envelope{
		Type: consts.ProfilesStr,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return nil
	}
	if resp.Error != "" {
		fmt.Printf(Warn+"%s\n", resp.Error)
		return nil
	}

	pbProfiles := &pb.Profiles{}
	err := proto.Unmarshal(resp.Data, pbProfiles)
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return nil
	}

	profiles := &map[string]*pb.Profile{}
	for _, profile := range pbProfiles.List {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}

func ping(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

}

func getPID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%d\n", ActiveSliver.Sliver.PID)
}

func getUID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.UID)
}

func getGID(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.GID)
}

func whoami(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSliver.Sliver.Username)
}
