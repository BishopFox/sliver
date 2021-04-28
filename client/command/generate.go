package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"time"

	"github.com/AlecAivazis/survey/v2"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

var (
	// SupportedCompilerTargets - Supported compiler targets
	SupportedCompilerTargets = map[string]bool{
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"linux/386":     true,
		"linux/amd64":   true,
		"windows/386":   true,
		"windows/amd64": true,
	}

	validFormats = []string{
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
)

func generate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	config := parseCompileFlags(ctx, rpc)
	if config == nil {
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	compile(config, save, rpc)
}

func regenerate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn+"Invalid implant name, see `help %s`\n", consts.RegenerateStr)
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}

	regenerate, err := rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
		ImplantName: ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"Failed to regenerate implant %s\n", err)
		return
	}
	if regenerate.File == nil {
		fmt.Printf(Warn + "Failed to regenerate implant (no data)\n")
		return
	}
	saveTo, err := saveLocation(save, regenerate.File.Name)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	err = ioutil.WriteFile(saveTo, regenerate.File.Data, 0700)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to %s\n", err)
		return
	}
	fmt.Printf(Info+"Implant binary saved to: %s\n", saveTo)
}

func saveLocation(save, defaultName string) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		log.Printf("%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			log.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0700)
			saveTo, _ = filepath.Abs(path.Join(saveTo, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			saveDir := filepath.Dir(save)
			_, err := os.Stat(saveTo)
			if os.IsNotExist(err) {
				os.MkdirAll(saveDir, 0700)
			}
			saveTo, _ = filepath.Abs(save)
		}
	} else {
		log.Printf("%s does exist\n", save)
		if fi.IsDir() {
			log.Printf("%s is dir\n", save)
			saveTo, _ = filepath.Abs(path.Join(save, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			prompt := &survey.Confirm{Message: "Overwrite existing file?"}
			var confirm bool
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return "", errors.New("File already exists")
			}
			saveTo, _ = filepath.Abs(save)
		}
	}
	return saveTo, nil
}

func generateStager(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	var stageProto clientpb.StageProtocol
	lhost := ctx.Flags.String("lhost")
	if lhost == "" {
		fmt.Println(Warn + "please specify a listening host")
		return
	}
	match, err := regexp.MatchString(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`, lhost)
	if err != nil {
		return
	}
	if !match {
		addr, err := net.LookupHost(lhost)
		if err != nil {
			fmt.Printf(Warn+"Error resolving %s: %v\n", lhost, err)
			return
		}
		if len(addr) > 1 {
			prompt := &survey.Select{
				Message: "Select an address",
				Options: addr,
			}
			err := survey.AskOne(prompt, &lhost)
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}
		} else {
			lhost = addr[0]
		}
	}
	lport := ctx.Flags.Int("lport")
	stageOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	proto := ctx.Flags.String("protocol")
	format := ctx.Flags.String("format")
	badchars := ctx.Flags.String("badchars")
	save := ctx.Flags.String("save")

	bChars := make([]string, 0)
	if len(badchars) > 0 {
		for _, b := range strings.Split(badchars, " ") {
			bChars = append(bChars, fmt.Sprintf("\\x%s", b))
		}
	}

	switch proto {
	case "tcp":
		stageProto = clientpb.StageProtocol_TCP
	case "http":
		stageProto = clientpb.StageProtocol_HTTP
	case "https":
		stageProto = clientpb.StageProtocol_HTTPS
	default:
		fmt.Printf(Warn+"%s staging protocol not supported\n", proto)
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Generating stager, please wait ...", ctrl)
	stageFile, err := rpc.MsfStage(context.Background(), &clientpb.MsfStagerReq{
		Arch:     arch,
		BadChars: bChars,
		Format:   format,
		Host:     lhost,
		Port:     uint32(lport),
		Protocol: stageProto,
		OS:       stageOS,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if save != "" || format == "raw" {
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			fmt.Printf(Warn+"Failed to generate sliver stager %v\n", err)
			return
		}
		if fi.IsDir() {
			saveTo = filepath.Join(saveTo, stageFile.GetFile().GetName())
		}
		err = ioutil.WriteFile(saveTo, stageFile.GetFile().GetData(), 0700)
		if err != nil {
			fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
			return
		}
		fmt.Printf(Info+"Sliver stager saved to: %s\n", saveTo)
	} else {
		fmt.Println(Info + "Here's your stager:")
		fmt.Println(string(stageFile.GetFile().GetData()))
	}

}

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) *clientpb.ImplantConfig {
	var name string
	if ctx.Flags["name"] != nil {
		name = strings.ToLower(ctx.Flags.String("name"))

		if name != "" {
			isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
			if !isAlphanumeric(name) {
				fmt.Printf(Warn + "Implant's name must be in alphanumeric only\n")
				return nil
			}
		}
	}

	c2s := []*clientpb.ImplantC2{}

	mtlsC2 := parseMTLSc2(ctx.Flags.String("mtls"))
	c2s = append(c2s, mtlsC2...)

	wgC2 := parseWGc2(ctx.Flags.String("wg"))
	c2s = append(c2s, wgC2...)

	httpC2 := parseHTTPc2(ctx.Flags.String("http"))
	c2s = append(c2s, httpC2...)

	dnsC2 := parseDNSc2(ctx.Flags.String("dns"))
	c2s = append(c2s, dnsC2...)

	namedPipeC2 := parseNamedPipec2(ctx.Flags.String("named-pipe"))
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2 := parseTCPPivotc2(ctx.Flags.String("tcp-pivot"))
	c2s = append(c2s, tcpPivotC2...)

	var symbolObfuscation bool
	if ctx.Flags.Bool("debug") {
		symbolObfuscation = false
	} else {
		symbolObfuscation = !ctx.Flags.Bool("skip-symbols")
	}

	if len(mtlsC2) == 0 && len(wgC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 && len(namedPipeC2) == 0 && len(tcpPivotC2) == 0 {
		fmt.Printf(Warn + "Must specify at least one of --mtls, --wg, --http, --dns, --named-pipe, or --tcp-pivot\n")
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

	reconnectInterval := ctx.Flags.Int("reconnect")
	pollInterval := ctx.Flags.Int("poll")
	maxConnectionErrors := ctx.Flags.Int("max-errors")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")
	limitFileExists := ctx.Flags.String("limit-fileexists")

	isSharedLib := false
	isService := false
	isShellcode := false

	format := ctx.Flags.String("format")
	var configFormat clientpb.ImplantConfig_OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	case "shared":
		configFormat = clientpb.ImplantConfig_SHARED_LIB
		isSharedLib = true
	case "shellcode":
		configFormat = clientpb.ImplantConfig_SHELLCODE
		isShellcode = true
	case "service":
		configFormat = clientpb.ImplantConfig_SERVICE
		isService = true
	default:
		// default to exe
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	}

	targetOS := strings.ToLower(ctx.Flags.String("os"))
	arch := strings.ToLower(ctx.Flags.String("arch"))
	targetOS, arch = getTargets(targetOS, arch)
	if targetOS == "" || arch == "" {
		return nil
	}

	if len(namedPipeC2) > 0 && targetOS != "windows" {
		fmt.Printf(Warn + "Named pipe pivoting can only be used in Windows.")
		return nil
	}

	var tunIP net.IP
	if wg := ctx.Flags.String("wg"); wg != "" {
		uniqueWGIP, err := rpc.GenerateUniqueIP(context.Background(), &commonpb.Empty{})
		tunIP = net.ParseIP(uniqueWGIP.IP)
		if err != nil {
			fmt.Println(Warn + "Failed to generate unique ip for wg peer tun interface")
			return nil
		}
		fmt.Printf(Info+"Generated unique ip for wg peer tun interface: %s\n", tunIP.String())
	}

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           arch,
		Name:             name,
		Debug:            ctx.Flags.Bool("debug"),
		Evasion:          ctx.Flags.Bool("evasion"),
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

		WGPeerTunIP:       tunIP.String(),
		WGKeyExchangePort: uint32(ctx.Flags.Int("key-exchange")),
		WGTcpCommsPort:    uint32(ctx.Flags.Int("tcp-comms")),

		ReconnectInterval:   uint32(reconnectInterval),
		PollInterval:        uint32(pollInterval),
		MaxConnectionErrors: uint32(maxConnectionErrors),

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,
		LimitFileExists:   limitFileExists,

		Format:      configFormat,
		IsSharedLib: isSharedLib,
		IsService:   isService,
		IsShellcode: isShellcode,
	}

	return config
}

func getTargets(targetOS string, targetArch string) (string, string) {

	/* For UX we convert some synonymous terms */
	if targetOS == "darwin" || targetOS == "mac" || targetOS == "macos" || targetOS == "osx" {
		targetOS = "darwin"
	}
	if targetOS == "windows" || targetOS == "win" || targetOS == "shit" {
		targetOS = "windows"
	}
	if targetOS == "linux" || targetOS == "lin" {
		targetOS = "linux"
	}

	if targetArch == "amd64" || targetArch == "x64" || strings.HasPrefix(targetArch, "64") {
		targetArch = "amd64"
	}
	if targetArch == "386" || targetArch == "x86" || strings.HasPrefix(targetArch, "32") {
		targetArch = "386"
	}

	target := fmt.Sprintf("%s/%s", targetOS, targetArch)
	if _, ok := SupportedCompilerTargets[target]; !ok {
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Unsupported compiler target %s, try to build anyways?", target),
		}
		var confirm bool
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return "", ""
		}
	}

	return targetOS, targetArch
}

func parseMTLSc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri := url.URL{Scheme: "mtls"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultMTLSLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseWGc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		uri := url.URL{Scheme: "wg"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultWGLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseHTTPc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
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
				log.Printf("Failed to parse C2 URL %s", err)
				continue
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("https://%s", arg))
			if err != nil {
				log.Printf("Failed to parse C2 URL %s", err)
				continue
			}
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseDNSc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
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
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseNamedPipec2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri, err := url.Parse("namedpipe://" + arg)
		if len(arg) < 1 {
			continue
		}
		if err != nil {
			return c2s
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseTCPPivotc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {

		uri := url.URL{Scheme: "tcppivot"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultTCPPivotPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func profileGenerate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("name")
	if name == "" && 1 <= len(ctx.Args) {
		name = ctx.Args[0]
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	profile := getImplantProfileByName(rpc, name)
	if profile != nil {
		implantFile, err := compile(profile.Config, save, rpc)
		if err != nil {
			return
		}
		profile.Config.Name = buildImplantName(implantFile.Name)
		_, err = rpc.SaveImplantProfile(context.Background(), profile)
		if err != nil {
			fmt.Printf(Warn+"could not update implant profile: %v\n", err)
			return
		}
	} else {
		fmt.Printf(Warn+"No profile with name '%s'", name)
	}
}

func compile(config *clientpb.ImplantConfig, save string, rpc rpcpb.SliverRPCClient) (*commonpb.File, error) {

	fmt.Printf(Info+"Generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)

	if config.ObfuscateSymbols {
		fmt.Printf(Info+"%sSymbol obfuscation is enabled.%s\n", bold, normal)
		fmt.Printf(Info + "This process can take awhile, and consumes significant amounts of CPU/Memory\n")
	} else if !config.Debug {
		fmt.Printf(Warn+"Symbol obfuscation is %sdisabled%s\n", bold, normal)
	}

	start := time.Now()
	ctrl := make(chan bool)
	go spin.Until("Compiling, please wait ...", ctrl)

	generated, err := rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Config: config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return nil, err
	}

	end := time.Now()
	elapsed := time.Time{}.Add(end.Sub(start))
	fmt.Printf(clearln+Info+"Build completed in %s\n", elapsed.Format("15:04:05"))
	if len(generated.File.Data) == 0 {
		fmt.Printf(Warn + "Build failed, no file data\n")
		return nil, errors.New("No file data")
	}

	saveTo, err := saveLocation(save, generated.File.Name)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(saveTo, generated.File.Data, 0700)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
		return nil, err
	}
	fmt.Printf(Info+"Implant saved to %s\n", saveTo)
	return generated.File, err
}

func profiles(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	profiles := getImplantProfiles(rpc)
	if profiles == nil {
		return
	}
	if len(profiles) == 0 {
		fmt.Printf(Info+"No profiles, create one with `%s`\n", consts.NewProfileStr)
		return
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Name\tPlatform\tCommand & Control\tDebug\tFormat\tObfuscation\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("Command & Control")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Format")),
		strings.Repeat("=", len("Obfuscation")),
		strings.Repeat("=", len("Limitations")))

	for _, profile := range profiles {
		config := profile.Config
		if 0 < len(config.C2) {
			obfuscation := "strings only"
			if config.ObfuscateSymbols {
				obfuscation = "symbols obfuscation"
			}
			if config.Debug {
				obfuscation = "none"
			}
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				profile.Name,
				fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
				fmt.Sprintf("[1] %s", config.C2[0].URL),
				fmt.Sprintf("%v", config.Debug),
				fmt.Sprintf("%v", config.Format),
				fmt.Sprintf("%s", obfuscation),
				getLimitsString(config),
			)
		}
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
					"",
					"",
					"",
					"",
				)
			}
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "", "", "", "", "", "", "")
	}
	table.Flush()
}

func getLimitsString(config *clientpb.ImplantConfig) string {
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
	if config.LimitFileExists != "" {
		limits = append(limits, fmt.Sprintf("fileexists=%s", config.LimitFileExists))
	}
	return strings.Join(limits, "; ")
}

func newProfile(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("profile-name")
	if name == "" {
		fmt.Printf(Warn + "Invalid profile name\n")
		return
	}
	config := parseCompileFlags(ctx, rpc)
	if config == nil {
		return
	}
	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"Saved new profile %s\n", resp.Name)
	}
}

func rmProfile(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn+"Invalid implant name, see `%s %s --help`\n", consts.ProfilesStr, consts.RmStr)
		return
	}
	_, err := rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
		Name: ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"Failed to delete profile %s\n", err)
		return
	}
}

func getImplantProfiles(rpc rpcpb.SliverRPCClient) []*clientpb.ImplantProfile {
	pbProfiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return nil
	}
	return pbProfiles.Profiles
}

func getImplantProfileByName(rpc rpcpb.SliverRPCClient, name string) *clientpb.ImplantProfile {
	pbProfiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return nil
	}
	for _, profile := range pbProfiles.Profiles {
		if profile.Name == name {
			return profile
		}
	}
	return nil
}

func canaries(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	canaries, err := rpc.Canaries(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Failed to list canaries %s", err)
		return
	}
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
			canary.ImplantName,
			canary.Domain,
			fmt.Sprintf("%v", canary.Triggered),
			canary.FirstTriggered,
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
