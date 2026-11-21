package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcompiler"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/magebuild"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/codenames"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/util"
)

const sliverExecutorID = "builtin-sliver-go-v1"

// SliverCoreAdapter owns the existing installed Sliver source, key preparation,
// rendering, and output-specific compilation. The neutral executor invokes it
// through buildcore.Adapter; legacy generation still uses the exported helper
// functions in this package.
type SliverCoreAdapter struct {
	// KeepWorkspace retains the exact local build directory, including private
	// material and rendered source, only after a successful final save.
	KeepWorkspace bool
}

func init() {
	if err := buildcore.RegisterDefault(SliverCoreAdapter{}); err != nil {
		panic(err)
	}
}

// ExecutorID returns the catalog identifier for the installed Sliver executor.
func (SliverCoreAdapter) ExecutorID() string { return sliverExecutorID }

// Supports reports the same compiler targets advertised by the legacy Sliver
// server. CGo requirements select the shared-library target because the
// executable target alone does not prove that a C compiler is available.
func (SliverCoreAdapter) Supports(target buildplan.Target, toolchain buildplan.Toolchain) bool {
	if toolchain.Driver != buildcompiler.GoDriverID {
		return false
	}
	format := clientpb.OutputFormat_EXECUTABLE
	if slices.Contains(toolchain.Requirements, "cgo") || slices.Contains(toolchain.Requirements, "native-or-zig-cc") {
		format = clientpb.OutputFormat_SHARED_LIB
	}
	for _, available := range GetCompilerTargets() {
		if available.GOOS == target.OS && available.GOARCH == target.Arch && available.Format == format {
			return true
		}
	}
	return false
}

// Build executes a resolved local plan. The core adapter interprets the
// Sliver-specific options; the generic registry and planner never see
// ImplantConfig or embedded Sliver source files.
func (adapter SliverCoreAdapter) Build(ctx context.Context, plan buildplan.Plan) (buildcore.Result, error) {
	config, err := adapter.ConfigFromPlan(plan)
	if err != nil {
		return buildcore.Result{}, err
	}
	return adapter.BuildLegacy(ctx, plan.Request.Name, config)
}

// BuildLegacy is the compatibility entry point for the existing Generate RPC.
// It keeps credential generation and compilation in the Sliver adapter while
// returning only a typed result to the caller.
//
//nolint:gocyclo // Legacy validation, output acceptance, and persistence share one workspace lifecycle.
func (adapter SliverCoreAdapter) BuildLegacy(ctx context.Context, name string, config *clientpb.ImplantConfig) (result buildcore.Result, buildErr error) {
	if config == nil {
		return buildcore.Result{}, fmt.Errorf("missing implant config")
	}
	if ctx == nil {
		return buildcore.Result{}, fmt.Errorf("missing build context")
	}
	if err := ctx.Err(); err != nil {
		return buildcore.Result{}, err
	}
	if name == "" {
		generated, err := codenames.GetCodename()
		if err != nil {
			return buildcore.Result{}, err
		}
		name = generated
	} else if err := util.AllowedName(name); err != nil {
		return buildcore.Result{}, err
	}
	if err := adapter.Normalize(config); err != nil {
		return buildcore.Result{}, err
	}

	build, err := adapter.Prepare(name, config)
	if err != nil {
		return buildcore.Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return buildcore.Result{}, err
	}
	httpConfig, err := db.LoadHTTPC2ConfigByName(config.HTTPC2ConfigName)
	if err != nil {
		return buildcore.Result{}, err
	}
	output, err := adapter.Compile(ctx, name, build, config, httpConfig.ImplantConfig)
	if err != nil {
		return buildcore.Result{}, err
	}
	workspacePath := filepath.Dir(filepath.Dir(output))
	saved := false
	defer func() {
		if adapter.KeepWorkspace && saved && buildErr == nil {
			result.WorkspacePath = workspacePath
			return
		}
		if err := workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), config.GOOS, config.GOARCH, name, output); err != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("remove private build workspace: %w", err))
			result = buildcore.Result{}
		}
	}()
	if err := ctx.Err(); err != nil {
		return buildcore.Result{}, err
	}
	if err := writeSliverResult(output); err != nil {
		return buildcore.Result{}, err
	}
	expected, err := sliverOutputArtifact(config)
	if err != nil {
		return buildcore.Result{}, err
	}
	data, artifact, err := artifactaccept.Accept(filepath.Dir(output), expected)
	if err != nil {
		return buildcore.Result{}, err
	}
	if err := ImplantBuildSave(build, config, output); err != nil {
		return buildcore.Result{}, err
	}
	saved = true
	return buildcore.Result{
		BuildID: build.ID, ImplantName: build.Name, FileName: artifact.Name, Data: data,
		ToolchainIdentity: sliverToolchainIdentity(),
	}, nil
}

func writeSliverResult(output string) error {
	binDir := filepath.Dir(output)
	file, err := os.CreateTemp(binDir, ".result-*.json")
	if err != nil {
		return fmt.Errorf("create Sliver output descriptor: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return err
	}
	name := filepath.Base(output)
	if err := json.NewEncoder(file).Encode(artifactaccept.Result{Primary: name, DisplayName: name}); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(binDir, "result.json"))
}

func sliverOutputArtifact(config *clientpb.ImplantConfig) (buildplan.Artifact, error) {
	artifact := buildplan.Artifact{Target: buildplan.Target{OS: config.GOOS, Arch: config.GOARCH}}
	switch config.Format {
	case clientpb.OutputFormat_EXECUTABLE:
		artifact.Kind, artifact.GoBuildMode = "executable", "default"
	case clientpb.OutputFormat_SERVICE:
		artifact.Kind, artifact.GoBuildMode = "service-executable", "default"
	case clientpb.OutputFormat_SHARED_LIB:
		artifact.Kind, artifact.GoBuildMode = "shared-library", "c-shared"
		artifact.EntrySymbols = slices.Clone(config.Exports)
	case clientpb.OutputFormat_GO_ARCHIVE:
		artifact.Kind, artifact.Container, artifact.GoBuildMode = "c-archive-bundle", "zip", "c-archive"
		artifact.EntrySymbols = slices.Clone(config.Exports)
	case clientpb.OutputFormat_SHELLCODE:
		artifact.Kind, artifact.Container = "shellcode", "raw"
	default:
		return buildplan.Artifact{}, fmt.Errorf("unsupported Sliver output format %s", config.Format)
	}
	if artifact.Container == "" {
		switch config.GOOS {
		case WINDOWS:
			artifact.Container = "pe"
		case LINUX:
			artifact.Container = "elf"
		case DARWIN:
			artifact.Container = "macho"
		default:
			return buildplan.Artifact{}, fmt.Errorf("unsupported Sliver output target %s", config.GOOS)
		}
	}
	return artifact, nil
}

// Normalize fills the Sliver defaults used by both legacy Generate planning
// and the adapter build. It does not create credentials or touch build storage.
func (SliverCoreAdapter) Normalize(config *clientpb.ImplantConfig) error {
	if config == nil {
		return fmt.Errorf("missing implant config")
	}
	includeSliverC2(config)
	if len(config.Exports) == 0 {
		config.Exports = []string{"StartW"}
	}
	config.IsService = config.IsService || config.Format == clientpb.OutputFormat_SERVICE
	config.IsSharedLib = config.IsSharedLib || config.Format == clientpb.OutputFormat_SHARED_LIB || config.Format == clientpb.OutputFormat_GO_ARCHIVE
	config.IsShellcode = config.IsShellcode || config.Format == clientpb.OutputFormat_SHELLCODE
	return nil
}

func sliverToolchainIdentity() string {
	mageIdentity := "Mage " + magebuild.Version
	if digest, err := magebuild.BuiltinSliverDigest(); err == nil {
		mageIdentity += " " + digest
	}
	versionFile := filepath.Join(gogo.GetGoRootDir(assets.GetRootAppDir()), "VERSION")
	if data, err := os.ReadFile(versionFile); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) != 0 {
			return mageIdentity + "; bundled " + fields[0]
		}
	}
	return mageIdentity + "; server runtime " + runtime.Version()
}

// Prepare creates Sliver's per-build keys and certificates. It is exported for
// compatibility callers that need the existing two-phase build flow.
func (adapter SliverCoreAdapter) Prepare(name string, config *clientpb.ImplantConfig) (*clientpb.ImplantBuild, error) {
	return adapter.generateConfig(name, config)
}

// Compile dispatches only to the installed Sliver output routines. The
// shellcode branch remains the legacy combined build/converter until M4.
func (adapter SliverCoreAdapter) Compile(ctx context.Context, name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig,
	httpConfig *clientpb.HTTPC2ImplantConfig) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch config.Format {
	case clientpb.OutputFormat_EXECUTABLE, clientpb.OutputFormat_SERVICE:
		return adapter.Executable(ctx, name, build, config, httpConfig)
	case clientpb.OutputFormat_SHARED_LIB:
		return adapter.SharedLibrary(ctx, name, build, config, httpConfig)
	case clientpb.OutputFormat_GO_ARCHIVE:
		return adapter.Archive(ctx, name, build, config, httpConfig)
	case clientpb.OutputFormat_SHELLCODE:
		return SliverShellcodeContext(ctx, name, build, config, httpConfig)
	default:
		return "", fmt.Errorf("no Sliver core capability for output format %s", config.Format)
	}
}

func includeSliverC2(config *clientpb.ImplantConfig) {
	config.IncludeMTLS = config.IncludeMTLS || models.IsC2Enabled([]string{"mtls"}, config.C2)
	config.IncludeWG = config.IncludeWG || models.IsC2Enabled([]string{"wg"}, config.C2)
	config.IncludeHTTP = config.IncludeHTTP || models.IsC2Enabled([]string{"http", "https"}, config.C2)
	config.IncludeDNS = config.IncludeDNS || models.IsC2Enabled([]string{"dns"}, config.C2)
	config.IncludeNamePipe = config.IncludeNamePipe || models.IsC2Enabled([]string{"namedpipe"}, config.C2)
	config.IncludeTCP = config.IncludeTCP || models.IsC2Enabled([]string{"tcppivot"}, config.C2)
}

// ConfigFromPlan translates the typed core request at the adapter boundary.
// This is intentionally Sliver-specific: adding another core never changes
// buildplan or the generic executor.
//
//nolint:gocyclo // Keep compatibility checks adjacent to the fields they translate.
func (SliverCoreAdapter) ConfigFromPlan(plan buildplan.Plan) (*clientpb.ImplantConfig, error) {
	unsupported := func(step, detail string) error {
		return &buildplan.PlanError{Code: buildplan.CodeUnsupportedExecution, Step: step, Detail: detail}
	}
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Spec.Name != "builtin/sliver" {
		return nil, unsupported("core", "the selected core or provider has no installed Sliver adapter")
	}
	converterIndex := -1
	for index := 1; index < len(plan.Steps); index++ {
		step := plan.Steps[index]
		if step.Kind != "transform" {
			return nil, unsupported(step.ID, "the selected step is not a transform")
		}
		switch step.Spec.Name {
		case "builtin/donut", "builtin/beignet", "builtin/malasada":
			if converterIndex != -1 || index != 1 {
				return nil, unsupported(step.ID, "shellcode converter must immediately follow the core")
			}
			converterIndex = index
		}
	}
	if converterIndex != -1 {
		converter := map[string]string{"windows": "builtin/donut", "darwin": "builtin/beignet", "linux": "builtin/malasada"}[plan.Request.Target.OS]
		if plan.Steps[converterIndex].Spec.Name != converter {
			return nil, unsupported(plan.Steps[converterIndex].ID, "shellcode converter does not match the target")
		}
	}
	var format clientpb.OutputFormat
	switch plan.Request.FinalFormat {
	case "exe":
		format = clientpb.OutputFormat_EXECUTABLE
	case "service":
		format = clientpb.OutputFormat_SERVICE
	case "shared":
		format = clientpb.OutputFormat_SHARED_LIB
	case "archive":
		format = clientpb.OutputFormat_GO_ARCHIVE
	case "shellcode":
		format = clientpb.OutputFormat_SHELLCODE
	case "stage":
		if converterIndex != -1 {
			format = clientpb.OutputFormat_SHELLCODE
		} else {
			switch plan.Steps[0].Output.Kind {
			case "executable":
				format = clientpb.OutputFormat_EXECUTABLE
			case "service-executable":
				format = clientpb.OutputFormat_SERVICE
			case "shared-library":
				format = clientpb.OutputFormat_SHARED_LIB
			case "c-archive-bundle":
				format = clientpb.OutputFormat_GO_ARCHIVE
			default:
				return nil, unsupported("core", "stage route has no supported Sliver core output")
			}
		}
	default:
		return nil, unsupported("final", "this output format has no installed Sliver adapter")
	}
	if (format == clientpb.OutputFormat_SHELLCODE) != (converterIndex != -1) {
		return nil, unsupported("transform", "Sliver shellcode requires one installed converter step")
	}
	config := &clientpb.ImplantConfig{
		GOOS: plan.Request.Target.OS, GOARCH: plan.Request.Target.Arch,
		IsBeacon: plan.Request.Mode == "beacon", BeaconInterval: int64(60 * time.Second),
		HTTPC2ConfigName:  "default",
		ReconnectInterval: int64(60 * time.Second), PollTimeout: int64(360 * time.Second), MaxConnectionErrors: 1000,
		Format: format, IsService: format == clientpb.OutputFormat_SERVICE,
		IsSharedLib: format == clientpb.OutputFormat_SHARED_LIB || format == clientpb.OutputFormat_GO_ARCHIVE,
		IsShellcode: format == clientpb.OutputFormat_SHELLCODE,
	}
	if settings := plan.Request.C2Settings; settings != nil {
		switch settings.ConnectionStrategy {
		case "sequential":
			config.ConnectionStrategy = ""
		case "random":
			config.ConnectionStrategy = "r"
		case "random-domain":
			config.ConnectionStrategy = "rd"
		default:
			return nil, unsupported("c2", "invalid connection strategy")
		}
		config.ReconnectInterval = int64(settings.ReconnectSeconds) * int64(time.Second)
		config.PollTimeout = int64(settings.PollTimeoutSeconds) * int64(time.Second)
		config.MaxConnectionErrors = settings.MaxConnectionErrors
		config.HTTPC2ConfigName = settings.HTTPC2Profile
		config.WGKeyExchangePort = settings.WireguardKeyExchangePort
		config.WGTcpCommsPort = settings.WireguardTCPCommsPort
	}
	if settings := plan.Request.BeaconSettings; settings != nil {
		config.BeaconInterval = int64(settings.IntervalSeconds) * int64(time.Second)
		config.BeaconJitter = int64(settings.JitterSeconds) * int64(time.Second)
	}
	for index, endpoint := range plan.Request.C2 {
		address, err := sliverEndpointURL(endpoint)
		if err != nil {
			return nil, &buildplan.PlanError{Code: buildplan.CodeInvalidRequest, Detail: err.Error()}
		}
		config.C2 = append(config.C2, &clientpb.ImplantC2{URL: address, Priority: uint32(index)})
	}
	for _, protocol := range plan.Request.IncludeProtocols {
		includeSliverProtocol(config, protocol)
	}
	for _, endpoint := range plan.Request.C2 {
		includeSliverProtocol(config, endpoint.Protocol)
	}
	for _, option := range plan.Steps[0].Options {
		if err := setSliverCoreOption(config, option); err != nil {
			return nil, unsupported("core", err.Error())
		}
	}
	if converterIndex != -1 {
		if err := setSliverConverterOptions(config, plan.Steps[converterIndex]); err != nil {
			return nil, unsupported(plan.Steps[converterIndex].ID, err.Error())
		}
	}
	return config, nil
}

func sliverEndpointURL(endpoint buildplan.Endpoint) (string, error) {
	parsed, err := url.Parse(endpoint.Address)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("C2 endpoint %q requires an absolute URL", endpoint.Address)
	}
	scheme := parsed.Scheme
	switch endpoint.Protocol {
	case "wireguard", "wg":
		if scheme != "wg" && scheme != "wireguard" {
			return "", fmt.Errorf("C2 endpoint protocol %q does not match URL scheme %q", endpoint.Protocol, scheme)
		}
		parsed.Scheme = "wg"
	case "named-pipe", "namedpipe":
		if scheme != "named-pipe" && scheme != "namedpipe" {
			return "", fmt.Errorf("C2 endpoint protocol %q does not match URL scheme %q", endpoint.Protocol, scheme)
		}
		parsed.Scheme = "namedpipe"
	case "tcp-pivot", "tcppivot":
		if scheme != "tcp-pivot" && scheme != "tcppivot" {
			return "", fmt.Errorf("C2 endpoint protocol %q does not match URL scheme %q", endpoint.Protocol, scheme)
		}
		parsed.Scheme = "tcppivot"
	default:
		if scheme != endpoint.Protocol {
			return "", fmt.Errorf("C2 endpoint protocol %q does not match URL scheme %q", endpoint.Protocol, scheme)
		}
	}
	return parsed.String(), nil
}

func includeSliverProtocol(config *clientpb.ImplantConfig, protocol string) {
	switch protocol {
	case "mtls":
		config.IncludeMTLS = true
	case "http", "https":
		config.IncludeHTTP = true
	case "wireguard", "wg":
		config.IncludeWG = true
	case "dns":
		config.IncludeDNS = true
	case "named-pipe", "namedpipe":
		config.IncludeNamePipe = true
	case "tcp-pivot", "tcppivot":
		config.IncludeTCP = true
	}
}

//nolint:gocyclo // A single switch records the complete option-to-field mapping and its decode errors.
func setSliverCoreOption(config *clientpb.ImplantConfig, option buildplan.Option) error {
	var err error
	switch option.Name {
	case "debug":
		err = json.Unmarshal(option.ValueJSON, &config.Debug)
	case "evasion":
		err = json.Unmarshal(option.ValueJSON, &config.Evasion)
	case "obfuscateSymbols":
		err = json.Unmarshal(option.ValueJSON, &config.ObfuscateSymbols)
	case "exports":
		err = json.Unmarshal(option.ValueJSON, &config.Exports)
	case "serviceName":
		err = json.Unmarshal(option.ValueJSON, &config.ServiceName)
	case "runAtLoad":
		err = json.Unmarshal(option.ValueJSON, &config.RunAtLoad)
	case "netgo":
		err = json.Unmarshal(option.ValueJSON, &config.NetGoEnabled)
	case "trafficEncoders":
		var names []string
		if err = json.Unmarshal(option.ValueJSON, &names); err == nil {
			config.TrafficEncodersEnabled = len(names) != 0
			for _, name := range names {
				config.Assets = append(config.Assets, &commonpb.File{Name: name})
			}
		}
	case "canaryDomains":
		err = json.Unmarshal(option.ValueJSON, &config.CanaryDomains)
	case "debugFile":
		err = json.Unmarshal(option.ValueJSON, &config.DebugFile)
	case "limitDomainJoined":
		err = json.Unmarshal(option.ValueJSON, &config.LimitDomainJoined)
	case "limitDatetime":
		err = json.Unmarshal(option.ValueJSON, &config.LimitDatetime)
	case "limitHostname":
		err = json.Unmarshal(option.ValueJSON, &config.LimitHostname)
	case "limitUsername":
		err = json.Unmarshal(option.ValueJSON, &config.LimitUsername)
	case "limitFileExists":
		err = json.Unmarshal(option.ValueJSON, &config.LimitFileExists)
	case "limitLocale":
		err = json.Unmarshal(option.ValueJSON, &config.LimitLocale)
	default:
		return fmt.Errorf("core option %q has no Sliver adapter", option.Name)
	}
	if err != nil {
		return fmt.Errorf("invalid %s option: %w", option.Name, err)
	}
	return nil
}

//nolint:gocyclo // Keep option decoding and platform validation with the compatibility field mapping.
func setSliverConverterOptions(config *clientpb.ImplantConfig, step buildplan.Step) error {
	config.ShellcodeConfig = &clientpb.ShellcodeConfig{}
	for _, option := range step.Options {
		switch option.Name {
		case "compress":
			var value bool
			if err := json.Unmarshal(option.ValueJSON, &value); err != nil {
				return err
			}
			config.ShellcodeConfig.Compress = 1
			if value {
				config.ShellcodeConfig.Compress = 2
			}
		case "entropy", "exitOption", "bypass", "headers", "originalEntryPoint":
			var value uint32
			if err := json.Unmarshal(option.ValueJSON, &value); err != nil {
				return err
			}
			switch option.Name {
			case "entropy":
				config.ShellcodeConfig.Entropy = value
			case "exitOption":
				config.ShellcodeConfig.ExitOpt = value
			case "bypass":
				config.ShellcodeConfig.Bypass = value
			case "headers":
				config.ShellcodeConfig.Headers = value
			case "originalEntryPoint":
				config.ShellcodeConfig.OEP = value
			}
		case "thread", "unicode":
			var value bool
			if err := json.Unmarshal(option.ValueJSON, &value); err != nil {
				return err
			}
			if option.Name == "thread" {
				config.ShellcodeConfig.Thread = value
			} else {
				config.ShellcodeConfig.Unicode = value
			}
		default:
			return fmt.Errorf("converter option %q has no legacy Sliver adapter", option.Name)
		}
	}
	if config.GOOS != "windows" {
		for _, option := range step.Options {
			if option.Name != "compress" {
				return fmt.Errorf("converter option %q is Windows-only", option.Name)
			}
		}
	}
	return nil
}
