package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
)

// legacyBuildRequest translates the compatibility Generate request into the
// neutral planner contract. Sliver-only configuration stays in ImplantConfig;
// it is not interpreted by the generic planner.
func legacyBuildRequest(name string, config *clientpb.ImplantConfig) (buildplan.Request, error) {
	if config == nil {
		return buildplan.Request{}, fmt.Errorf("missing implant config")
	}
	format, err := legacyFinalFormat(config)
	if err != nil {
		return buildplan.Request{}, err
	}
	mode := "session"
	if config.IsBeacon {
		mode = "beacon"
	}
	request := buildplan.Request{
		Name:        name,
		Target:      buildplan.Target{OS: config.GOOS, Arch: config.GOARCH},
		Mode:        mode,
		FinalFormat: format,
		Core:        buildplan.Reference{Name: "builtin/sliver"},
		Provider:    buildplan.ProviderPolicy{Mode: "pinned", Builder: "server"},
	}
	for _, c2 := range config.C2 {
		if c2 == nil {
			return buildplan.Request{}, fmt.Errorf("nil C2 endpoint")
		}
		uri, err := url.Parse(c2.URL)
		if err != nil {
			return buildplan.Request{}, fmt.Errorf("invalid C2 endpoint: %w", err)
		}
		protocol := uri.Scheme
		switch protocol {
		case "wg":
			protocol = "wireguard"
		case "namedpipe":
			protocol = "named-pipe"
		case "tcppivot":
			protocol = "tcp-pivot"
		}
		request.C2 = append(request.C2, buildplan.Endpoint{Protocol: protocol, Address: c2.URL})
	}
	for _, item := range []struct {
		enabled  bool
		protocol string
	}{
		{config.IncludeMTLS, "mtls"}, {config.IncludeHTTP, "http"},
		{config.IncludeWG, "wireguard"}, {config.IncludeDNS, "dns"},
		{config.IncludeNamePipe, "named-pipe"}, {config.IncludeTCP, "tcp-pivot"},
	} {
		if item.enabled {
			request.IncludeProtocols = append(request.IncludeProtocols, item.protocol)
			if item.protocol == "http" {
				request.IncludeProtocols = append(request.IncludeProtocols, "https")
			}
		}
	}
	coreOptions := []struct {
		name  string
		value any
	}{
		{"debug", config.Debug}, {"evasion", config.Evasion},
		{"obfuscateSymbols", config.ObfuscateSymbols},
		{"serviceName", config.ServiceName}, {"runAtLoad", config.RunAtLoad},
		{"netgo", config.NetGoEnabled}, {"canaryDomains", append([]string{}, config.CanaryDomains...)},
		{"debugFile", config.DebugFile}, {"limitDomainJoined", config.LimitDomainJoined},
		{"limitDatetime", config.LimitDatetime}, {"limitHostname", config.LimitHostname},
		{"limitUsername", config.LimitUsername}, {"limitFileExists", config.LimitFileExists},
		{"limitLocale", config.LimitLocale},
	}
	if len(config.Exports) != 0 {
		coreOptions = append(coreOptions, struct {
			name  string
			value any
		}{"exports", config.Exports})
	}
	if config.TrafficEncodersEnabled {
		names := []string{}
		for _, asset := range config.Assets {
			if asset != nil {
				names = append(names, asset.Name)
			}
		}
		coreOptions = append(coreOptions, struct {
			name  string
			value any
		}{"trafficEncoders", names})
	}
	for _, option := range coreOptions {
		encoded, err := json.Marshal(option.value)
		if err != nil {
			return buildplan.Request{}, err
		}
		request.CoreOptions = append(request.CoreOptions, buildplan.Option{Name: option.name, ValueJSON: encoded})
	}
	if format == "shellcode" {
		converter := map[string]string{"windows": "builtin/donut", "darwin": "builtin/beignet", "linux": "builtin/malasada"}[config.GOOS]
		if converter == "" {
			return buildplan.Request{}, fmt.Errorf("no shellcode converter for %s", config.GOOS)
		}
		step := buildplan.TransformRequest{Spec: buildplan.Reference{Name: converter}}
		shellcodeConfig := config.ShellcodeConfig
		if shellcodeConfig == nil {
			shellcodeConfig = &clientpb.ShellcodeConfig{}
		}
		values := []struct {
			name  string
			value any
		}{
			{"compress", shellcodeConfig.Compress == 2},
		}
		if config.GOOS == "windows" {
			for _, number := range []struct {
				name  string
				value uint32
				max   uint32
			}{
				{"entropy", shellcodeConfig.Entropy, 3},
				{"exitOption", shellcodeConfig.ExitOpt, 3},
				{"bypass", shellcodeConfig.Bypass, 3},
				{"headers", shellcodeConfig.Headers, 2},
			} {
				if number.value >= 1 && number.value <= number.max {
					values = append(values, struct {
						name  string
						value any
					}{number.name, number.value})
				}
			}
			values = append(values,
				struct {
					name  string
					value any
				}{"thread", shellcodeConfig.Thread || config.ShellcodeConfig == nil},
				struct {
					name  string
					value any
				}{"unicode", shellcodeConfig.Unicode},
				struct {
					name  string
					value any
				}{"originalEntryPoint", shellcodeConfig.OEP},
			)
		}
		for _, option := range values {
			encoded, err := json.Marshal(option.value)
			if err != nil {
				return buildplan.Request{}, err
			}
			step.Options = append(step.Options, buildplan.Option{Name: option.name, ValueJSON: encoded})
		}
		request.Transforms = append(request.Transforms, step)
		encoder := config.ShellcodeEncoder
		legacySGN := encoder == clientpb.ShellcodeEncoder_NONE && config.SGNEnabled
		if legacySGN {
			encoder = clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
		}
		encoderSpec := map[clientpb.ShellcodeEncoder]string{
			clientpb.ShellcodeEncoder_SHIKATA_GA_NAI: "builtin/shikata-ga-nai",
			clientpb.ShellcodeEncoder_XOR:            "builtin/xor",
			clientpb.ShellcodeEncoder_XOR_DYNAMIC:    "builtin/xor-dynamic",
		}[encoder]
		if encoder != clientpb.ShellcodeEncoder_NONE && encoderSpec == "" {
			return buildplan.Request{}, fmt.Errorf("unknown shellcode encoder enum %d", int32(encoder))
		}
		if encoderSpec != "" && (!legacySGN || buildtransform.SupportsEncoder(encoderSpec, request.Target)) {
			request.Transforms = append(request.Transforms, buildplan.TransformRequest{Spec: buildplan.Reference{Name: encoderSpec}})
		}
	}
	return request, nil
}

func legacyFinalFormat(config *clientpb.ImplantConfig) (string, error) {
	switch config.Format {
	case clientpb.OutputFormat_EXECUTABLE:
		return "exe", nil
	case clientpb.OutputFormat_SERVICE:
		return "service", nil
	case clientpb.OutputFormat_SHARED_LIB:
		return "shared", nil
	case clientpb.OutputFormat_GO_ARCHIVE:
		return "archive", nil
	case clientpb.OutputFormat_SHELLCODE:
		return "shellcode", nil
	default:
		return "", fmt.Errorf("no core capability for output format %s", config.Format)
	}
}

func (rpc *Server) replaceRecordedFinal(buildID string, fileData []byte) error {
	store, err := rpc.buildRecordStore()
	if err != nil {
		return err
	}
	record, err := store.Get(buildID)
	if errors.Is(err, os.ErrNotExist) {
		return nil // A pre-M2 build has no build record to update.
	}
	if err != nil {
		return err
	}
	if record.Final == nil {
		return fmt.Errorf("build %s has no recorded final artifact", buildID)
	}
	hash := sha256.Sum256(fileData)
	final := *record.Final
	final.Size = int64(len(fileData))
	final.SHA256 = hex.EncodeToString(hash[:])
	_, err = store.ReplaceFinal(buildID, final, fileData)
	return err
}
