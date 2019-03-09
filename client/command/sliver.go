package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	consts "sliver/client/constants"
	"sliver/client/spin"
	clientpb "sliver/protobuf/client"
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
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgSessions,
		Data: []byte{},
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "Command timeout\n")
		return
	}
	sessions := &clientpb.Sessions{}
	proto.Unmarshal(resp.Data, sessions)

	slivers := map[uint32]*clientpb.Sliver{}
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
func printSlivers(sessions map[uint32]*clientpb.Sliver) {
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
		sliver := sessions[uint32(key)]
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
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgKill,
		Data: data,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}

	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf(Info+"Killed %s (%d)\n", sliver.Name, sliver.ID)
	}
}

func info(ctx *grumble.Context, rpc RPCServer) {

	var sliver *clientpb.Sliver
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
	targetOS := strings.ToLower(ctx.Flags.String("os"))
	arch := strings.ToLower(ctx.Flags.String("arch"))

	debug := ctx.Flags.Bool("debug")

	mtlsServer := ctx.Flags.String("mtls")
	mtlsLPort := ctx.Flags.Int("mtls-lport")

	httpServer := ctx.Flags.String("http")
	httpLPort := ctx.Flags.Int("http-lport")
	noVerify := ctx.Flags.Bool("no-verify")

	dnsParent := ctx.Flags.String("dns")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")

	save := ctx.Flags.String("save")

	/* For UX we convert some synonymous terms */
	if targetOS == "mac" || targetOS == "macos" || targetOS == "m" {
		targetOS = "darwin"
	}
	if targetOS == "win" || targetOS == "w" || targetOS == "shit" {
		targetOS = "windows"
	}
	if targetOS == "unix" || targetOS == "l" {
		targetOS = "linux"
	}
	if arch == "x64" || strings.HasPrefix(arch, "64") {
		arch = "amd64"
	}
	if arch == "x86" || strings.HasPrefix(arch, "32") {
		arch = "386"
	}

	if mtlsServer == "" && httpServer == "" && dnsParent == "" {
		fmt.Printf(Warn + "Must specify --mtls, --http, or --dns\n")
		return
	}
	if save == "" {
		save, _ = os.Getwd()
	}

	// Make sure we have the FQDN
	if dnsParent != "" && !strings.HasSuffix(dnsParent, ".") {
		dnsParent += "."
	}
	if dnsParent != "" && strings.HasPrefix(dnsParent, ".") {
		dnsParent = dnsParent[1:]
	}
	compile(&clientpb.SliverConfig{
		GOOS:   targetOS,
		GOARCH: arch,
		Debug:  debug,

		MTLSServer: mtlsServer,
		MTLSLPort:  int32(mtlsLPort),

		HTTPServer: httpServer,
		HTTPLPort:  int32(httpLPort),
		NoVerify:   noVerify,

		DNSParent: dnsParent,

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,
	}, save, rpc)
}

func profileGenerate(ctx *grumble.Context, rpc RPCServer) {
	name := ctx.Flags.String("name")
	if name == "" && 1 <= len(ctx.Args) {
		name = ctx.Args[0]
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	profiles := getSliverProfiles(rpc)
	if profile, ok := (*profiles)[name]; ok {
		compile(profile.Config, save, rpc)
	} else {
		fmt.Printf(Warn+"No profile with name '%s'", name)
	}
}

func compile(config *clientpb.SliverConfig, save string, rpc RPCServer) {

	fmt.Printf(Info+"Generating new %s/%s sliver binary \n", config.GOOS, config.GOARCH)
	ctrl := make(chan bool)
	go spin.Until("Compiling ...", ctrl)

	generateReq, _ := proto.Marshal(&clientpb.GenerateReq{Config: config})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgGenerate,
		Data: generateReq,
	}, 1200*time.Second) // TODO: make timeout a parameter
	ctrl <- true
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	generated := &clientpb.Generate{}
	proto.Unmarshal(resp.Data, generated)

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate sliver %v\n", err)
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

func profiles(ctx *grumble.Context, rpc RPCServer) {
	profiles := getSliverProfiles(rpc)
	if profiles == nil {
		return
	}
	if len(*profiles) == 0 {
		fmt.Printf(Info+"No profiles, create one with `%s`\n", consts.NewProfileStr)
		return
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Name\tPlatform\tmTLS\tHTTP\tDNS\tDebug\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("mTLS")),
		strings.Repeat("=", len("HTTP")),
		strings.Repeat("=", len("DNS")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Limitations")))
	for name, profile := range *profiles {
		config := profile.Config
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			name,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			fmt.Sprintf("%s:%d", config.MTLSServer, config.MTLSLPort),
			fmt.Sprintf("%s:%d", config.HTTPServer, config.HTTPLPort),
			config.DNSParent,
			fmt.Sprintf("%v", config.Debug),
			getLimitsString(config),
		)
	}
	table.Flush()
}

func getLimitsString(config *clientpb.SliverConfig) string {
	limits := []string{}
	if config.LimitDatetime != "" {
		limits = append(limits, fmt.Sprintf("datetime=%s", config.LimitDatetime))
	}
	if config.LimitDomainJoined {
		limits = append(limits, fmt.Sprintf("domainjoined=%v", config.LimitDomainJoined))
	}
	if config.LimitUsername != "" {
		limits = append(limits, fmt.Sprintf("username=%s", config.LimitUsername))
	}
	if config.LimitHostname != "" {
		limits = append(limits, fmt.Sprintf("hostname=%s", config.LimitHostname))
	}
	return strings.Join(limits, "; ")
}

func newProfile(ctx *grumble.Context, rpc RPCServer) {
	name := ctx.Flags.String("name")
	if name == "" {
		fmt.Printf(Warn + "Invalid profile name\n")
		return
	}

	targetOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	mtlsServer := ctx.Flags.String("mtls")
	mtlsLPort := ctx.Flags.Int("mtls-lport")

	httpServer := ctx.Flags.String("http")
	httpLPort := ctx.Flags.Int("http-lport")
	noVerify := ctx.Flags.Bool("no-verify")

	debug := ctx.Flags.Bool("debug")
	dnsParent := ctx.Flags.String("dns")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")

	/* For UX we convert some synonymous terms */
	if targetOS == "mac" || targetOS == "macos" || targetOS == "m" {
		targetOS = "darwin"
	}
	if targetOS == "win" || targetOS == "w" || targetOS == "shit" {
		targetOS = "windows"
	}
	if targetOS == "unix" || targetOS == "l" {
		targetOS = "linux"
	}
	if arch == "x64" || strings.HasPrefix(arch, "64") {
		arch = "amd64"
	}
	if arch == "x86" || strings.HasPrefix(arch, "32") {
		arch = "386"
	}

	data, _ := proto.Marshal(&clientpb.Profile{
		Name: name,
		Config: &clientpb.SliverConfig{
			GOOS:   targetOS,
			GOARCH: arch,

			Debug: debug,

			MTLSServer: mtlsServer,
			MTLSLPort:  int32(mtlsLPort),
			HTTPServer: httpServer,
			HTTPLPort:  int32(httpLPort),
			NoVerify:   noVerify,
			DNSParent:  dnsParent,

			LimitDomainJoined: limitDomainJoined,
			LimitHostname:     limitHostname,
			LimitUsername:     limitUsername,
			LimitDatetime:     limitDatetime,
		},
	})

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgNewProfile,
		Data: data,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return
	}
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf(Info + "Saved new profile\n")
	}
}

func getSliverProfiles(rpc RPCServer) *map[string]*clientpb.Profile {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgProfiles,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "No response from server\n")
		return nil
	}
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return nil
	}

	pbProfiles := &clientpb.Profiles{}
	err := proto.Unmarshal(resp.Data, pbProfiles)
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return nil
	}

	profiles := &map[string]*clientpb.Profile{}
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
