package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// previewGeneratePlan deliberately reads flags without calling parseCompileFlags.
// That path allocates a WireGuard peer IP and may prompt for a compiler or
// shellcode encoder. A preview must have no build-side effects.
func previewGeneratePlan(cmd *cobra.Command, con *console.SliverClient, beacon bool) {
	request, err := planRequestFromFlags(cmd, beacon)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	plan, err := con.Rpc.PlanBuild(ctx, request)
	if err != nil {
		con.PrintErrorf("Build plan failed: %s\n", err)
		return
	}
	con.PrintInfof("Plan %s: %s/%s %s -> %s\n", plan.GetPlanSHA256(), request.GetTarget().GetGOOS(), request.GetTarget().GetGOARCH(), request.GetMode(), request.GetFormat())
	for index, step := range plan.GetSteps() {
		ref := step.GetSpec()
		out := step.GetOutput()
		con.PrintInfof("  %d. %s %s@%s (%s) via %s [%s] -> %s/%s\n", index+1, step.GetKind(), ref.GetName(), ref.GetVersion(), ref.GetDigest(), step.GetProvider(), step.GetToolchain(), out.GetKind(), out.GetContainer())
	}
}

func planRequestFromFlags(cmd *cobra.Command, beacon bool) (*clientpb.BuildRequest, error) {
	if cmd == nil {
		return nil, errors.New("missing generate command")
	}
	name, _ := cmd.Flags().GetString("name")
	name = strings.ToLower(name)
	if name != "" {
		if err := util.AllowedName(name); err != nil {
			return nil, err
		}
	}
	goos, _ := cmd.Flags().GetString("os")
	goarch, _ := cmd.Flags().GetString("arch")
	goos, goarch = previewTarget(goos, goarch)
	if !SupportedCompilerTargets[goos+"/"+goarch] {
		return nil, fmt.Errorf("no core capability for target %s/%s", goos, goarch)
	}
	format, _ := cmd.Flags().GetString("format")
	selection, err := selectOutputFormat(format)
	if err != nil {
		return nil, err
	}
	mode := "session"
	var beaconSettings *clientpb.BuildBeaconSettings
	if beacon {
		mode = "beacon"
		config := &clientpb.ImplantConfig{}
		if err := parseBeaconFlags(cmd, config); err != nil {
			return nil, err
		}
		beaconSettings = &clientpb.BuildBeaconSettings{
			IntervalSeconds: uint64(config.BeaconInterval / int64(time.Second)),
			JitterSeconds:   uint64(config.BeaconJitter / int64(time.Second)),
		}
	}
	c2, err := previewEndpoints(cmd)
	if err != nil {
		return nil, err
	}
	if len(c2) == 0 {
		return nil, errors.New("must specify at least one C2 endpoint")
	}
	coreOptions, err := previewCoreOptions(cmd, format)
	if err != nil {
		return nil, err
	}
	transforms, err := previewTransforms(cmd, goos, format)
	if err != nil {
		return nil, err
	}
	metadata, err := parseSpoofMetadataFlag(cmd, &clientpb.ImplantConfig{GOOS: goos, GOARCH: goarch, Format: selection.format})
	if err != nil {
		return nil, err
	}
	if metadata != nil {
		if goos != "windows" || format != "exe" && format != "service" && format != "shared" {
			return nil, errors.New("PE metadata requires a Windows executable, service, or shared library")
		}
		source := metadata.GetPE().GetSource()
		if source == nil || len(source.GetData()) == 0 {
			return nil, errors.New("PE metadata requires donor bytes")
		}
		if err := validateSpoofMetadataSourceForTarget(&clientpb.ImplantConfig{
			GOOS: goos, GOARCH: goarch, Format: selection.format,
		}, source.GetName(), source.GetData()); err != nil {
			return nil, err
		}
		step, err := previewMetadataTransform(metadata)
		if err != nil {
			return nil, err
		}
		transforms = append(transforms, step)
	}
	providerMode := "automatic"
	if external, _ := cmd.Flags().GetBool("external-builder"); external {
		providerMode = "remote"
	}
	if remoteStep, _ := cmd.Flags().GetBool("remote-step"); remoteStep {
		providerMode = "remote"
	}
	builderName, _ := cmd.Flags().GetString("builder")
	if cmd.Flags().Changed("builder") && strings.TrimSpace(builderName) == "" {
		return nil, errors.New("builder pin cannot be empty")
	}
	if builderName != "" {
		providerMode = "pinned"
	}
	return &clientpb.BuildRequest{
		Name:             name,
		Target:           &clientpb.BuildTarget{GOOS: goos, GOARCH: goarch},
		Mode:             mode,
		C2:               c2,
		IncludeProtocols: previewIncludedProtocols(cmd),
		Format:           format,
		Core:             &clientpb.BuildSpecRef{Name: "builtin/sliver"},
		CoreOptions:      coreOptions,
		Transforms:       transforms,
		Provider:         &clientpb.BuildProviderPolicy{Mode: providerMode, Builder: builderName},
		BeaconSettings:   beaconSettings,
	}, nil
}

func previewMetadataTransform(metadata *clientpb.SpoofMetadataConfig) (*clientpb.BuildTransformRequest, error) {
	if metadata == nil {
		return nil, errors.New("missing PE metadata configuration")
	}
	serialized, err := proto.MarshalOptions{Deterministic: true}.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(serialized)
	refName := "metadata-sha256:" + hex.EncodeToString(hash[:])
	option, err := previewOption("donorArtifact", map[string]string{"kind": "input-artifact", "name": refName})
	if err != nil {
		return nil, err
	}
	return &clientpb.BuildTransformRequest{
		Spec: &clientpb.BuildSpecRef{Name: "builtin/pe-metadata"}, Options: []*clientpb.BuildOption{option},
	}, nil
}

func previewTarget(goos, goarch string) (string, string) {
	switch strings.ToLower(goos) {
	case "mac", "macos", "osx":
		goos = "darwin"
	case "win", "shit":
		goos = "windows"
	case "lin":
		goos = "linux"
	default:
		goos = strings.ToLower(goos)
	}
	switch strings.ToLower(goarch) {
	case "x64", "x86_64":
		goarch = "amd64"
	case "x86", "i386":
		goarch = "386"
	default:
		goarch = strings.ToLower(goarch)
	}
	return goos, goarch
}

func previewEndpoints(cmd *cobra.Command) ([]*clientpb.BuildEndpoint, error) {
	type c2Parser struct {
		flag  string
		parse func(string) ([]*clientpb.ImplantC2, error)
	}
	parsers := []c2Parser{
		{"mtls", ParseMTLSc2}, {"wg", ParseWGc2}, {"http", ParseHTTPc2},
		{"dns", ParseDNSc2}, {"tcp-pivot", ParseTCPPivotc2},
	}
	endpoints := []*clientpb.BuildEndpoint{}
	for _, parser := range parsers {
		raw, _ := cmd.Flags().GetString(parser.flag)
		parsed, err := parser.parse(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", parser.flag, err)
		}
		for _, c2 := range parsed {
			uri, err := url.Parse(c2.GetURL())
			if err != nil {
				return nil, err
			}
			protocol := uri.Scheme
			switch protocol {
			case "wg":
				protocol = "wireguard"
			case "tcppivot":
				protocol = "tcp-pivot"
			}
			endpoints = append(endpoints, &clientpb.BuildEndpoint{Protocol: protocol, Address: c2.GetURL()})
		}
	}
	pipe, _ := cmd.Flags().GetString("named-pipe")
	for _, raw := range strings.Split(pipe, ",") {
		if raw == "" {
			continue
		}
		// Avoid ParseNamedPipec2's interactive confirmation in a preview.
		value := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
		value = strings.TrimPrefix(strings.TrimPrefix(value, "/"), "/")
		if !strings.HasPrefix(value, "namedpipe://") {
			value = "namedpipe://" + value
		}
		uri, err := url.Parse(value)
		if err != nil || !strings.HasPrefix(uri.Path, "/pipe/") {
			return nil, fmt.Errorf("invalid named pipe endpoint %q", raw)
		}
		endpoints = append(endpoints, &clientpb.BuildEndpoint{Protocol: "named-pipe", Address: uri.String()})
	}
	return endpoints, nil
}

func previewIncludedProtocols(cmd *cobra.Command) []string {
	all, _ := cmd.Flags().GetBool("all-protocols")
	checks := []struct{ flag, protocol string }{
		{"include-mtls", "mtls"}, {"include-http", "http"},
		{"include-wg", "wireguard"}, {"include-dns", "dns"},
		{"include-named-pipe", "named-pipe"}, {"include-tcp-pivot", "tcp-pivot"},
	}
	protocols := []string{}
	for _, check := range checks {
		include, _ := cmd.Flags().GetBool(check.flag)
		if all || include {
			protocols = append(protocols, check.protocol)
			if check.protocol == "http" {
				protocols = append(protocols, "https")
			}
		}
	}
	return protocols
}

func previewOption(name string, value any) (*clientpb.BuildOption, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &clientpb.BuildOption{Name: name, ValueJSON: encoded}, nil
}

func previewCoreOptions(cmd *cobra.Command, format string) ([]*clientpb.BuildOption, error) {
	options := []*clientpb.BuildOption{}
	add := func(flag, name string, value any) error {
		if !cmd.Flags().Changed(flag) {
			return nil
		}
		option, err := previewOption(name, value)
		if err == nil {
			options = append(options, option)
		}
		return err
	}
	for _, entry := range []struct{ flag, name string }{
		{"debug", "debug"}, {"evasion", "evasion"}, {"run-at-load", "runAtLoad"},
		{"netgo", "netgo"}, {"limit-domainjoined", "limitDomainJoined"},
	} {
		value, _ := cmd.Flags().GetBool(entry.flag)
		if entry.flag == "run-at-load" && format != "shared" {
			continue
		}
		if err := add(entry.flag, entry.name, value); err != nil {
			return nil, err
		}
	}
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		option, err := previewOption("obfuscateSymbols", false)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	} else if skipped, _ := cmd.Flags().GetBool("skip-symbols"); cmd.Flags().Changed("skip-symbols") {
		if err := add("skip-symbols", "obfuscateSymbols", !skipped); err != nil {
			return nil, err
		}
	}
	for _, entry := range []struct{ flag, name string }{
		{"service-name", "serviceName"}, {"debug-file", "debugFile"},
		{"limit-datetime", "limitDatetime"}, {"limit-hostname", "limitHostname"},
		{"limit-username", "limitUsername"}, {"limit-fileexists", "limitFileExists"},
		{"limit-locale", "limitLocale"},
	} {
		if entry.flag == "service-name" && format != "service" {
			continue
		}
		value, _ := cmd.Flags().GetString(entry.flag)
		if err := add(entry.flag, entry.name, value); err != nil {
			return nil, err
		}
	}
	for _, entry := range []struct{ flag, name string }{
		{"exports", "exports"}, {"traffic-encoders", "trafficEncoders"},
		{"canary", "canaryDomains"},
	} {
		value, _ := cmd.Flags().GetString(entry.flag)
		if !cmd.Flags().Changed(entry.flag) {
			continue
		}
		parts := []string{}
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				if entry.flag == "canary" && !strings.HasSuffix(part, ".") {
					part += "."
				}
				parts = append(parts, part)
			}
		}
		if err := add(entry.flag, entry.name, parts); err != nil {
			return nil, err
		}
	}
	return options, nil
}

func previewTransforms(cmd *cobra.Command, goos, format string) ([]*clientpb.BuildTransformRequest, error) {
	if format != "shellcode" {
		if encoder, _ := cmd.Flags().GetString("shellcode-encoder"); encoder != "" {
			return nil, errors.New("shellcode encoder requires --format shellcode")
		}
		return nil, nil
	}
	transforms := []*clientpb.BuildTransformRequest{}
	converter := map[string]string{"windows": "builtin/donut", "darwin": "builtin/beignet", "linux": "builtin/malasada"}[goos]
	if converter == "" {
		return nil, fmt.Errorf("no shellcode converter for %s", goos)
	}
	step := &clientpb.BuildTransformRequest{Spec: &clientpb.BuildSpecRef{Name: converter}}
	anyOption := false
	for _, name := range []string{"shellcode-entropy", "shellcode-compress", "shellcode-exitopt", "shellcode-bypass", "shellcode-headers", "shellcode-thread", "shellcode-unicode", "shellcode-oep", "donut-entropy", "donut-compress", "donut-exitopt", "donut-bypass", "donut-headers", "donut-thread", "donut-unicode", "donut-oep"} {
		anyOption = anyOption || cmd.Flags().Changed(name)
	}
	if anyOption {
		values := map[string]any{"compress": previewShellcodeBool(cmd, "shellcode-compress", "donut-compress")}
		if goos == "windows" {
			for _, rule := range []struct {
				flag, legacy, option string
				minimum, maximum     uint32
			}{
				{"shellcode-entropy", "donut-entropy", "entropy", 1, 3},
				{"shellcode-exitopt", "donut-exitopt", "exitOption", 1, 3},
				{"shellcode-bypass", "donut-bypass", "bypass", 1, 3},
				{"shellcode-headers", "donut-headers", "headers", 1, 2},
				{"shellcode-oep", "donut-oep", "originalEntryPoint", 0, ^uint32(0)},
			} {
				value := previewShellcodeUint(cmd, rule.flag, rule.legacy)
				if value < rule.minimum || value > rule.maximum {
					return nil, fmt.Errorf("%s must be between %d and %d", rule.flag, rule.minimum, rule.maximum)
				}
				values[rule.option] = value
			}
			values["thread"] = previewShellcodeBool(cmd, "shellcode-thread", "donut-thread")
			values["unicode"] = previewShellcodeBool(cmd, "shellcode-unicode", "donut-unicode")
		}
		for _, name := range []string{"compress", "entropy", "exitOption", "bypass", "headers", "thread", "unicode", "originalEntryPoint"} {
			if value, ok := values[name]; ok {
				option, err := previewOption(name, value)
				if err != nil {
					return nil, err
				}
				step.Options = append(step.Options, option)
			}
		}
	}
	transforms = append(transforms, step)
	encoder, _ := cmd.Flags().GetString("shellcode-encoder")
	encoder = strings.ReplaceAll(normalizeShellcodeEncoderName(encoder), "_", "-")
	if encoder != "" && encoder != "none" {
		switch encoder {
		case "shikata-ga-nai", "xor", "xor-dynamic":
			transforms = append(transforms, &clientpb.BuildTransformRequest{Spec: &clientpb.BuildSpecRef{Name: "builtin/" + encoder}})
		default:
			return nil, fmt.Errorf("no installed shellcode encoder capability %q", encoder)
		}
	}
	return transforms, nil
}

func previewShellcodeUint(cmd *cobra.Command, flag, legacy string) uint32 {
	if cmd.Flags().Changed(flag) {
		value, _ := cmd.Flags().GetUint32(flag)
		return value
	}
	value, _ := cmd.Flags().GetUint32(legacy)
	return value
}

func previewShellcodeBool(cmd *cobra.Command, flag, legacy string) bool {
	if cmd.Flags().Changed(flag) {
		value, _ := cmd.Flags().GetBool(flag)
		return value
	}
	value, _ := cmd.Flags().GetBool(legacy)
	return value
}
