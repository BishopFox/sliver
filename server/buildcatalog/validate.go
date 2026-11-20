package buildcatalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
)

const specAPIVersion = "sliver.bishopfox.com/v1alpha1"

var allowedRequestNames = makeSet(strings.Fields(`
MsgPing MsgCallExtensionReq MsgCdReq MsgChmodReq MsgChownReq MsgChtimesReq
MsgCpReq MsgCurrentTokenOwnerReq MsgDeregisterWasmExtensionReq MsgDownloadReq
MsgEnvReq MsgExecWasmExtensionReq MsgExecuteAssemblyReq MsgExecuteChildrenReq
MsgExecuteReq MsgExecuteWindowsReq MsgGetPrivsReq MsgGrepReq MsgIfconfigReq
MsgImpersonateReq MsgInvokeExecuteAssemblyReq MsgInvokeGetSystemReq
MsgInvokeInProcExecuteAssemblyReq MsgInvokeMigrateReq MsgKillSessionReq
MsgListExtensionsReq MsgListWasmExtensionsReq MsgLsReq MsgMakeTokenReq
MsgMemfilesAddReq MsgMemfilesListReq MsgMemfilesRmReq MsgMkdirReq MsgMountReq
MsgMvReq MsgNetstatReq MsgPivotListenersReq MsgPivotStartListenerReq
MsgPivotStopListenerReq MsgPortfwdReq MsgProcessDumpReq MsgPsReq MsgPwdReq
MsgRPortfwdReq MsgReconfigureReq MsgRegisterExtensionReq
MsgRegisterWasmExtensionReq MsgRegistryCreateKeyReq MsgRegistryDeleteKeyReq
MsgRegistryListValuesReq MsgRegistryReadHiveReq MsgRegistryReadReq
MsgRegistrySubKeysListReq MsgRegistryWriteReq MsgRemoveServiceReq
MsgRevToSelfReq MsgRmReq MsgRportFwdListenersReq MsgRportFwdStartListenerReq
MsgRportFwdStopListenerReq MsgRunAsReq MsgSSHCommandReq MsgScreenshotReq
MsgServiceDetailReq MsgServicesReq MsgSetEnvReq MsgShellReq MsgShellResizeReq
MsgSideloadReq MsgSpawnDllReq MsgStartServiceByNameReq MsgStartServiceReq
MsgStopServiceReq MsgTaskReq MsgTerminateReq MsgUnsetEnvReq MsgUploadReq
MsgWGListForwardersReq MsgWGListSocksReq MsgWGStartPortFwdReq
MsgWGStartSocksReq MsgWGStopPortFwdReq MsgWGStopSocksReq
`))

func makeSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func distinctStrings(values []string) bool {
	return len(makeSet(values)) == len(values)
}

func validateReference(reference Reference) error {
	if !canonicalNameRE.MatchString(reference.Name) {
		return fmt.Errorf("invalid canonical spec name %q", reference.Name)
	}
	if !versionRE.MatchString(reference.Version) {
		return fmt.Errorf("invalid semantic version %q", reference.Version)
	}
	if !digestRE.MatchString(reference.Digest) {
		return fmt.Errorf("invalid SHA-256 digest %q", reference.Digest)
	}
	return nil
}

func validateCoreSpec(spec *CoreSpec) error {
	if spec.APIVersion != specAPIVersion || spec.Kind != "CoreSpec" {
		return errors.New("invalid CoreSpec kind or API version")
	}
	if err := validateReference(spec.Metadata); err != nil {
		return err
	}
	if err := validateExecutor(spec.Executor); err != nil {
		return err
	}
	if err := validateCoreBuild(spec.Source, spec.Builder); err != nil {
		return err
	}
	if err := validateOptions(spec.Options); err != nil {
		return err
	}
	if err := validateCoreRuntime(spec.Runtime, spec.Executor); err != nil {
		return err
	}
	if len(spec.Capabilities) == 0 {
		return errors.New("CoreSpec must declare at least one capability")
	}
	seenCapabilities := make(map[string]struct{}, len(spec.Capabilities))
	for index, capability := range spec.Capabilities {
		if err := validateCoreCapability(capability); err != nil {
			return fmt.Errorf("capability %d: %w", index, err)
		}
		encoded, _ := json.Marshal(capability)
		if _, exists := seenCapabilities[string(encoded)]; exists {
			return fmt.Errorf("duplicate core capability %d", index)
		}
		seenCapabilities[string(encoded)] = struct{}{}
	}
	return nil
}

func validateCoreBuild(source *CoreSource, builder *CoreBuilder) error {
	if source != nil || builder != nil {
		if source == nil || builder == nil {
			return errors.New("core requires both source and builder declarations for Mage")
		}
		if source.Mode != "copy" && source.Mode != "none" {
			return fmt.Errorf("invalid core source mode %q", source.Mode)
		}
		if source.Mode == "none" && source.Renderer != "" {
			return errors.New("source renderer requires copy mode")
		}
		if source.Renderer != "" && !identifierRE.MatchString(source.Renderer) {
			return fmt.Errorf("invalid source renderer %q", source.Renderer)
		}
		if builder.Engine != "mage" || !safeBundlePath(builder.Magefile) ||
			!mageTargetRE.MatchString(builder.Target) {
			return errors.New("invalid installed Mage builder declaration")
		}
	}
	return nil
}

func validateCoreRuntime(runtime Runtime, executor Executor) error {
	if runtime.Protocol != "sliver-v1" {
		return fmt.Errorf("unsupported runtime protocol %q", runtime.Protocol)
	}
	switch runtime.CommandSource {
	case "installed-catalog":
		if runtime.CommandCatalog != "builtin-sliver-v1" || len(runtime.Commands) != 0 || executor.ID != "builtin-sliver-go-v1" {
			return errors.New("unrecognized installed command catalog or executor pairing")
		}
	case "declared-subset":
		if runtime.CommandCatalog != "" || len(runtime.Commands) == 0 || !distinctStrings(runtime.Commands) {
			return errors.New("declared command subset must be nonempty, unique, and have no installed catalog")
		}
		for _, command := range runtime.Commands {
			if _, valid := allowedRequestNames[command]; !valid {
				return fmt.Errorf("unknown or response-only sliver-v1 command %q", command)
			}
		}
	default:
		return fmt.Errorf("invalid command source %q", runtime.CommandSource)
	}
	return nil
}

func validateTransformSpec(spec *TransformSpec) error {
	if spec.APIVersion != specAPIVersion || spec.Kind != "TransformSpec" {
		return errors.New("invalid TransformSpec kind or API version")
	}
	if err := validateReference(spec.Metadata); err != nil {
		return err
	}
	if err := validateExecutor(spec.Executor); err != nil {
		return err
	}
	if err := validateOptions(spec.Options); err != nil {
		return err
	}
	if len(spec.Capabilities) == 0 {
		return errors.New("TransformSpec must declare at least one capability")
	}
	seenCapabilities := make(map[string]struct{}, len(spec.Capabilities))
	for index, capability := range spec.Capabilities {
		if err := validateTransformCapability(capability); err != nil {
			return fmt.Errorf("capability %d: %w", index, err)
		}
		encoded, _ := json.Marshal(capability)
		if _, exists := seenCapabilities[string(encoded)]; exists {
			return fmt.Errorf("duplicate transform capability %d", index)
		}
		seenCapabilities[string(encoded)] = struct{}{}
	}
	return nil
}

func validateExecutor(executor Executor) error {
	if !identifierRE.MatchString(executor.ID) {
		return fmt.Errorf("invalid executor ID %q", executor.ID)
	}
	return nil
}

func safeBundlePath(value string) bool {
	if value == "" || path.IsAbs(value) || path.Clean(value) != value ||
		strings.ContainsAny(value, `\:`) {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validateOptions(options []OptionDefinition) error {
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		if !optionNameRE.MatchString(option.Name) {
			return fmt.Errorf("invalid option name %q", option.Name)
		}
		if _, exists := seen[option.Name]; exists {
			return fmt.Errorf("duplicate option name %q", option.Name)
		}
		seen[option.Name] = struct{}{}
		if err := validateOptionBounds(option); err != nil {
			return err
		}
		if err := validateOptionDefinition(option); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionBounds(option OptionDefinition) error {
	if option.Minimum != nil && option.Maximum != nil && *option.Minimum > *option.Maximum {
		return fmt.Errorf("option %q minimum exceeds maximum", option.Name)
	}
	for _, bound := range []*float64{option.Minimum, option.Maximum} {
		if bound != nil && (!safeFloat(*bound) || option.Type == "integer" && float64(int64(*bound)) != *bound) {
			return fmt.Errorf("option %q has invalid numeric bound", option.Name)
		}
	}
	return nil
}

func validateOptionDefinition(option OptionDefinition) error {
	switch option.Type {
	case "boolean", "string", "integer", "number", "enum", "string-list", "secret-ref", "input-artifact-ref":
	default:
		return fmt.Errorf("option %q has invalid type %q", option.Name, option.Type)
	}
	if option.Type == "enum" {
		if len(option.Values) == 0 || !distinctStrings(option.Values) {
			return fmt.Errorf("enum option %q needs distinct values", option.Name)
		}
	} else if len(option.Values) != 0 {
		return fmt.Errorf("option %q cannot declare enum values", option.Name)
	}
	if option.Type != "integer" && option.Type != "number" && (option.Minimum != nil || option.Maximum != nil) {
		return fmt.Errorf("option %q cannot declare numeric bounds", option.Name)
	}
	if option.Type == "secret-ref" || option.Type == "input-artifact-ref" {
		if len(option.Default) != 0 {
			return fmt.Errorf("reference option %q cannot have a default", option.Name)
		}
	}
	if len(option.Default) != 0 {
		return validateOptionDefault(option)
	}
	return nil
}

func validateOptionDefault(option OptionDefinition) error {
	var value string
	switch option.Type {
	case "string", "enum":
		if err := json.Unmarshal(option.Default, &value); err != nil {
			return fmt.Errorf("option %q default must be a string", option.Name)
		}
		if option.Type == "enum" {
			for _, allowed := range option.Values {
				if value == allowed {
					return nil
				}
			}
			return fmt.Errorf("option %q default is outside enum values", option.Name)
		}
	case "boolean":
		var boolean bool
		if err := json.Unmarshal(option.Default, &boolean); err != nil {
			return fmt.Errorf("option %q default must be boolean", option.Name)
		}
	case "integer", "number":
		return validateNumericDefault(option)
	case "string-list":
		var values []string
		if err := json.Unmarshal(option.Default, &values); err != nil || values == nil {
			return fmt.Errorf("option %q default must be a string list", option.Name)
		}
	}
	return nil
}

func validateNumericDefault(option OptionDefinition) error {
	if len(option.Default) == 0 || option.Default[0] == '"' {
		return fmt.Errorf("option %q default must be numeric", option.Name)
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(option.Default))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return fmt.Errorf("option %q default must be numeric", option.Name)
	}
	value, err := number.Float64()
	if err != nil || !safeFloat(value) {
		return fmt.Errorf("option %q default exceeds safe number range", option.Name)
	}
	if option.Type == "integer" && !decimalIntRE.MatchString(number.String()) {
		return fmt.Errorf("option %q default must be an integer", option.Name)
	}
	if option.Minimum != nil && value < *option.Minimum || option.Maximum != nil && value > *option.Maximum {
		return fmt.Errorf("option %q default is outside numeric bounds", option.Name)
	}
	return nil
}

func validateCoreCapability(capability CoreCapability) error {
	if capability.Mode != "session" && capability.Mode != "beacon" {
		return fmt.Errorf("invalid mode %q", capability.Mode)
	}
	if err := validateTargets(capability.Targets); err != nil {
		return err
	}
	if err := validateProtocols(capability.Protocols, capability.Mode, capability.Targets); err != nil {
		return err
	}
	if err := validateArtifact(capability.Output.Artifact); err != nil {
		return err
	}
	if err := validateArtifactTargets(capability.Output.Artifact, capability.Targets); err != nil {
		return err
	}
	if err := validateToolchain(capability.Toolchain, capability.Resources); err != nil {
		return err
	}
	if capability.Toolchain.Driver == "go" && capability.Output.Artifact.GoBuildMode == "" {
		return errors.New("core output requires goBuildMode when using the Go toolchain")
	}
	if !distinctStrings(capability.Output.OperatorFormats) {
		return errors.New("duplicate operator output format")
	}
	for _, format := range capability.Output.OperatorFormats {
		expectedKind := map[string]string{
			"exe": "executable", "service": "service-executable", "shared": "shared-library",
			"archive": "c-archive-bundle", "shellcode": "shellcode", "stage": "stage",
		}[format]
		if expectedKind == "" || capability.Output.Artifact.Kind != expectedKind {
			return fmt.Errorf("operator format %q does not match artifact kind %q", format, capability.Output.Artifact.Kind)
		}
	}
	return nil
}

func validateTransformCapability(capability TransformCapability) error {
	if err := validateTargets(capability.Targets); err != nil {
		return err
	}
	if err := validateArtifact(capability.Input); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	if err := validateArtifact(capability.Output); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	if err := validateArtifactTargets(capability.Input, capability.Targets); err != nil {
		return fmt.Errorf("input: %w", err)
	}
	if err := validateArtifactTargets(capability.Output, capability.Targets); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	return validateToolchain(capability.Toolchain, capability.Resources)
}

func validateTargets(targets []Target) error {
	if len(targets) == 0 {
		return errors.New("capability needs at least one target")
	}
	seen := make(map[Target]struct{}, len(targets))
	for _, target := range targets {
		if target.OS != "windows" && target.OS != "linux" && target.OS != "darwin" {
			return fmt.Errorf("unsupported target OS %q", target.OS)
		}
		if target.Arch != "386" && target.Arch != "amd64" && target.Arch != "arm64" || target.OS == "darwin" && target.Arch == "386" {
			return fmt.Errorf("unsupported target architecture %s/%s", target.OS, target.Arch)
		}
		if _, exists := seen[target]; exists {
			return fmt.Errorf("duplicate target %s/%s", target.OS, target.Arch)
		}
		seen[target] = struct{}{}
	}
	return nil
}

func validateProtocols(protocols []string, mode string, targets []Target) error {
	if len(protocols) == 0 || !distinctStrings(protocols) {
		return errors.New("capability protocols must be nonempty and unique")
	}
	valid := makeSet([]string{"mtls", "http", "https", "dns", "wireguard", "named-pipe", "tcp-pivot"})
	for _, protocol := range protocols {
		if _, known := valid[protocol]; !known {
			return fmt.Errorf("unknown protocol %q", protocol)
		}
		if mode == "beacon" && (protocol == "named-pipe" || protocol == "tcp-pivot") {
			return fmt.Errorf("%s is session-only", protocol)
		}
		if protocol == "named-pipe" {
			for _, target := range targets {
				if target.OS != "windows" {
					return errors.New("named-pipe is Windows-only")
				}
			}
		}
	}
	return nil
}

func validateArtifact(artifact Artifact) error {
	if !distinctStrings(artifact.EntrySymbols) {
		return errors.New("artifact has duplicate entry symbols")
	}
	for _, symbol := range artifact.EntrySymbols {
		if symbol == "" {
			return errors.New("artifact has empty entry symbol")
		}
	}
	shape, known := artifactShapes[artifact.Kind]
	if !known {
		return fmt.Errorf("unknown artifact kind %q", artifact.Kind)
	}
	if !oneOf(artifact.Container, shape.containers...) || !oneOf(artifact.GoBuildMode, shape.modes...) {
		return errors.New(shape.invalid)
	}
	return nil
}

var artifactShapes = map[string]struct {
	containers []string
	modes      []string
	invalid    string
}{
	"executable":         {[]string{"pe", "elf", "macho"}, []string{"", "default", "exe", "pie"}, "invalid executable artifact"},
	"service-executable": {[]string{"pe"}, []string{"", "default"}, "invalid service executable artifact"},
	"shared-library":     {[]string{"pe", "elf", "macho"}, []string{"", "c-shared"}, "invalid shared library artifact"},
	"go-archive":         {[]string{"ar"}, []string{"archive"}, "invalid Go archive artifact"},
	"c-archive":          {[]string{"ar"}, []string{"c-archive"}, "invalid C archive artifact"},
	"c-archive-bundle":   {[]string{"zip"}, []string{"c-archive"}, "invalid C archive bundle"},
	"go-plugin":          {[]string{"elf", "macho"}, []string{"plugin"}, "invalid Go plugin artifact"},
	"go-shared-library":  {[]string{"elf", "macho"}, []string{"shared"}, "invalid Go shared library artifact"},
	"shellcode":          {[]string{"raw"}, []string{""}, "invalid raw artifact"},
	"stage":              {[]string{"raw"}, []string{""}, "invalid raw artifact"},
}

func validateArtifactTargets(artifact Artifact, targets []Target) error {
	for _, target := range targets {
		switch artifact.Container {
		case "pe":
			if target.OS != "windows" {
				return errors.New("PE artifact requires Windows target")
			}
		case "elf":
			if target.OS != "linux" {
				return errors.New("ELF artifact requires Linux target")
			}
		case "macho":
			if target.OS != "darwin" {
				return errors.New("Mach-O artifact requires Darwin target")
			}
		}
		if artifact.Kind == "service-executable" && target.OS != "windows" {
			return errors.New("service output requires Windows target")
		}
	}
	return nil
}

func validateToolchain(toolchain Toolchain, resources *Resources) error {
	if !identifierRE.MatchString(toolchain.Driver) || !distinctStrings(toolchain.Requirements) {
		return errors.New("invalid toolchain driver or duplicate requirements")
	}
	for _, requirement := range toolchain.Requirements {
		if !identifierRE.MatchString(requirement) {
			return fmt.Errorf("invalid toolchain requirement %q", requirement)
		}
	}
	if resources != nil && (resources.EstimatedPeakMemoryBytes == 0 || resources.EstimatedPeakMemoryBytes > uint64(maxSafeInteger)) {
		return errors.New("invalid estimated peak memory")
	}
	return nil
}

func oneOf(value string, values ...string) bool {
	for _, allowed := range values {
		if value == allowed {
			return true
		}
	}
	return false
}
