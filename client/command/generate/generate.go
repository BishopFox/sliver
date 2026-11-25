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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Binject/debug/pe"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		"linux/arm64":   true,
		"windows/386":   true,
		"windows/amd64": true,
		"windows/arm64": true,
	}
)

const (
	spoofMetadataFlagName      = "spoof-metadata"
	imageFileDLLCharacteristic = 0x2000
)

// GenerateCmd - The main command used to generate implant binaries
func GenerateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	retainWorkspace, err := retainWorkspaceFromFlags(cmd)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if preview, _ := cmd.Flags().GetBool("plan"); preview {
		previewGeneratePlan(cmd, con, false)
		return
	}
	if shouldRunGenerateForm(cmd, con, args) {
		compiler, _ := compilerTargets(con)
		result, err := forms.GenerateForm(compiler)
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Generate form failed: %s\n", err)
			return
		}
		if err := applyGenerateForm(cmd, result); err != nil {
			con.PrintErrorf("Failed to apply generate form values: %s\n", err)
			return
		}
	}
	name, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if !remoteStepRequested(cmd) {
		_, _ = compileWithWorkspaceRetention(name, config, spoofMetadata, save, retainWorkspace, con)
	} else {
		if err := remoteStepCompatibility(cmd, config); err != nil {
			con.PrintErrorf("Remote assigned-step build cannot preserve this configuration: %s\n", err)
			return
		}
		if _, err := remoteStepBuild(cmd, name, config, spoofMetadata, save, con); err != nil {
			con.PrintErrorf("%s\n", err)
		}
	}
}

func retainWorkspaceFromFlags(cmd *cobra.Command) (bool, error) {
	if cmd == nil {
		return false, errors.New("missing generate command")
	}
	raw, err := cmd.Flags().GetString("cleanup")
	if err != nil {
		return false, fmt.Errorf("read --cleanup: %w", err)
	}
	cleanup, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid --cleanup value %q: expected a boolean", raw)
	}
	if !cleanup {
		if remoteStepRequested(cmd) {
			return false, errors.New("--cleanup false is supported only for local generation; remote builders do not return a local server workspace")
		}
	}
	return !cleanup, nil
}

func shouldRunGenerateForm(cmd *cobra.Command, con *console.SliverClient, args []string) bool {
	if con == nil || con.IsCLI {
		return false
	}
	if len(args) != 0 {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func applyGenerateForm(cmd *cobra.Command, result *forms.GenerateFormResult) error {
	if err := cmd.Flags().Set("os", result.OS); err != nil {
		return err
	}
	if err := cmd.Flags().Set("arch", result.Arch); err != nil {
		return err
	}
	if err := cmd.Flags().Set("format", result.Format); err != nil {
		return err
	}
	name := strings.TrimSpace(result.Name)
	if name != "" {
		if err := cmd.Flags().Set("name", name); err != nil {
			return err
		}
	}
	save := strings.TrimSpace(result.Save)
	if save != "" {
		if err := cmd.Flags().Set("save", save); err != nil {
			return err
		}
	}

	c2Value := strings.TrimSpace(result.C2Value)
	switch result.C2Type {
	case "mtls":
		return cmd.Flags().Set("mtls", c2Value)
	case "wg":
		return cmd.Flags().Set("wg", c2Value)
	case "http":
		return cmd.Flags().Set("http", c2Value)
	case "dns":
		return cmd.Flags().Set("dns", c2Value)
	case "named-pipe":
		return cmd.Flags().Set("named-pipe", c2Value)
	case "tcp-pivot":
		return cmd.Flags().Set("tcp-pivot", c2Value)
	default:
		return errors.New("unsupported C2 transport selection")
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
	if !util.ValidPortableFileName(DefaultName) {
		return "", errors.New("unsafe artifact filename")
	}
	var saveTo string
	if save == "" {
		var err error
		save, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	save = expandPath(save)
	fi, err := os.Stat(save)
	if err == nil {
		if fi.IsDir() {
			saveTo = filepath.Join(save, DefaultName)
		} else {
			var confirm bool
			_ = forms.Confirm("Overwrite existing file?", &confirm)
			if !confirm {
				return "", errors.New("file already exists")
			}
			saveTo = save
		}
	} else if os.IsNotExist(err) {
		if strings.HasSuffix(save, "/") || strings.HasSuffix(save, string(os.PathSeparator)) {
			if err := os.MkdirAll(save, 0o700); err != nil {
				return "", err
			}
			saveTo = filepath.Join(save, DefaultName)
		} else {
			if err := os.MkdirAll(filepath.Dir(save), 0o700); err != nil {
				return "", err
			}
			saveTo = save
		}
	} else {
		return "", err
	}
	return filepath.Abs(saveTo)
}

func nameOfOutputFormat(value clientpb.OutputFormat) string {
	switch value {
	case clientpb.OutputFormat_EXECUTABLE:
		return "Executable"
	case clientpb.OutputFormat_SERVICE:
		return "Service"
	case clientpb.OutputFormat_SHARED_LIB:
		return "Shared Library"
	case clientpb.OutputFormat_GO_ARCHIVE:
		return "Go Archive"
	case clientpb.OutputFormat_SHELLCODE:
		return "Shellcode"
	default:
		return "Unknown"
	}
}

type outputFormatSelection struct {
	format    clientpb.OutputFormat
	shared    bool
	service   bool
	shellcode bool
}

func selectOutputFormat(value string) (outputFormatSelection, error) {
	switch value {
	case "exe":
		return outputFormatSelection{format: clientpb.OutputFormat_EXECUTABLE}, nil
	case "shared":
		return outputFormatSelection{format: clientpb.OutputFormat_SHARED_LIB, shared: true}, nil
	case "archive":
		return outputFormatSelection{format: clientpb.OutputFormat_GO_ARCHIVE, shared: true}, nil
	case "shellcode":
		return outputFormatSelection{format: clientpb.OutputFormat_SHELLCODE, shellcode: true}, nil
	case "service":
		return outputFormatSelection{format: clientpb.OutputFormat_SERVICE, service: true}, nil
	default:
		return outputFormatSelection{}, fmt.Errorf("no core capability for output format %q", value)
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

	var exports []string
	exportsArg, _ := cmd.Flags().GetString("exports")
	if exportsArg == "" {
		con.PrintErrorf("Shared libraries need at least one export\n")
		return "", nil
	} else {
		exports = strings.Split(exportsArg, ",")
	}

	format, _ := cmd.Flags().GetString("format")
	runAtLoad := false
	selection, err := selectOutputFormat(format)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return "", nil
	}
	configFormat := selection.format
	isSharedLib := selection.shared
	isService := selection.service
	isShellcode := selection.shellcode
	if format == "shared" {
		runAtLoad, _ = cmd.Flags().GetBool("run-at-load")
	}
	serviceName, _ := cmd.Flags().GetString("service-name")
	if !isService {
		serviceName = ""
	}

	targetOSF, _ := cmd.Flags().GetString("os")
	targetOS := strings.ToLower(targetOSF)
	targetArchF, _ := cmd.Flags().GetString("arch")
	targetArch := strings.ToLower(targetArchF)
	targetOS, targetArch = getTargets(targetOS, targetArch, con)
	if targetOS == "" || targetArch == "" {
		return "", nil
	}
	if configFormat == clientpb.OutputFormat_SHELLCODE {
		switch targetOS {
		case "windows":
			// Server-side implementation supports amd64 and 386 only.
			if targetArch != "amd64" && targetArch != "386" {
				con.PrintErrorf("Windows shellcode format is only supported on windows/amd64 and windows/386\n")
				return "", nil
			}
		case "darwin":
			if targetArch != "arm64" {
				con.PrintErrorf("macOS shellcode format is only supported on darwin/arm64\n")
				return "", nil
			}
		case "linux":
			if targetArch != "amd64" && targetArch != "arm64" {
				con.PrintErrorf("Linux shellcode format is only supported on linux/amd64 and linux/arm64\n")
				return "", nil
			}
		default:
			con.PrintErrorf("Shellcode format is currently only supported on Windows, macOS, and Linux\n")
			return "", nil
		}
	}
	if len(namedPipeC2) > 0 && targetOS != "windows" {
		con.PrintErrorf("Named pipe pivoting can only be used in Windows.")
		return "", nil
	}

	// Only the local server needs its own cross-compiler for this request.
	if !remoteStepRequested(cmd) && !checkBuildTargetCompatibility(configFormat, targetOS, targetArch, con) {
		return "", nil
	}

	shellcodeConfig, err := parseShellcodeFlags(cmd, targetOS, configFormat, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return "", nil
	}

	shellcodeEncoder, err := parseShellcodeEncoderFlag(cmd, targetArch, configFormat, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return "", nil
	}
	sgnEnabled := shellcodeEncoder == clientpb.ShellcodeEncoder_SHIKATA_GA_NAI // legacy field

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

	allProtocols, _ := cmd.Flags().GetBool("all-protocols")
	includeMTLS, _ := cmd.Flags().GetBool("include-mtls")
	includeHTTP, _ := cmd.Flags().GetBool("include-http")
	includeWG, _ := cmd.Flags().GetBool("include-wg")
	includeDNS, _ := cmd.Flags().GetBool("include-dns")
	includeNamedPipe, _ := cmd.Flags().GetBool("include-named-pipe")
	includeTCP, _ := cmd.Flags().GetBool("include-tcp-pivot")

	if allProtocols {
		includeMTLS = true
		includeHTTP = true
		includeWG = true
		includeDNS = true
		includeNamedPipe = true
		includeTCP = true
	}

	// exports if its a shared library

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           targetArch,
		Debug:            debug,
		Evasion:          evasion,
		SGNEnabled:       sgnEnabled,
		ShellcodeEncoder: shellcodeEncoder,
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

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
		Exports:     exports,
		ServiceName: serviceName,

		RunAtLoad:              runAtLoad,
		NetGoEnabled:           netGo,
		TrafficEncodersEnabled: trafficEncodersEnabled,
		Assets:                 trafficEncoderAssets,

		DebugFile:        debugFile,
		HTTPC2ConfigName: c2Profile,
		ShellcodeConfig:  shellcodeConfig,

		IncludeMTLS:     includeMTLS,
		IncludeHTTP:     includeHTTP,
		IncludeWG:       includeWG,
		IncludeDNS:      includeDNS,
		IncludeNamePipe: includeNamedPipe,
		IncludeTCP:      includeTCP,
	}

	return name, config
}

func parseSpoofMetadataFlag(cmd *cobra.Command, config *clientpb.ImplantConfig) (*clientpb.SpoofMetadataConfig, error) {
	if cmd == nil {
		return nil, nil
	}
	spoofMetadataFlag := cmd.Flags().Lookup(spoofMetadataFlagName)
	if spoofMetadataFlag == nil || !spoofMetadataFlag.Changed {
		return nil, nil
	}

	spoofMetadataValue := ""
	if spoofMetadataFlag.Value != nil {
		spoofMetadataValue = strings.TrimSpace(spoofMetadataFlag.Value.String())
	}

	switch strings.ToLower(spoofMetadataValue) {
	case "false":
		if len(cmd.Flags().Args()) > 0 {
			return nil, fmt.Errorf("--%s=false does not accept a donor path argument", spoofMetadataFlagName)
		}
		return nil, nil
	case "", "true":
		pathArg, err := spoofMetadataPathArg(cmd)
		if err != nil {
			return nil, err
		}
		if pathArg == "" {
			return parseSpoofMetadataFromConfigFile()
		}
		return parseSpoofMetadataFromPath(pathArg, config)
	default:
		if len(cmd.Flags().Args()) > 0 {
			return nil, fmt.Errorf("--%s accepts a donor path from either the flag value or positional argument, not both", spoofMetadataFlagName)
		}
		return parseSpoofMetadataFromPath(spoofMetadataValue, config)
	}
}

func spoofMetadataPathArg(cmd *cobra.Command) (string, error) {
	if cmd == nil {
		return "", nil
	}
	switch args := cmd.Flags().Args(); len(args) {
	case 0:
		return "", nil
	case 1:
		return strings.TrimSpace(args[0]), nil
	default:
		return "", fmt.Errorf("--%s accepts at most one donor path argument", spoofMetadataFlagName)
	}
}

func parseSpoofMetadataFromConfigFile() (*clientpb.SpoofMetadataConfig, error) {
	configPath := assets.SpoofMetadataConfigPath()
	config, err := assets.LoadSpoofMetadataConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load spoof metadata config %s: %w", configPath, err)
	}

	spoofMetadata, err := spoofMetadataConfigToProtobuf(config)
	if err != nil {
		return nil, fmt.Errorf("invalid spoof metadata config %s: %w", configPath, err)
	}
	return spoofMetadata, nil
}

func parseSpoofMetadataFromPath(pathValue string, config *clientpb.ImplantConfig) (*clientpb.SpoofMetadataConfig, error) {
	if config == nil {
		return nil, errors.New("missing implant configuration for spoof metadata validation")
	}
	pathValue = expandPath(pathValue)
	sourceData, err := os.ReadFile(pathValue)
	if err != nil {
		return nil, fmt.Errorf("failed to read spoof metadata source path %s: %w", pathValue, err)
	}
	if err := validateSpoofMetadataSourceForTarget(config, pathValue, sourceData); err != nil {
		return nil, err
	}
	return &clientpb.SpoofMetadataConfig{
		PE: &clientpb.PESpoofMetadataConfig{
			Source: &clientpb.SpoofMetadataFile{
				Name: filepath.Base(pathValue),
				Data: sourceData,
			},
		},
	}, nil
}

func validateSpoofMetadataSourceForTarget(config *clientpb.ImplantConfig, sourcePath string, sourceData []byte) error {
	if config == nil {
		return errors.New("missing implant configuration for spoof metadata validation")
	}
	if strings.ToLower(config.GetGOOS()) != "windows" {
		return fmt.Errorf("--%s path source currently supports only windows PE targets (got %s/%s)", spoofMetadataFlagName, config.GetGOOS(), config.GetGOARCH())
	}
	switch config.GetFormat() {
	case clientpb.OutputFormat_EXECUTABLE, clientpb.OutputFormat_SERVICE, clientpb.OutputFormat_SHARED_LIB:
	default:
		return fmt.Errorf("--%s path source requires an executable/service/shared target format (got %s)", spoofMetadataFlagName, nameOfOutputFormat(config.GetFormat()))
	}

	peFile, err := pe.NewFile(bytes.NewReader(sourceData))
	if err != nil {
		return fmt.Errorf("spoof metadata source %s is not a valid PE: %w", sourcePath, err)
	}
	defer peFile.Close()

	expectedMachine, err := expectedPEMachineForGoArch(config.GetGOARCH())
	if err != nil {
		return err
	}
	if peFile.FileHeader.Machine != expectedMachine {
		return fmt.Errorf(
			"spoof metadata source %s has machine %s, expected %s for target arch %s",
			sourcePath,
			peMachineName(peFile.FileHeader.Machine),
			peMachineName(expectedMachine),
			config.GetGOARCH(),
		)
	}

	isDLL := peFile.FileHeader.Characteristics&imageFileDLLCharacteristic != 0
	if config.GetFormat() == clientpb.OutputFormat_SHARED_LIB && !isDLL {
		return fmt.Errorf("spoof metadata source %s must be a DLL for shared target format", sourcePath)
	}
	if (config.GetFormat() == clientpb.OutputFormat_EXECUTABLE || config.GetFormat() == clientpb.OutputFormat_SERVICE) && isDLL {
		return fmt.Errorf("spoof metadata source %s appears to be a DLL but target format is %s", sourcePath, nameOfOutputFormat(config.GetFormat()))
	}

	return nil
}

func expectedPEMachineForGoArch(goarch string) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(goarch)) {
	case "amd64", "x64":
		return pe.IMAGE_FILE_MACHINE_AMD64, nil
	case "386", "x86", "i386":
		return pe.IMAGE_FILE_MACHINE_I386, nil
	case "arm64", "aarch64":
		return pe.IMAGE_FILE_MACHINE_ARM64, nil
	default:
		return 0, fmt.Errorf("unsupported windows architecture for spoof metadata source validation: %s", goarch)
	}
}

func peMachineName(machine uint16) string {
	switch machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "amd64"
	case pe.IMAGE_FILE_MACHINE_I386:
		return "386"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64"
	default:
		return fmt.Sprintf("0x%x", machine)
	}
}

func spoofMetadataConfigToProtobuf(config *assets.SpoofMetadataConfig) (*clientpb.SpoofMetadataConfig, error) {
	if config == nil {
		return nil, errors.New("missing spoof metadata configuration")
	}

	spoofMetadata := &clientpb.SpoofMetadataConfig{}
	if config.PE != nil {
		source, err := resolveSpoofMetadataDataSource("pe.source", config.PE.Source)
		if err != nil {
			return nil, err
		}
		icon, err := resolveSpoofMetadataDataSource("pe.icon", config.PE.Icon)
		if err != nil {
			return nil, err
		}
		spoofMetadata.PE = &clientpb.PESpoofMetadataConfig{
			Source:                   source,
			Icon:                     icon,
			ResourceDirectory:        imageResourceDirectoryToProtobuf(config.PE.ResourceDirectory),
			ResourceDirectoryEntries: imageResourceDirectoryEntriesToProtobuf(config.PE.ResourceDirectoryEntries),
			ResourceDataEntries:      imageResourceDataEntriesToProtobuf(config.PE.ResourceDataEntries),
			ExportDirectory:          imageExportDirectoryToProtobuf(config.PE.ExportDirectory),
		}
	}

	if spoofMetadata.GetPE() == nil {
		return nil, errors.New("no supported executable metadata configuration found")
	}
	if spoofMetadata.GetPE().GetSource() == nil || len(spoofMetadata.GetPE().GetSource().GetData()) == 0 {
		return nil, errors.New("pe.source requires either path or base64 content")
	}

	return spoofMetadata, nil
}

func resolveSpoofMetadataDataSource(name string, source *assets.SpoofMetadataDataSource) (*clientpb.SpoofMetadataFile, error) {
	if source == nil {
		return nil, nil
	}

	pathValue := strings.TrimSpace(source.Path)
	base64Value := strings.TrimSpace(source.Base64)

	if pathValue != "" && base64Value != "" {
		return nil, fmt.Errorf("%s cannot set both path and base64", name)
	}
	if pathValue == "" && base64Value == "" {
		return nil, nil
	}

	var (
		data []byte
		err  error
	)
	fileName := strings.TrimSpace(source.Name)
	if pathValue != "" {
		pathValue = expandPath(pathValue)
		data, err = os.ReadFile(pathValue)
		if err != nil {
			return nil, fmt.Errorf("%s failed to read path %s: %w", name, pathValue, err)
		}
		if fileName == "" {
			fileName = filepath.Base(pathValue)
		}
	} else {
		data, err = base64.StdEncoding.DecodeString(base64Value)
		if err != nil {
			data, err = base64.RawStdEncoding.DecodeString(base64Value)
			if err != nil {
				return nil, fmt.Errorf("%s has invalid base64 data: %w", name, err)
			}
		}
		if fileName == "" {
			fileName = "inline-data"
		}
	}

	return &clientpb.SpoofMetadataFile{
		Name: fileName,
		Data: data,
	}, nil
}

func imageResourceDirectoryToProtobuf(src *assets.ImageResourceDirectory) *clientpb.IMAGE_RESOURCE_DIRECTORY {
	if src == nil {
		return nil
	}
	return &clientpb.IMAGE_RESOURCE_DIRECTORY{
		Characteristics:      src.Characteristics,
		TimeDateStamp:        src.TimeDateStamp,
		MajorVersion:         src.MajorVersion,
		MinorVersion:         src.MinorVersion,
		NumberOfNamedEntries: src.NumberOfNamedEntries,
		NumberOfIdEntries:    src.NumberOfIDEntries,
	}
}

func imageResourceDirectoryEntriesToProtobuf(src []*assets.ImageResourceDirectoryEntry) []*clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY {
	if len(src) == 0 {
		return nil
	}
	out := make([]*clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY, 0, len(src))
	for _, entry := range src {
		if entry == nil {
			continue
		}
		out = append(out, &clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY{
			Name:         entry.Name,
			OffsetToData: entry.OffsetToData,
		})
	}
	return out
}

func imageResourceDataEntriesToProtobuf(src []*assets.ImageResourceDataEntry) []*clientpb.IMAGE_RESOURCE_DATA_ENTRY {
	if len(src) == 0 {
		return nil
	}
	out := make([]*clientpb.IMAGE_RESOURCE_DATA_ENTRY, 0, len(src))
	for _, entry := range src {
		if entry == nil {
			continue
		}
		out = append(out, &clientpb.IMAGE_RESOURCE_DATA_ENTRY{
			OffsetToData: entry.OffsetToData,
			Size:         entry.Size,
			CodePage:     entry.CodePage,
			Reserved:     entry.Reserved,
		})
	}
	return out
}

func imageExportDirectoryToProtobuf(src *assets.ImageExportDirectory) *clientpb.IMAGE_EXPORT_DIRECTORY {
	if src == nil {
		return nil
	}
	return &clientpb.IMAGE_EXPORT_DIRECTORY{
		Characteristics:       src.Characteristics,
		TimeDateStamp:         src.TimeDateStamp,
		MajorVersion:          src.MajorVersion,
		MinorVersion:          src.MinorVersion,
		Name:                  src.Name,
		Base:                  src.Base,
		NumberOfFunctions:     src.NumberOfFunctions,
		NumberOfNames:         src.NumberOfNames,
		AddressOfFunctions:    src.AddressOfFunctions,
		AddressOfNames:        src.AddressOfNames,
		AddressOfNameOrdinals: src.AddressOfNameOrdinals,
	}
}

func parseShellcodeFlags(cmd *cobra.Command, targetOS string, configFormat clientpb.OutputFormat, con *console.SliverClient) (*clientpb.ShellcodeConfig, error) {
	// Prefer the new `--shellcode-*` flags. Fall back to hidden deprecated `--donut-*` flags for compatibility.
	shellcodeEntropy, _ := cmd.Flags().GetUint32("shellcode-entropy")
	if !cmd.Flags().Changed("shellcode-entropy") {
		shellcodeEntropy, _ = cmd.Flags().GetUint32("donut-entropy")
	}

	shellcodeCompressEnabled, _ := cmd.Flags().GetBool("shellcode-compress")
	if !cmd.Flags().Changed("shellcode-compress") {
		shellcodeCompressEnabled, _ = cmd.Flags().GetBool("donut-compress")
	}

	shellcodeExitOpt, _ := cmd.Flags().GetUint32("shellcode-exitopt")
	if !cmd.Flags().Changed("shellcode-exitopt") {
		shellcodeExitOpt, _ = cmd.Flags().GetUint32("donut-exitopt")
	}

	shellcodeBypass, _ := cmd.Flags().GetUint32("shellcode-bypass")
	if !cmd.Flags().Changed("shellcode-bypass") {
		shellcodeBypass, _ = cmd.Flags().GetUint32("donut-bypass")
	}

	shellcodeHeaders, _ := cmd.Flags().GetUint32("shellcode-headers")
	if !cmd.Flags().Changed("shellcode-headers") {
		shellcodeHeaders, _ = cmd.Flags().GetUint32("donut-headers")
	}

	shellcodeThread, _ := cmd.Flags().GetBool("shellcode-thread")
	if !cmd.Flags().Changed("shellcode-thread") {
		shellcodeThread, _ = cmd.Flags().GetBool("donut-thread")
	}

	shellcodeUnicode, _ := cmd.Flags().GetBool("shellcode-unicode")
	if !cmd.Flags().Changed("shellcode-unicode") {
		shellcodeUnicode, _ = cmd.Flags().GetBool("donut-unicode")
	}

	shellcodeOEP, _ := cmd.Flags().GetUint32("shellcode-oep")
	if !cmd.Flags().Changed("shellcode-oep") {
		shellcodeOEP, _ = cmd.Flags().GetUint32("donut-oep")
	}

	anyChanged := cmd.Flags().Changed("shellcode-entropy") ||
		cmd.Flags().Changed("shellcode-compress") ||
		cmd.Flags().Changed("shellcode-exitopt") ||
		cmd.Flags().Changed("shellcode-bypass") ||
		cmd.Flags().Changed("shellcode-headers") ||
		cmd.Flags().Changed("shellcode-thread") ||
		cmd.Flags().Changed("shellcode-unicode") ||
		cmd.Flags().Changed("shellcode-oep") ||
		cmd.Flags().Changed("donut-entropy") ||
		cmd.Flags().Changed("donut-compress") ||
		cmd.Flags().Changed("donut-exitopt") ||
		cmd.Flags().Changed("donut-bypass") ||
		cmd.Flags().Changed("donut-headers") ||
		cmd.Flags().Changed("donut-thread") ||
		cmd.Flags().Changed("donut-unicode") ||
		cmd.Flags().Changed("donut-oep")

	if configFormat != clientpb.OutputFormat_SHELLCODE {
		if anyChanged {
			con.PrintWarnf("Shellcode options only apply when using `--format shellcode`, ignoring.\n")
		}
		return nil, nil
	}

	windowsOnlyChanged := cmd.Flags().Changed("shellcode-entropy") ||
		cmd.Flags().Changed("shellcode-exitopt") ||
		cmd.Flags().Changed("shellcode-bypass") ||
		cmd.Flags().Changed("shellcode-headers") ||
		cmd.Flags().Changed("shellcode-thread") ||
		cmd.Flags().Changed("shellcode-unicode") ||
		cmd.Flags().Changed("shellcode-oep") ||
		cmd.Flags().Changed("donut-entropy") ||
		cmd.Flags().Changed("donut-exitopt") ||
		cmd.Flags().Changed("donut-bypass") ||
		cmd.Flags().Changed("donut-headers") ||
		cmd.Flags().Changed("donut-thread") ||
		cmd.Flags().Changed("donut-unicode") ||
		cmd.Flags().Changed("donut-oep")

	// macOS and Linux shellcode currently only support compression.
	if targetOS != "windows" {
		if targetOS != "darwin" && targetOS != "linux" {
			// parseCompileFlags normally blocks this, but keep a safe fallback.
			if anyChanged {
				con.PrintWarnf("Shellcode options are only supported on Windows, macOS, and Linux shellcode, ignoring.\n")
			}
			return nil, nil
		}

		if windowsOnlyChanged {
			if targetOS == "darwin" {
				con.PrintWarnf("Windows-only shellcode options are ignored on macOS shellcode.\n")
			} else {
				con.PrintWarnf("Windows-only shellcode options are ignored on Linux shellcode.\n")
			}
		}

		shellcodeCompress := uint32(1)
		if shellcodeCompressEnabled {
			shellcodeCompress = 2
		}
		return &clientpb.ShellcodeConfig{
			Compress: shellcodeCompress,
		}, nil
	}

	if shellcodeEntropy < 1 || shellcodeEntropy > 3 {
		return nil, fmt.Errorf("shellcode-entropy must be between 1 and 3")
	}
	if shellcodeExitOpt < 1 || shellcodeExitOpt > 3 {
		return nil, fmt.Errorf("shellcode-exitopt must be between 1 and 3")
	}
	if shellcodeBypass < 1 || shellcodeBypass > 3 {
		return nil, fmt.Errorf("shellcode-bypass must be between 1 and 3")
	}
	if shellcodeHeaders < 1 || shellcodeHeaders > 2 {
		return nil, fmt.Errorf("shellcode-headers must be 1 or 2")
	}

	shellcodeCompress := uint32(1)
	if shellcodeCompressEnabled {
		shellcodeCompress = 2
	}

	return &clientpb.ShellcodeConfig{
		Entropy:  shellcodeEntropy,
		Compress: shellcodeCompress,
		ExitOpt:  shellcodeExitOpt,
		Bypass:   shellcodeBypass,
		Headers:  shellcodeHeaders,
		Thread:   shellcodeThread,
		Unicode:  shellcodeUnicode,
		OEP:      shellcodeOEP,
	}, nil
}

func normalizeShellcodeEncoderName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func normalizeShellcodeArch(arch string) string {
	normalized := strings.ToLower(strings.TrimSpace(arch))
	switch normalized {
	case "amd64", "x64", "x86_64":
		return "amd64"
	case "386", "x86", "i386":
		return "386"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return normalized
	}
}

func fetchShellcodeEncoderMap(con *console.SliverClient) (*clientpb.ShellcodeEncoderMap, error) {
	if con == nil || con.Rpc == nil {
		return nil, errors.New("no RPC client")
	}
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	return con.Rpc.ShellcodeEncoderMap(grpcCtx, &commonpb.Empty{})
}

func compatibleShellcodeEncoderNames(encoderMap *clientpb.ShellcodeEncoderMap, arch string) []string {
	if encoderMap == nil {
		return nil
	}
	arch = normalizeShellcodeArch(arch)
	archMap := encoderMap.GetEncoders()[arch]
	if archMap == nil {
		return nil
	}

	names := make([]string, 0, len(archMap.GetEncoders()))
	for name := range archMap.GetEncoders() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func shellcodeEncoderEnumForArch(encoderMap *clientpb.ShellcodeEncoderMap, arch, name string) (clientpb.ShellcodeEncoder, bool) {
	if encoderMap == nil {
		return clientpb.ShellcodeEncoder_NONE, false
	}
	arch = normalizeShellcodeArch(arch)
	name = normalizeShellcodeEncoderName(name)
	archMap := encoderMap.GetEncoders()[arch]
	if archMap == nil {
		return clientpb.ShellcodeEncoder_NONE, false
	}
	encoder, ok := archMap.GetEncoders()[name]
	return encoder, ok
}

func parseShellcodeEncoderFlag(cmd *cobra.Command, targetArch string, configFormat clientpb.OutputFormat, con *console.SliverClient) (clientpb.ShellcodeEncoder, error) {
	rawEncoder, _ := cmd.Flags().GetString("shellcode-encoder")
	rawEncoder = strings.TrimSpace(rawEncoder)

	if configFormat != clientpb.OutputFormat_SHELLCODE {
		if cmd.Flags().Changed("shellcode-encoder") && rawEncoder != "" {
			con.PrintWarnf("Shellcode encoder only applies when using `--format shellcode`, ignoring.\n")
		}
		return clientpb.ShellcodeEncoder_NONE, nil
	}

	if cmd.Flags().Changed("shellcode-encoder") && rawEncoder == "" {
		return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("shellcode-encoder cannot be empty; use 'none' to disable encoding")
	}

	normalized := normalizeShellcodeEncoderName(rawEncoder)
	if normalized == "none" {
		return clientpb.ShellcodeEncoder_NONE, nil
	}

	encoderMap, err := fetchShellcodeEncoderMap(con)
	if err != nil {
		return clientpb.ShellcodeEncoder_NONE, err
	}

	// If no encoder specified, prompt for a compatible encoder (or none).
	if rawEncoder == "" && !cmd.Flags().Changed("shellcode-encoder") {
		compatible := compatibleShellcodeEncoderNames(encoderMap, targetArch)
		if len(compatible) == 0 {
			// No known compatible encoders for this arch.
			return clientpb.ShellcodeEncoder_NONE, nil
		}

		options := append([]string{"none"}, compatible...)
		choice := "none"
		if err := forms.Select("Select a shellcode encoder (optional)", options, &choice); err != nil {
			return clientpb.ShellcodeEncoder_NONE, err
		}
		if normalizeShellcodeEncoderName(choice) == "none" {
			return clientpb.ShellcodeEncoder_NONE, nil
		}

		encoder, ok := shellcodeEncoderEnumForArch(encoderMap, targetArch, choice)
		if !ok {
			return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("unsupported shellcode encoder %q for arch %s", choice, normalizeShellcodeArch(targetArch))
		}
		return encoder, nil
	}

	// Encoder explicitly specified by name.
	encoder, ok := shellcodeEncoderEnumForArch(encoderMap, targetArch, rawEncoder)
	if !ok {
		compatible := compatibleShellcodeEncoderNames(encoderMap, targetArch)
		if len(compatible) == 0 {
			return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("no shellcode encoders are available for arch %s", normalizeShellcodeArch(targetArch))
		}
		return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("unsupported shellcode encoder %q for arch %s (valid: %s)", rawEncoder, normalizeShellcodeArch(targetArch), strings.Join(compatible, ", "))
	}
	return encoder, nil
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
		con.Printf("⚠️  Unsupported compiler target %s, but we can try to compile a generic implant.\n", console.StyleBold.Render(target))
		con.Printf("⚠️  Generic implants do not support all commands/features.\n")
		var confirm bool
		_ = forms.Confirm("Attempt to build generic implant?", &confirm)
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
		case "force-http", "disable-accept-header", "disable-upgrade-header", "ask-proxy-creds":
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
			var confirm bool
			_ = forms.Confirm(fmt.Sprintf("Named pipe '%s' is missing the 'pipe' path prefix\nContinue anyways?", uri), &confirm)
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

func applySpoofMetadataToBuild(implantBuildID string, implantName string, spoofMetadata *clientpb.SpoofMetadataConfig, con *console.SliverClient) error {
	if spoofMetadata == nil {
		return nil
	}

	req := &clientpb.GenerateSpoofMetadataReq{
		SpoofMetadata: spoofMetadata,
	}
	if buildID := strings.TrimSpace(implantBuildID); buildID != "" {
		req.ImplantBuildID = buildID
	} else if buildName := strings.TrimSpace(implantName); buildName != "" {
		req.ImplantName = buildName
	}

	if req.GetImplantBuildID() == "" && req.GetImplantName() == "" {
		return errors.New("unable to determine build selector for spoof metadata")
	}

	printSpoofMetadataSummary(spoofMetadata, con)
	con.PrintInfof("Applying spoof metadata to %s ... ", spoofMetadataBuildSelectorDescription(req))
	_, err := con.Rpc.GenerateSpoofMetadata(context.Background(), req)
	if err != nil {
		con.Printf("failed\n")
		return err
	}
	con.Printf("done\n")
	return nil
}

func printSpoofMetadataSummary(spoofMetadata *clientpb.SpoofMetadataConfig, con *console.SliverClient) {
	if spoofMetadata == nil || con == nil {
		return
	}
	peMetadata := spoofMetadata.GetPE()
	if peMetadata == nil {
		return
	}

	sourceName := "<unset>"
	sourceSize := 0
	if source := peMetadata.GetSource(); source != nil {
		sourceName = strings.TrimSpace(source.GetName())
		if sourceName == "" {
			sourceName = "inline-data"
		}
		sourceSize = len(source.GetData())
	}

	iconSummary := "none"
	if icon := peMetadata.GetIcon(); icon != nil && len(icon.GetData()) > 0 {
		iconName := strings.TrimSpace(icon.GetName())
		if iconName == "" {
			iconName = "inline-data"
		}
		iconSummary = fmt.Sprintf("%s (%d bytes)", iconName, len(icon.GetData()))
	}

	overrideLabels := []string{}
	if peMetadata.GetResourceDirectory() != nil {
		overrideLabels = append(overrideLabels, "resource-directory")
	}
	if len(peMetadata.GetResourceDirectoryEntries()) > 0 {
		overrideLabels = append(overrideLabels, fmt.Sprintf("resource-entries(%d)", len(peMetadata.GetResourceDirectoryEntries())))
	}
	if len(peMetadata.GetResourceDataEntries()) > 0 {
		overrideLabels = append(overrideLabels, fmt.Sprintf("resource-data-entries(%d)", len(peMetadata.GetResourceDataEntries())))
	}
	if peMetadata.GetExportDirectory() != nil {
		overrideLabels = append(overrideLabels, "export-directory")
	}
	overridesSummary := "none"
	if len(overrideLabels) > 0 {
		overridesSummary = strings.Join(overrideLabels, ", ")
	}

	con.PrintInfof("%s\n", console.StyleBold.Render("Spoof Metadata Details:"))
	con.PrintInfof("  Source: %s (%d bytes)\n", sourceName, sourceSize)
	con.PrintInfof("  Icon: %s\n", iconSummary)
	con.PrintInfof("  Overrides: %s\n", overridesSummary)
}

func spoofMetadataBuildSelectorDescription(req *clientpb.GenerateSpoofMetadataReq) string {
	if req == nil {
		return "build"
	}
	if buildID := strings.TrimSpace(req.GetImplantBuildID()); buildID != "" {
		return fmt.Sprintf("build id %s", buildID)
	}
	if buildName := strings.TrimSpace(req.GetImplantName()); buildName != "" {
		return fmt.Sprintf("build name %s", buildName)
	}
	return "build"
}

func requiresWindowsExecutableSuffix(config *clientpb.ImplantConfig) bool {
	if config == nil || !strings.EqualFold(config.GetGOOS(), "windows") {
		return false
	}
	switch config.GetFormat() {
	case clientpb.OutputFormat_EXECUTABLE, clientpb.OutputFormat_SERVICE:
		return true
	default:
		return false
	}
}

func ensureOutputPathSuffix(pathValue string, config *clientpb.ImplantConfig) string {
	if pathValue == "" || !requiresWindowsExecutableSuffix(config) {
		return pathValue
	}
	if strings.HasSuffix(strings.ToLower(pathValue), ".exe") {
		return pathValue
	}
	return pathValue + ".exe"
}

func outputSavePath(save string, defaultName string, config *clientpb.ImplantConfig, con *console.SliverClient) (string, error) {
	defaultName = ensureOutputPathSuffix(defaultName, config)
	saveTo, err := saveLocation(save, defaultName, con)
	if err != nil {
		return "", err
	}
	adjustedPath := ensureOutputPathSuffix(saveTo, config)
	if adjustedPath == saveTo {
		return saveTo, nil
	}

	if info, err := os.Stat(adjustedPath); err == nil && !info.IsDir() {
		var confirm bool
		_ = forms.Confirm("Overwrite existing file?", &confirm)
		if !confirm {
			return "", errors.New("file already exists")
		}
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if con != nil {
		con.PrintInfof("Adjusted output filename to %s for Windows executable format\n", adjustedPath)
	}
	return adjustedPath, nil
}

func compile(name string, config *clientpb.ImplantConfig, spoofMetadata *clientpb.SpoofMetadataConfig, save string, con *console.SliverClient) (*commonpb.File, error) {
	return compileWithWorkspaceRetention(name, config, spoofMetadata, save, false, con)
}

func compileWithWorkspaceRetention(name string, config *clientpb.ImplantConfig, spoofMetadata *clientpb.SpoofMetadataConfig,
	save string, retainWorkspace bool, con *console.SliverClient) (*commonpb.File, error) {
	if config.IsBeacon {
		interval := time.Duration(config.BeaconInterval)
		con.PrintInfof("Generating new %s/%s beacon implant binary (%v)\n", config.GOOS, config.GOARCH, interval)
	} else {
		con.PrintInfof("Generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)
	}
	if config.ObfuscateSymbols {
		con.PrintInfof("%s\n", console.StyleBold.Render("Symbol obfuscation is enabled"))
	} else if !config.Debug {
		con.PrintErrorf("Symbol obfuscation is %s\n", console.StyleBold.Render("disabled"))
	}

	start := time.Now()
	ctrl := make(chan bool)
	con.SpinUntil("Compiling, please wait ...", ctrl)

	generated, err := con.Rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Name:            name,
		Config:          config,
		SpoofMetadata:   spoofMetadata,
		RetainWorkspace: retainWorkspace,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil, err
	}

	elapsed := time.Since(start)
	con.PrintInfof("Build completed in %s\n", elapsed.Round(time.Second))
	if generated == nil || generated.File == nil || len(generated.File.Data) == 0 {
		con.PrintErrorf("Build failed, no file data\n")
		return nil, errors.New("no file data")
	}

	// New servers return the completed pipeline's final bytes. Older servers
	// ignore GenerateReq.SpoofMetadata and leave shellcode encoding to the
	// client, so inspect the build record before trusting those bytes.
	file, err := verifiedGenerateOutput(generated, name, config, spoofMetadata, con)
	if err != nil {
		return nil, err
	}
	if !util.ValidPortableFileName(file.Name) {
		return nil, errors.New("unsafe generated artifact filename")
	}

	saveTo, err := outputSavePath(save, file.Name, config, con)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(saveTo, file.Data, 0o700)
	if err != nil {
		con.PrintErrorf("Failed to write to: %s\n", saveTo)
		return nil, err
	}
	con.PrintInfof("Implant saved to %s\n", saveTo)
	if retainWorkspace {
		if generated.GetWorkspacePath() == "" {
			err := errors.New("--cleanup false was requested, but the server did not return a retained workspace path; check that the server supports workspace retention")
			con.PrintErrorf("%s\n", err)
			return file, err
		}
		con.PrintInfof("Build workspace retained on server at %q\n", generated.GetWorkspacePath())
	}
	return file, nil
}

//nolint:gocyclo // Compatibility dispatch and verification share the decision to trust generated bytes.
func verifiedGenerateOutput(generated *clientpb.Generate, requestedName string, config *clientpb.ImplantConfig,
	spoofMetadata *clientpb.SpoofMetadataConfig, con *console.SliverClient) (*commonpb.File, error) {
	if generated == nil || generated.File == nil || len(generated.File.Data) == 0 || config == nil || con == nil || con.Rpc == nil {
		return nil, errors.New("generation returned no artifact or connection")
	}
	encoder := config.ShellcodeEncoder
	if encoder == clientpb.ShellcodeEncoder_NONE && config.SGNEnabled {
		encoder = clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
	}
	if config.Format != clientpb.OutputFormat_SHELLCODE && !config.IsShellcode {
		encoder = clientpb.ShellcodeEncoder_NONE
	}
	if spoofMetadata == nil && encoder == clientpb.ShellcodeEncoder_NONE {
		return generated.File, nil
	}

	record, err := con.Rpc.GetBuildRecord(context.Background(), &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if status.Code(err) == codes.Unimplemented {
		return finishLegacyGenerate(generated, requestedName, config, spoofMetadata, encoder, con)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot verify server-side generation transforms: %w", err)
	}
	if err := verifyGenerateRecord(record, generated, spoofMetadata != nil, encoder); err != nil {
		return nil, err
	}
	return generated.File, nil
}

//nolint:gocyclo // Check record identity, artifact bytes, and requested transforms as one verification.
func verifyGenerateRecord(record *clientpb.BuildRecord, generated *clientpb.Generate, metadata bool,
	encoder clientpb.ShellcodeEncoder) error {
	if record == nil || record.ID == "" || record.ID != generated.ImplantBuildID ||
		record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.Plan == nil ||
		record.FinalArtifact == nil || record.FinalArtifact.Type == nil {
		return errors.New("server did not provide a completed build record for generation")
	}
	hash := sha256.Sum256(generated.File.Data)
	if record.FinalArtifact.FileName != generated.File.Name ||
		record.FinalArtifact.SizeBytes != uint64(len(generated.File.Data)) ||
		record.FinalArtifact.SHA256 != hex.EncodeToString(hash[:]) {
		return errors.New("generated bytes differ from the completed build record")
	}
	want := map[string]bool{}
	if metadata {
		want["builtin/pe-metadata"] = true
	}
	if encoder != clientpb.ShellcodeEncoder_NONE {
		name := map[clientpb.ShellcodeEncoder]string{
			clientpb.ShellcodeEncoder_SHIKATA_GA_NAI: "builtin/shikata-ga-nai",
			clientpb.ShellcodeEncoder_XOR:            "builtin/xor",
			clientpb.ShellcodeEncoder_XOR_DYNAMIC:    "builtin/xor-dynamic",
		}[encoder]
		if name == "" {
			return fmt.Errorf("unknown shellcode encoder enum %d", int32(encoder))
		}
		want[name] = true
	}
	for _, step := range record.Plan.Steps {
		delete(want, step.GetSpec().GetName())
	}
	if len(want) != 0 {
		return fmt.Errorf("server build record omits requested generation transform(s): %v", want)
	}
	return nil
}

//nolint:gocyclo // Keep the legacy transform order and its error handling together.
func finishLegacyGenerate(generated *clientpb.Generate, requestedName string, config *clientpb.ImplantConfig,
	spoofMetadata *clientpb.SpoofMetadataConfig, encoder clientpb.ShellcodeEncoder,
	con *console.SliverClient) (*commonpb.File, error) {
	file := generated.File
	if spoofMetadata != nil {
		if err := applySpoofMetadataToBuild(generated.ImplantBuildID, generated.ImplantName, spoofMetadata, con); err != nil {
			return nil, fmt.Errorf("legacy metadata spoofing failed: %w", err)
		}
		name := generated.ImplantName
		if name == "" {
			name = requestedName
		}
		if name == "" {
			return nil, errors.New("legacy metadata spoofing cannot regenerate an unnamed build")
		}
		regenerated, err := con.Rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{ImplantName: name})
		if err != nil {
			return nil, fmt.Errorf("retrieve legacy spoofed artifact: %w", err)
		}
		if regenerated == nil || regenerated.File == nil || len(regenerated.File.Data) == 0 {
			return nil, errors.New("legacy metadata spoofing returned no artifact")
		}
		file = regenerated.File
	}
	if encoder == clientpb.ShellcodeEncoder_NONE {
		return file, nil
	}
	encoderName := map[clientpb.ShellcodeEncoder]string{
		clientpb.ShellcodeEncoder_SHIKATA_GA_NAI: "shikata_ga_nai",
		clientpb.ShellcodeEncoder_XOR:            "xor",
		clientpb.ShellcodeEncoder_XOR_DYNAMIC:    "xor_dynamic",
	}[encoder]
	if encoderName == "" {
		return nil, fmt.Errorf("unknown shellcode encoder enum %d", int32(encoder))
	}
	encoderMap, err := fetchShellcodeEncoderMap(con)
	if err != nil {
		return nil, fmt.Errorf("fetch legacy shellcode encoders: %w", err)
	}
	if _, ok := shellcodeEncoderEnumForArch(encoderMap, config.GOARCH, encoderName); !ok {
		return nil, fmt.Errorf("shellcode encoder %q is not supported for arch %s", encoderName, normalizeShellcodeArch(config.GOARCH))
	}
	con.PrintInfof("Encoding shellcode with %s ... ", encoderName)
	response, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
		Encoder: encoder, Architecture: config.GOARCH, Iterations: 1, Data: file.Data,
	})
	if err != nil {
		return nil, fmt.Errorf("legacy shellcode encoding failed: %w", err)
	}
	if response == nil || len(response.Data) == 0 {
		return nil, errors.New("legacy shellcode encoder returned no artifact")
	}
	con.Printf("success!\n")
	return &commonpb.File{Name: file.Name, Data: response.Data}, nil
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
	_ = forms.Confirm("Try to compile anyways (will likely fail)?", &confirm)
	return confirm
}
