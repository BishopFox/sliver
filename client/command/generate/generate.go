package generate

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
	"runtime"
	"strings"

	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

const (
	// DefaultMTLSLPort is the default port for mtls
	DefaultMTLSLPort = 8888
	// DefaultWGPort is the default port for wg
	DefaultWGLPort = 53
	// DefaultWGNPort is the default n port for wg
	DefaultWGNPort = 8888
	// DefaultWGKeyExPort is the default port for wg key exchange
	DefaultWGKeyExPort = 1337
	// DefaultHTTPLPort is the default port for http
	DefaultHTTPLPort = 80
	// DefaultHTTPSLPort is the default port for https
	DefaultHTTPSLPort = 443
	// DefaultDNSLPortis the default port for dns
	DefaultDNSLPort = 53
	// DefaultTCPPivotPort is the default port for tcp pivots
	DefaultTCPPivotPort = 9898

	// DefaultReconnect is the default reconnect time
	DefaultReconnect = 60
	// DefaultPollTimeout is the default poll timeout
	DefaultPollTimeout = 360 // 6 minutes
	// DefaultMaxErrors is the default max reconnection errors before giving up
	DefaultMaxErrors = 1000
)

const (
	crossCompilerInfoURL = "https://github.com/BishopFox/sliver/wiki/Cross-Compiling-Implants"
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
)

// GenerateCmd - The main command used to generate implant binaries
func GenerateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	config := parseCompileFlags(ctx, con)
	if config == nil {
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	compile(config, save, con)
}

func expandPath(path string) string {
	// unless path starts with ~
	if len(path) == 0 || path[0] != 126 {
		return path
	}

	return filepath.Join(os.Getenv("HOME"), path[1:])
}

func saveLocation(save, DefaultName string) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	save = expandPath(save)
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		log.Printf("%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			log.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0700)
			saveTo, _ = filepath.Abs(path.Join(saveTo, DefaultName))
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
			saveTo, _ = filepath.Abs(path.Join(save, DefaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			prompt := &survey.Confirm{Message: "Overwrite existing file?"}
			var confirm bool
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return "", errors.New("file already exists")
			}
			saveTo, _ = filepath.Abs(save)
		}
	}
	return saveTo, nil
}

func nameOfOutputFormat(value clientpb.OutputFormat) string {
	switch value {
	case clientpb.OutputFormat_EXECUTABLE:
		return "Executable"
	case clientpb.OutputFormat_SERVICE:
		return "Service"
	case clientpb.OutputFormat_SHARED_LIB:
		return "Shared Library"
	case clientpb.OutputFormat_SHELLCODE:
		return "Shellcode"
	}
	panic(fmt.Sprintf("Unknown format %v", value))
}

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(ctx *grumble.Context, con *console.SliverConsoleClient) *clientpb.ImplantConfig {
	var name string
	if ctx.Flags["name"] != nil {
		name = strings.ToLower(ctx.Flags.String("name"))

		if name != "" {
			isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
			if !isAlphanumeric(name) {
				con.PrintErrorf("Implant's name must be in alphanumeric only\n")
				return nil
			}
		}
	}

	c2s := []*clientpb.ImplantC2{}

	mtlsC2, err := ParseMTLSc2(ctx.Flags.String("mtls"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, mtlsC2...)

	wgC2, err := ParseWGc2(ctx.Flags.String("wg"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, wgC2...)

	httpC2, err := ParseHTTPc2(ctx.Flags.String("http"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, httpC2...)

	dnsC2, err := ParseDNSc2(ctx.Flags.String("dns"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, dnsC2...)

	namedPipeC2, err := ParseNamedPipec2(ctx.Flags.String("named-pipe"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2, err := ParseTCPPivotc2(ctx.Flags.String("tcp-pivot"))
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return nil
	}
	c2s = append(c2s, tcpPivotC2...)

	var symbolObfuscation bool
	if ctx.Flags.Bool("debug") {
		symbolObfuscation = false
	} else {
		symbolObfuscation = !ctx.Flags.Bool("skip-symbols")
	}

	if len(mtlsC2) == 0 && len(wgC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 && len(namedPipeC2) == 0 && len(tcpPivotC2) == 0 {
		con.PrintErrorf("Must specify at least one of --mtls, --wg, --http, --dns, --named-pipe, or --tcp-pivot\n")
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
	pollTimeout := ctx.Flags.Int("poll-timeout")
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
	var configFormat clientpb.OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.OutputFormat_EXECUTABLE
	case "shared":
		configFormat = clientpb.OutputFormat_SHARED_LIB
		isSharedLib = true
	case "shellcode":
		configFormat = clientpb.OutputFormat_SHELLCODE
		isShellcode = true
	case "service":
		configFormat = clientpb.OutputFormat_SERVICE
		isService = true
	default:
		// Default to exe
		configFormat = clientpb.OutputFormat_EXECUTABLE
	}

	targetOS := strings.ToLower(ctx.Flags.String("os"))
	targetArch := strings.ToLower(ctx.Flags.String("arch"))
	targetOS, targetArch = getTargets(targetOS, targetArch, con)
	if targetOS == "" || targetArch == "" {
		return nil
	}
	if len(namedPipeC2) > 0 && targetOS != "windows" {
		con.PrintErrorf("Named pipe pivoting can only be used in Windows.")
		return nil
	}

	// Check to see if we can *probably* build the target binary
	if !checkBuildTargetCompatibility(configFormat, targetOS, targetArch, con) {
		return nil
	}

	var tunIP net.IP
	if wg := ctx.Flags.String("wg"); wg != "" {
		uniqueWGIP, err := con.Rpc.GenerateUniqueIP(context.Background(), &commonpb.Empty{})
		tunIP = net.ParseIP(uniqueWGIP.IP)
		if err != nil {
			con.PrintErrorf("Failed to generate unique ip for wg peer tun interface")
			return nil
		}
		con.PrintInfof("Generated unique ip for wg peer tun interface: %s\n", tunIP.String())
	}

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           targetArch,
		Name:             name,
		Debug:            ctx.Flags.Bool("debug"),
		Evasion:          ctx.Flags.Bool("evasion"),
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

		WGPeerTunIP:       tunIP.String(),
		WGKeyExchangePort: uint32(ctx.Flags.Int("key-exchange")),
		WGTcpCommsPort:    uint32(ctx.Flags.Int("tcp-comms")),

		ReconnectInterval:   int64(reconnectInterval) * int64(time.Second),
		PollTimeout:         int64(pollTimeout) * int64(time.Second),
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

func getTargets(targetOS string, targetArch string, con *console.SliverConsoleClient) (string, string) {

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
		con.Printf("⚠️  Unsupported compiler target %s%s%s, but we can try to compile a generic implant.\n",
			console.Bold, target, console.Normal,
		)
		con.Printf("⚠️  Generic implants do not support all commands/features.\n")
		prompt := &survey.Confirm{Message: "Attempt to build generic implant?"}
		var confirm bool
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return "", ""
		}
	}

	return targetOS, targetArch
}

// ParseMTLSc2 - Parse mtls connection string arg
func ParseMTLSc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "mtls://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("mtls://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Hostname(), DefaultMTLSLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseWGc2 - Parse wg connect string arg
func ParseWGc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "mtls://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("mtls://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Hostname(), DefaultWGLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseHTTPc2 - Parse HTTP connection string arg
func ParseHTTPc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("https://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseDNSc2 - Parse DNS connection string arg
func ParseDNSc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "dns://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("dns://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseNamedPipec2 - Parse named pipe connection string arg
func ParseNamedPipec2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "namedpipe://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("namedpipe://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseTCPPivotc2 - Parse tcp pivot connection string arg
func ParseTCPPivotc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "tcp-pivot://") {
			arg = strings.Replace(arg, "tcp-pivot://", "tcppivot://", 1)
		}
		if strings.HasPrefix(arg, "tcppivot://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("tcppivot://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Hostname(), DefaultTCPPivotPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

func compile(config *clientpb.ImplantConfig, save string, con *console.SliverConsoleClient) (*commonpb.File, error) {
	if config.IsBeacon {
		interval := time.Duration(config.BeaconInterval)
		con.PrintInfof("Generating new %s/%s beacon implant binary (%v)\n", config.GOOS, config.GOARCH, interval)
	} else {
		con.PrintInfof("Generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)
	}
	if config.ObfuscateSymbols {
		con.PrintInfof("%sSymbol obfuscation is enabled%s\n", console.Bold, console.Normal)
	} else if !config.Debug {
		con.PrintErrorf("Symbol obfuscation is %sdisabled%s\n", console.Bold, console.Normal)
	}

	start := time.Now()
	ctrl := make(chan bool)
	con.SpinUntil("Compiling, please wait ...", ctrl)

	generated, err := con.Rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Config: config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil, err
	}

	end := time.Now()
	elapsed := time.Time{}.Add(end.Sub(start))
	con.PrintInfof("Build completed in %s\n", elapsed.Format("15:04:05"))
	if len(generated.File.Data) == 0 {
		con.PrintErrorf("Build failed, no file data\n")
		return nil, errors.New("no file data")
	}

	saveTo, err := saveLocation(save, generated.File.Name)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(saveTo, generated.File.Data, 0700)
	if err != nil {
		con.PrintErrorf("Failed to write to: %s\n", saveTo)
		return nil, err
	}
	con.PrintInfof("Implant saved to %s\n", saveTo)
	return generated.File, err
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

func checkBuildTargetCompatibility(format clientpb.OutputFormat, targetOS string, targetArch string, con *console.SliverConsoleClient) bool {
	if format == clientpb.OutputFormat_EXECUTABLE {
		return true // We don't need cross-compilers when targeting EXECUTABLE formats
	}

	compilers, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to check target compatibility: %s\n", err)
		return true
	}

	if runtime.GOOS != "windows" && targetOS == "windows" {
		if !hasCC(targetOS, targetArch, compilers.CrossCompilers) {
			return warnMissingCrossCompiler(format, targetOS, targetArch, con)
		}
	}

	if runtime.GOOS != "darwin" && targetOS == "darwin" {
		if !hasCC(targetOS, targetArch, compilers.CrossCompilers) {
			return warnMissingCrossCompiler(format, targetOS, targetArch, con)
		}
	}

	if runtime.GOOS != "linux" && targetOS == "linux" {
		if !hasCC(targetOS, targetArch, compilers.CrossCompilers) {
			return warnMissingCrossCompiler(format, targetOS, targetArch, con)
		}
	}

	return true
}

func hasCC(targetOS string, targetArch string, crossCompilers []*clientpb.CrossCompiler) bool {
	for _, cc := range crossCompilers {
		if cc.GetTargetGOOS() == targetOS && cc.GetTargetGOARCH() == targetArch {
			return true
		}
	}
	return false
}

func warnMissingCrossCompiler(format clientpb.OutputFormat, targetOS string, targetArch string, con *console.SliverConsoleClient) bool {
	con.PrintWarnf("Missing cross-compiler for %s on %s/%s\n", nameOfOutputFormat(format), targetOS, targetArch)
	switch targetOS {
	case "windows":
		con.PrintWarnf("The server cannot find an installation of mingw")
	case "darwin":
		con.PrintWarnf("The server cannot find an installation of osxcross")
	case "linux":
		con.PrintWarnf("The server cannot find an installation of musl-cross")
	}
	con.PrintWarnf("For more information please read %s\n", crossCompilerInfoURL)

	confirm := false
	prompt := &survey.Confirm{Message: "Try to compile anyways (will likely fail)?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
