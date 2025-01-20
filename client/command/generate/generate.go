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
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
)

const (
	// DefaultMTLSLPort is the default port for mtls.
	DefaultMTLSLPort = 8888
	// DefaultWGPort is the default port for wg.
	DefaultWGLPort = 53
	// DefaultWGNPort is the default n port for wg.
	DefaultWGNPort = 8888
	// DefaultWGKeyExPort is the default port for wg key exchange.
	DefaultWGKeyExPort = 1337
	// DefaultHTTPLPort is the default port for http.
	DefaultHTTPLPort = 80
	// DefaultHTTPSLPort is the default port for https.
	DefaultHTTPSLPort = 443
	// DefaultDNSLPortis the default port for dns.
	DefaultDNSLPort = 53
	// DefaultTCPPivotPort is the default port for tcp pivots.
	DefaultTCPPivotPort = 9898

	// DefaultReconnect is the default reconnect time.
	DefaultReconnect = 60
	// DefaultPollTimeout is the default poll timeout.
	DefaultPollTimeout = 360 // 6 minutes
	// DefaultMaxErrors is the default max reconnection errors before giving up.
	DefaultMaxErrors = 1000
)

const (
	crossCompilerInfoURL = "https://github.com/BishopFox/sliver/wiki/Cross-Compiling-Implants"
)

var (
	// SupportedCompilerTargets - Supported compiler targets.
	SupportedCompilerTargets = map[string]bool{
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"linux/386":     true,
		"linux/amd64":   true,
		"windows/386":   true,
		"windows/amd64": true,
	}

	ErrNoExternalBuilder = errors.New("no external builders are available")
	ErrNoValidBuilders   = errors.New("no valid external builders for target")
)

// GenerateCmd - The main command used to generate implant binaries
func GenerateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if external, _ := cmd.Flags().GetBool("external-builder"); !external {
		compile(name, config, save, con)
	} else {
		_, err := externalBuild(name, config, save, con)
		if err != nil {
			if err == ErrNoExternalBuilder {
				con.PrintErrorf("There are no external builders currently connected to the server\n")
				con.PrintErrorf("See 'builders' command for more information\n")
			} else if err == ErrNoValidBuilders {
				con.PrintErrorf("There are external builders connected to the server, but none can build the target you specified\n")
				con.PrintErrorf("Invalid target %s\n", fmt.Sprintf("%s:%s/%s", config.Format, config.GOOS, config.GOARCH))
				con.PrintErrorf("See 'builders' command for more information\n")
			} else {
				con.PrintErrorf("%s\n", err)
			}
			return
		}
	}
}

func expandPath(path string) string {
	// unless path starts with ~
	if len(path) == 0 || path[0] != 126 {
		return path
	}

	return filepath.Join(os.Getenv("HOME"), path[1:])
}

func saveLocation(save, DefaultName string, con *console.SliverClient) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	save = expandPath(save)
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		con.Printf("%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			con.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0o700)
			saveTo, _ = filepath.Abs(filepath.Join(saveTo, DefaultName))
		} else {
			con.Printf("%s is not dir\n", save)
			saveDir := filepath.Dir(save)
			_, err := os.Stat(saveTo)
			if os.IsNotExist(err) {
				os.MkdirAll(saveDir, 0o700)
			}
			saveTo, _ = filepath.Abs(save)
		}
	} else {
		if fi.IsDir() {
			saveTo, _ = filepath.Abs(filepath.Join(save, DefaultName))
		} else {
			con.PrintInfof("%s is not dir\n", save)
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
	default:
		return "Unknown"
	}
}

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(cmd *cobra.Command, con *console.SliverClient) (string, *clientpb.ImplantConfig) {
	var name string
	if nameF, _ := cmd.Flags().GetString("name"); nameF != "" {
		name = strings.ToLower(nameF)

		if err := util.AllowedName(name); err != nil {
			con.PrintErrorf("%s\n", err)
			return "", nil
		}
	}

	c2s := []*clientpb.ImplantC2{}

	mtlsC2F, _ := cmd.Flags().GetString("mtls")
	mtlsC2, err := ParseMTLSc2(mtlsC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	c2s = append(c2s, mtlsC2...)

	wgC2F, _ := cmd.Flags().GetString("wg")
	wgC2, err := ParseWGc2(wgC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	wgKeyExchangePort, _ := cmd.Flags().GetUint32("key-exchange")
	wgTcpCommsPort, _ := cmd.Flags().GetUint32("tcp-comms")

	c2s = append(c2s, wgC2...)

	httpC2F, _ := cmd.Flags().GetString("http")
	httpC2, err := ParseHTTPc2(httpC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	c2s = append(c2s, httpC2...)

	dnsC2F, _ := cmd.Flags().GetString("dns")
	dnsC2, err := ParseDNSc2(dnsC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	c2s = append(c2s, dnsC2...)

	namedPipeC2F, _ := cmd.Flags().GetString("named-pipe")
	namedPipeC2, err := ParseNamedPipec2(namedPipeC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2F, _ := cmd.Flags().GetString("tcp-pivot")
	tcpPivotC2, err := ParseTCPPivotc2(tcpPivotC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return "", nil
	}
	c2s = append(c2s, tcpPivotC2...)

	var symbolObfuscation bool
	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		symbolObfuscation = false
	} else {
		symbolObfuscation, _ = cmd.Flags().GetBool("skip-symbols")
		symbolObfuscation = !symbolObfuscation
	}

	if len(mtlsC2) == 0 && len(wgC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 && len(namedPipeC2) == 0 && len(tcpPivotC2) == 0 {
		con.PrintErrorf("Must specify at least one of --mtls, --wg, --http, --dns, --named-pipe, or --tcp-pivot\n")
		return "", nil
	}

	rawCanaries, _ := cmd.Flags().GetString("canary")
	canaryDomains := []string{}
	if 0 < len(rawCanaries) {
		for _, canaryDomain := range strings.Split(rawCanaries, ",") {
			if !strings.HasSuffix(canaryDomain, ".") {
				canaryDomain += "." // Ensure we have the FQDN
			}
			canaryDomains = append(canaryDomains, canaryDomain)
		}
	}

	debug, _ := cmd.Flags().GetBool("debug")
	evasion, _ := cmd.Flags().GetBool("evasion")
	templateName, _ := cmd.Flags().GetString("template")

	reconnectInterval, _ := cmd.Flags().GetInt64("reconnect")
	pollTimeout, _ := cmd.Flags().GetInt64("poll-timeout")
	maxConnectionErrors, _ := cmd.Flags().GetUint32("max-errors")

	limitDomainJoined, _ := cmd.Flags().GetBool("limit-domainjoined")
	limitHostname, _ := cmd.Flags().GetString("limit-hostname")
	limitUsername, _ := cmd.Flags().GetString("limit-username")
	limitDatetime, _ := cmd.Flags().GetString("limit-datetime")
	limitFileExists, _ := cmd.Flags().GetString("limit-fileexists")
	limitLocale, _ := cmd.Flags().GetString("limit-locale")
	debugFile, _ := cmd.Flags().GetString("debug-file")

	isSharedLib := false
	isService := false
	isShellcode := false
	sgnEnabled := false

	format, _ := cmd.Flags().GetString("format")
	runAtLoad := false
	var configFormat clientpb.OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.OutputFormat_EXECUTABLE
	case "shared":
		configFormat = clientpb.OutputFormat_SHARED_LIB
		isSharedLib = true
		runAtLoad, _ = cmd.Flags().GetBool("run-at-load")
	case "shellcode":
		configFormat = clientpb.OutputFormat_SHELLCODE
		isShellcode = true
		sgnEnabled, _ = cmd.Flags().GetBool("disable-sgn")
		sgnEnabled = !sgnEnabled
	case "service":
		configFormat = clientpb.OutputFormat_SERVICE
		isService = true
	default:
		// Default to exe
		configFormat = clientpb.OutputFormat_EXECUTABLE
	}

	targetOSF, _ := cmd.Flags().GetString("os")
	targetOS := strings.ToLower(targetOSF)
	targetArchF, _ := cmd.Flags().GetString("arch")
	targetArch := strings.ToLower(targetArchF)
	targetOS, targetArch = getTargets(targetOS, targetArch, con)
	if targetOS == "" || targetArch == "" {
		return "", nil
	}
	if configFormat == clientpb.OutputFormat_SHELLCODE && targetOS != "windows" {
		con.PrintErrorf("Shellcode format is currently only supported on Windows\n")
		return "", nil
	}
	if len(namedPipeC2) > 0 && targetOS != "windows" {
		con.PrintErrorf("Named pipe pivoting can only be used in Windows.")
		return "", nil
	}

	// Check to see if we can *probably* build the target binary
	if !checkBuildTargetCompatibility(configFormat, targetOS, targetArch, con) {
		return "", nil
	}

	var tunIP net.IP
	if wg, _ := cmd.Flags().GetString("wg"); wg != "" {
		uniqueWGIP, err := con.Rpc.GenerateUniqueIP(context.Background(), &commonpb.Empty{})
		tunIP = net.ParseIP(uniqueWGIP.IP)
		if err != nil {
			con.PrintErrorf("Failed to generate unique ip for wg peer tun interface")
			return "", nil
		}
		con.PrintInfof("Generated unique ip for wg peer tun interface: %s\n", tunIP.String())
	}

	netGo, _ := cmd.Flags().GetBool("netgo")

	// TODO: Use generics or something to check in a slice
	connectionStrategy, _ := cmd.Flags().GetString("strategy")
	if connectionStrategy != "" && connectionStrategy != "s" && connectionStrategy != "r" && connectionStrategy != "rd" {
		con.PrintErrorf("Invalid connection strategy: %s\n", connectionStrategy)
		return "", nil
	}

	// Parse Traffic Encoder Args
	httpC2Enabled := 0 < len(httpC2)
	trafficEncodersEnabled, trafficEncoderAssets := parseTrafficEncoderArgs(cmd, httpC2Enabled, con)

	c2Profile, _ := cmd.Flags().GetString("c2profile")
	if c2Profile == "" {
		c2Profile = consts.DefaultC2Profile
	}

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           targetArch,
		Debug:            debug,
		Evasion:          evasion,
		SGNEnabled:       sgnEnabled,
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,
		TemplateName:     templateName,

		WGPeerTunIP:       tunIP.String(),
		WGKeyExchangePort: wgKeyExchangePort,
		WGTcpCommsPort:    wgTcpCommsPort,

		ConnectionStrategy:  connectionStrategy,
		ReconnectInterval:   reconnectInterval * int64(time.Second),
		PollTimeout:         pollTimeout * int64(time.Second),
		MaxConnectionErrors: maxConnectionErrors,

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,
		LimitFileExists:   limitFileExists,
		LimitLocale:       limitLocale,

		Format:      configFormat,
		IsSharedLib: isSharedLib,
		IsService:   isService,
		IsShellcode: isShellcode,

		RunAtLoad:              runAtLoad,
		NetGoEnabled:           netGo,
		TrafficEncodersEnabled: trafficEncodersEnabled,
		Assets:                 trafficEncoderAssets,

		DebugFile:        debugFile,
		HTTPC2ConfigName: c2Profile,
	}

	return name, config
}

// parseTrafficEncoderArgs - parses the traffic encoder args and returns a bool indicating if traffic encoders are enabled.
func parseTrafficEncoderArgs(cmd *cobra.Command, httpC2Enabled bool, con *console.SliverClient) (bool, []*commonpb.File) {
	trafficEncoders, _ := cmd.Flags().GetString("traffic-encoders")
	encoders := []*commonpb.File{}
	if trafficEncoders != "" {
		if !httpC2Enabled {
			con.PrintWarnf("Traffic encoders are only supported with HTTP C2, flag will be ignored\n")
			return false, encoders
		}
		enabledEncoders := strings.Split(trafficEncoders, ",")
		for _, encoder := range enabledEncoders {
			if !strings.HasSuffix(encoder, ".wasm") {
				encoder += ".wasm"
			}
			encoders = append(encoders, &commonpb.File{Name: encoder})
		}
		return true, encoders
	}
	return false, encoders
}

func getTargets(targetOS string, targetArch string, con *console.SliverClient) (string, string) {
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

// ParseMTLSc2 - Parse mtls connection string arg.
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
		if uri.Scheme != "mtls" {
			return nil, fmt.Errorf("invalid mtls schema: %s", uri.Scheme)
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

// ParseWGc2 - Parse wg connect string arg.
func ParseWGc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "wg://") {
			uri, err = url.Parse(arg)
			if err != nil {
				return nil, err
			}
		} else {
			uri, err = url.Parse(fmt.Sprintf("wg://%s", arg))
			if err != nil {
				return nil, err
			}
		}
		if uri.Scheme != "wg" {
			return nil, fmt.Errorf("invalid wg schema: %s", uri.Scheme)
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

func hasValidC2AdvancedOptions(options url.Values) (bool, error) {
	for key, value := range options {
		if len(value) > 1 {
			return false, fmt.Errorf("too many values specified for advanced option %s. Only one value for %s can be specified", key, key)
		}
		testValue := value[0]

		switch key {
		/*
			The following options are passed through as-is:
			proxy-username
			proxy-password
			host-header
			force-resolv-conf (validation is handled server side)
			resolvers
		*/
		case "net-timeout", "tls-timeout", "poll-timeout", "timeout", "retry-wait":
			if _, err := time.ParseDuration(testValue); err != nil {
				return false, fmt.Errorf("error parsing C2 option \"%s\": %s", key, err.Error())
			}
		case "max-errors", "retry-count", "workers-per-resolver":
			if _, err := strconv.Atoi(testValue); err != nil {
				return false, fmt.Errorf("error parsing C2 option \"%s\": %s", key, err.Error())
			}
		case "driver":
			// If this is specified, then it should be wininet (the only alternative driver currently supported)
			if testValue != "wininet" {
				return false, fmt.Errorf("C2 option \"driver\" must be empty for the default driver or \"wininet\" for the wininet driver (Windows only)")
			}
		case "force-http", "disable-accept-header", "disable-upgrade-header", "ask-proxy-creds", "force-base32":
			if testValue != "true" && testValue != "false" {
				return false, fmt.Errorf("C2 option \"%s\" must be a boolean value: true or false", key)
			}
		case "proxy":
			proxyUri, err := url.Parse(testValue)
			if err != nil {
				return false, fmt.Errorf("invalid C2 option \"proxy\" specified: %s", err.Error())
			}
			if proxyUri.Scheme != "http" && proxyUri.Scheme != "https" && proxyUri.Scheme != "socks5" {
				if proxyUri.Scheme == "" {
					return false, fmt.Errorf("a proxy scheme must be specified: http, https, or socks5")
				} else {
					return false, fmt.Errorf("%s is not a valid proxy scheme. Accepted values are http, https, and socks5", proxyUri.Scheme)
				}
			}
			if proxyUri.Port() != "" {
				port, err := strconv.Atoi(proxyUri.Port())
				if err != nil {
					return false, fmt.Errorf("invalid proxy port \"%s\" specified: proxy port must be a number between 1 and 65535", proxyUri.Port())
				}
				if port <= 0 || port > 65535 {
					return false, fmt.Errorf("invalid proxy port \"%s\" specified: proxy port must be a number between 1 and 65535", proxyUri.Port())
				}
			}
		}
	}

	return true, nil
}

func checkOptionValue(c2Options url.Values, option string, value string) bool {
	if !c2Options.Has(option) {
		return false
	} else {
		optionValue := c2Options.Get(option)
		return strings.ToLower(optionValue) == value
	}
}

func uriWithoutProxyOptions(uri *url.URL) {
	options := uri.Query()
	// If any of the options do not exist, there is no error
	options.Del("proxy")
	options.Del("proxy-username")
	options.Del("proxy-password")
	options.Del("ask-proxy-creds")
	options.Del("fallback")

	uri.RawQuery = options.Encode()
}

// ParseHTTPc2 - Parse HTTP connection string arg.
func ParseHTTPc2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	allArguments := strings.Split(args, ",")
	for index, arg := range allArguments {
		var uri *url.URL
		var err error
		if cmp := strings.ToLower(arg); strings.HasPrefix(cmp, "http://") || strings.HasPrefix(cmp, "https://") {
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
		uri.Path = strings.TrimSuffix(uri.Path, "/")
		if uri.Scheme != "http" && uri.Scheme != "https" {
			return nil, fmt.Errorf("invalid http(s) scheme: %s", uri.Scheme)
		}
		if ok, err := hasValidC2AdvancedOptions(uri.Query()); !ok {
			return nil, err
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
		/* If a proxy is defined and the operator wants to fallback to connecting directly, add
		   a C2 that connects directly without the proxy settings.
		*/
		if checkOptionValue(uri.Query(), "fallback", "true") && uri.Query().Has("proxy") && !checkOptionValue(uri.Query(), "driver", "wininet") {
			uriWithoutProxyOptions(uri)
			c2s = append(c2s, &clientpb.ImplantC2{
				Priority: uint32(index + len(allArguments)),
				URL:      uri.String(),
			})
		}
	}
	return c2s, nil
}

// ParseDNSc2 - Parse DNS connection string arg.
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
		if uri.Scheme != "dns" {
			return nil, fmt.Errorf("invalid dns scheme: %s", uri.Scheme)
		}
		if ok, err := hasValidC2AdvancedOptions(uri.Query()); !ok {
			return nil, err
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseNamedPipec2 - Parse named pipe connection string arg.
func ParseNamedPipec2(args string) ([]*clientpb.ImplantC2, error) {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s, nil
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		arg = strings.ReplaceAll(arg, "\\", "/")
		arg = strings.TrimPrefix(arg, "/")
		arg = strings.TrimPrefix(arg, "/")
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
		if uri.Scheme != "namedpipe" {
			return nil, fmt.Errorf("invalid namedpipe scheme: %s", uri.Scheme)
		}

		if !strings.HasPrefix(uri.Path, "/pipe/") {
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Named pipe '%s' is missing the 'pipe' path prefix\nContinue anyways?", uri),
			}
			var confirm bool
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return nil, fmt.Errorf("invalid namedpipe path: %s", uri.Path)
			}
		}

		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s, nil
}

// ParseTCPPivotc2 - Parse tcp pivot connection string arg.
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
		if uri.Scheme != "tcppivot" {
			return nil, fmt.Errorf("invalid tcppivot scheme: %s", uri.Scheme)
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

func externalBuild(name string, config *clientpb.ImplantConfig, save string, con *console.SliverClient) (*commonpb.File, error) {
	potentialBuilders, err := findExternalBuilders(config, con)
	if err != nil {
		return nil, err
	}
	var externalBuilder *clientpb.Builder
	if len(potentialBuilders) == 1 {
		externalBuilder = potentialBuilders[0]
	} else {
		con.PrintInfof("Found %d external builders that can compile this configuration", len(potentialBuilders))
		externalBuilder, err = selectExternalBuilder(potentialBuilders, con)
		if err != nil {
			return nil, err
		}
	}
	con.PrintInfof("Using external builder: %s\n", externalBuilder.Name)

	if config.IsBeacon {
		interval := time.Duration(config.BeaconInterval)
		con.PrintInfof("Externally generating new %s/%s beacon implant binary (%v)\n", config.GOOS, config.GOARCH, interval)
	} else {
		con.PrintInfof("Externally generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)
	}
	if config.ObfuscateSymbols {
		con.PrintInfof("%sSymbol obfuscation is enabled%s\n", console.Bold, console.Normal)
	} else if !config.Debug {
		con.PrintErrorf("Symbol obfuscation is %sdisabled%s\n", console.Bold, console.Normal)
	}
	start := time.Now()

	listenerID, listener := con.CreateEventListener()

	waiting := true
	spinner := spin.New()

	sigint := make(chan os.Signal, 1) // Catch keyboard interrupts
	signal.Notify(sigint, os.Interrupt)

	con.PrintInfof("Creating external build ... ")
	externalImplantConfig, err := con.Rpc.GenerateExternal(context.Background(), &clientpb.ExternalGenerateReq{
		Name:        name,
		Config:      config,
		BuilderName: externalBuilder.Name,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil, err
	}
	con.Printf("done\n")

	msgF := "Waiting for external builder to acknowledge build (template: %s) ... %s"
	for waiting {
		select {

		case <-time.After(100 * time.Millisecond):
			elapsed := time.Since(start)
			msg := fmt.Sprintf(msgF, externalImplantConfig.Config.TemplateName, elapsed.Round(time.Second))
			fmt.Fprintf(con.App.ActiveMenu().OutOrStdout(), console.Clearln+" %s  %s", spinner.Next(), msg)

		case event := <-listener:
			switch event.EventType {

			case consts.ExternalBuildFailedEvent:
				parts := strings.SplitN(string(event.Data), ":", 2)
				if len(parts) != 2 {
					continue
				}
				if parts[0] == externalImplantConfig.Build.ID {
					con.RemoveEventListener(listenerID)
					return nil, fmt.Errorf("external build failed: %s", parts[1])
				}

			case consts.AcknowledgeBuildEvent:
				if string(event.Data) == externalImplantConfig.Build.ID {
					msgF = "External build acknowledged by builder (template: %s) ... %s"
				}

			case consts.ExternalBuildCompletedEvent:
				parts := strings.SplitN(string(event.Data), ":", 2)
				if len(parts) != 2 {
					continue
				}
				if parts[0] == externalImplantConfig.Build.ID {
					con.RemoveEventListener(listenerID)
					name = parts[1]
					waiting = false
				}

			}

		case <-sigint:
			waiting = false
			con.Printf("\n")
			return nil, fmt.Errorf("user interrupt")
		}
	}

	elapsed := time.Since(start)
	con.PrintInfof("Build completed in %s\n", elapsed.Round(time.Second))

	generated, err := con.Rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
		ImplantName: name,
	})
	if err != nil {
		return nil, err
	}
	con.PrintInfof("Build name: %s (%d bytes)\n", name, len(generated.File.Data))

	saveTo, err := saveLocation(save, filepath.Base(generated.File.Name), con)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(saveTo, generated.File.Data, 0o700)
	if err != nil {
		con.PrintErrorf("Failed to write to: %s\n", saveTo)
		return nil, err
	}
	con.PrintInfof("Implant saved to %s\n", saveTo)

	return nil, nil
}

func compile(name string, config *clientpb.ImplantConfig, save string, con *console.SliverClient) (*commonpb.File, error) {
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
		Name:   name,
		Config: config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil, err
	}

	elapsed := time.Since(start)
	con.PrintInfof("Build completed in %s\n", elapsed.Round(time.Second))
	if len(generated.File.Data) == 0 {
		con.PrintErrorf("Build failed, no file data\n")
		return nil, errors.New("no file data")
	}

	fileData := generated.File.Data
	if config.IsShellcode {
		if !config.SGNEnabled {
			con.PrintErrorf("Shikata ga nai encoder is %sdisabled%s\n", console.Bold, console.Normal)
		} else {
			con.PrintInfof("Encoding shellcode with shikata ga nai ... ")
			resp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
				Encoder:      clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
				Architecture: config.GOARCH,
				Iterations:   1,
				BadChars:     []byte{},
				Data:         fileData,
			})
			if err != nil {
				con.PrintErrorf("%s\n", err)
			} else {
				con.Printf("success!\n")
				fileData = resp.GetData()
			}
		}
	}

	saveTo, err := saveLocation(save, generated.File.Name, con)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(saveTo, fileData, 0o700)
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
	if config.LimitLocale != "" {
		limits = append(limits, fmt.Sprintf("locale=%s", config.LimitLocale))
	}
	return strings.Join(limits, "; ")
}

func checkBuildTargetCompatibility(format clientpb.OutputFormat, targetOS string, targetArch string, con *console.SliverClient) bool {
	if format == clientpb.OutputFormat_EXECUTABLE {
		return true // We don't need cross-compilers when targeting EXECUTABLE formats
	}

	compilers, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to check target compatibility: %s\n", err)
		return true
	}

	if runtime.GOOS != "darwin" && targetOS == "darwin" {
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

func warnMissingCrossCompiler(format clientpb.OutputFormat, targetOS string, targetArch string, con *console.SliverClient) bool {
	con.PrintWarnf("Missing cross-compiler for %s on %s/%s\n", nameOfOutputFormat(format), targetOS, targetArch)
	switch targetOS {
	case "darwin":
		con.PrintWarnf("The server cannot find an installation of osxcross")
	}
	con.PrintWarnf("For more information please read %s\n", crossCompilerInfoURL)

	confirm := false
	prompt := &survey.Confirm{Message: "Try to compile anyways (will likely fail)?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

func findExternalBuilders(config *clientpb.ImplantConfig, con *console.SliverClient) ([]*clientpb.Builder, error) {
	builders, err := con.Rpc.Builders(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if len(builders.Builders) < 1 {
		return []*clientpb.Builder{}, ErrNoExternalBuilder
	}

	validBuilders := []*clientpb.Builder{}
	for _, builder := range builders.Builders {
		for _, target := range builder.Targets {
			if target.GOOS == config.GOOS && target.GOARCH == config.GOARCH && config.Format == target.Format {
				validBuilders = append(validBuilders, builder)
				break
			}
		}
	}

	if len(validBuilders) < 1 {
		return []*clientpb.Builder{}, ErrNoValidBuilders
	}

	return validBuilders, nil
}

func selectExternalBuilder(builders []*clientpb.Builder, _ *console.SliverClient) (*clientpb.Builder, error) {
	choices := []string{}
	for _, builder := range builders {
		choices = append(choices, builder.Name)
	}
	choice := ""
	prompt := &survey.Select{
		Message: "Select an external builder:",
		Options: choices,
	}
	survey.AskOne(prompt, &choice, nil)
	for _, builder := range builders {
		if builder.Name == choice {
			return builder, nil
		}
	}
	return nil, ErrNoValidBuilders
}
