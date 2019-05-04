package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	consts "sliver/client/constants"
	"sliver/client/spin"
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

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

func regenerate(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn+"Invalid sliver name, see 'help %s'\n", consts.RegenerateStr)
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}

	regenerateReq, _ := proto.Marshal(&clientpb.Regenerate{
		SliverName: ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgRegenerate,
		Data: regenerateReq,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	regen := &clientpb.Regenerate{}
	proto.Unmarshal(resp.Data, regen)

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to regenerate sliver %s\n", err)
		return
	}
	if regen.File == nil {
		fmt.Printf(Warn + "Failed to regenerate sliver (no data)\n")
		return
	}

	if fi.IsDir() {
		var fileName string
		if 0 < len(regen.File.Name) {
			fileName = path.Base(regen.File.Name)
		} else {
			fileName = path.Base(ctx.Args[0])
		}
		saveTo = filepath.Join(saveTo, fileName)
	}
	err = ioutil.WriteFile(saveTo, regen.File.Data, os.ModePerm)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to %s\n", err)
		return
	}
	fmt.Printf(Info+"Sliver binary saved to: %s\n", saveTo)
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

	canaries := ctx.Flags.String("canary")
	canaryDomains := []string{}
	if 0 < len(canaries) {
		canaryDomains = strings.Split(canaries, ",")
	}

	reconnectInverval := ctx.Flags.Int("reconnect")
	maxConnectionErrors := ctx.Flags.Int("max-errors")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")

	isSharedLib := false

	format := ctx.Flags.String("format")
	var configFormat clientpb.SliverConfig_OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.SliverConfig_EXECUTABLE
	case "shared":
		configFormat = clientpb.SliverConfig_SHARED_LIB
		isSharedLib = true
	case "shellcode":
		configFormat = clientpb.SliverConfig_SHELLCODE
		isSharedLib = true
	default:
		// default to exe
		configFormat = clientpb.SliverConfig_EXECUTABLE
	}
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
		GOOS:          targetOS,
		GOARCH:        arch,
		Debug:         debug,
		C2:            c2s,
		CanaryDomains: canaryDomains,

		ReconnectInterval:   uint32(reconnectInverval),
		MaxConnectionErrors: uint32(maxConnectionErrors),

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,

		Format:      configFormat,
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
			uri = &url.URL{Scheme: "https"} // HTTPS is the default, will fallback to HTTP
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
		saveTo = filepath.Join(saveTo, path.Base(generated.File.Name))
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
	fmt.Fprintf(table, "Name\tPlatform\tCommand & Control\tDebug\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("Command & Control")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Limits")))

	for name, profile := range *profiles {
		config := profile.Config
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			name,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			fmt.Sprintf("[1] %s", config.C2[0].URL),
			fmt.Sprintf("%v", config.Debug),
			getLimitsString(config),
		)
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
					"",
					"",
				)
			}
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", "", "", "", "", "")
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

func canaries(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgListCanaries,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	canaries := &clientpb.Canaries{}
	proto.Unmarshal(resp.Data, canaries)
	for i, canary := range canaries.Canaries {
		fmt.Printf("%d. %v\n", i, canary)
	}

}
