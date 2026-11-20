package buildprofile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildplan"
	"gopkg.in/yaml.v3"
)

const MaxYAMLBytes = 1 << 20

var namePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var inputPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var builderPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ValidName(name string) bool { return len(name) <= 255 && namePattern.MatchString(name) }

type Metadata struct {
	Name     string `json:"name"`
	Revision uint64 `json:"revision"`
}

type Artifact struct {
	Kind         string   `json:"kind"`
	Container    string   `json:"container"`
	GoBuildMode  string   `json:"goBuildMode,omitempty"`
	EntrySymbols []string `json:"entrySymbols,omitempty"`
}

type C2 struct {
	Endpoints                []buildplan.Endpoint `json:"endpoints"`
	IncludeProtocols         []string             `json:"includeProtocols,omitempty"`
	ConnectionStrategy       string               `json:"connectionStrategy"`
	ReconnectSeconds         uint64               `json:"reconnectSeconds"`
	PollTimeoutSeconds       uint64               `json:"pollTimeoutSeconds"`
	MaxConnectionErrors      uint32               `json:"maxConnectionErrors"`
	HTTPC2Profile            string               `json:"httpC2Profile"`
	WireguardKeyExchangePort uint32               `json:"wireguardKeyExchangePort,omitempty"`
	WireguardTCPCommsPort    uint32               `json:"wireguardTCPCommsPort,omitempty"`
}

type Input struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Artifact *Artifact `json:"artifact,omitempty"`
}

type FinalOutput struct {
	Format   string   `json:"format"`
	Artifact Artifact `json:"artifact"`
}

type Provider struct {
	Policy  string `json:"policy"`
	Builder string `json:"builder,omitempty"`
}

type Profile struct {
	APIVersion     string                       `json:"apiVersion"`
	Kind           string                       `json:"kind"`
	Metadata       Metadata                     `json:"metadata"`
	Target         buildplan.Target             `json:"target"`
	Mode           string                       `json:"mode"`
	C2             C2                           `json:"c2"`
	Beacon         *buildplan.BeaconSettings    `json:"beacon,omitempty"`
	PerBuildInputs []Input                      `json:"perBuildInputs,omitempty"`
	Core           buildplan.Reference          `json:"core"`
	CoreOutput     Artifact                     `json:"coreOutput"`
	CoreOptions    []buildplan.Option           `json:"coreOptions"`
	Transforms     []buildplan.TransformRequest `json:"transforms"`
	FinalOutput    FinalOutput                  `json:"finalOutput"`
	Provider       Provider                     `json:"provider"`
}

// Parse accepts only one restricted YAML document and rejects unknown fields.
func Parse(raw []byte) (Profile, error) {
	data, err := buildcatalog.RestrictedYAMLJSON(raw, MaxYAMLBytes)
	if err != nil {
		return Profile{}, err
	}
	var profile Profile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("decode profile fields: %w", err)
	}
	if profile.CoreOptions == nil || profile.Transforms == nil {
		return Profile{}, errors.New("coreOptions and transforms arrays are required")
	}
	if profile.C2.HTTPC2Profile == "" {
		profile.C2.HTTPC2Profile = "default"
	}
	if err := profile.Validate(); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

// CanonicalYAML writes normalized, deterministic YAML without generated keys.
func CanonicalYAML(profile Profile) ([]byte, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	if profile.CoreOptions == nil {
		profile.CoreOptions = []buildplan.Option{}
	}
	if profile.Transforms == nil {
		profile.Transforms = []buildplan.TransformRequest{}
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(encoded, &node); err != nil {
		return nil, err
	}
	return yaml.Marshal(&node)
}

func (profile Profile) Validate() error {
	if profile.APIVersion != "sliver.bishopfox.com/v1alpha1" || profile.Kind != "BuildProfile" {
		return errors.New("unsupported BuildProfile version or kind")
	}
	if !ValidName(profile.Metadata.Name) || profile.Metadata.Revision == 0 || profile.Metadata.Revision > 9007199254740991 {
		return errors.New("invalid profile name or revision")
	}
	if !validTarget(profile.Target) || profile.Mode != "session" && profile.Mode != "beacon" {
		return errors.New("invalid target or mode")
	}
	if (profile.Mode == "beacon") != (profile.Beacon != nil) {
		return errors.New("beacon settings are required only for beacon mode")
	}
	if len(profile.C2.Endpoints) == 0 ||
		profile.C2.ConnectionStrategy != "sequential" && profile.C2.ConnectionStrategy != "random" && profile.C2.ConnectionStrategy != "random-domain" {
		return errors.New("invalid C2 settings")
	}
	protocols := map[string]bool{}
	for _, endpoint := range profile.C2.Endpoints {
		if !validProtocol(endpoint.Protocol) || endpoint.Address == "" ||
			profile.Mode == "beacon" && (endpoint.Protocol == "named-pipe" || endpoint.Protocol == "tcp-pivot") ||
			endpoint.Protocol == "named-pipe" && profile.Target.OS != "windows" {
			return errors.New("invalid C2 endpoint")
		}
	}
	for _, protocol := range profile.C2.IncludeProtocols {
		if !validProtocol(protocol) || protocols[protocol] ||
			profile.Mode == "beacon" && (protocol == "named-pipe" || protocol == "tcp-pivot") ||
			protocol == "named-pipe" && profile.Target.OS != "windows" {
			return errors.New("invalid or duplicate included C2 protocol")
		}
		protocols[protocol] = true
	}
	if profile.C2.WireguardKeyExchangePort > 65535 || profile.C2.WireguardTCPCommsPort > 65535 {
		return errors.New("invalid WireGuard port")
	}
	if profile.Provider.Policy != "automatic" && profile.Provider.Policy != "pinned" ||
		(profile.Provider.Policy == "pinned") != (profile.Provider.Builder != "") ||
		profile.Provider.Builder != "" && !builderPattern.MatchString(profile.Provider.Builder) {
		return errors.New("invalid provider policy")
	}
	if profile.Beacon != nil && profile.Beacon.IntervalSeconds == 0 {
		return errors.New("beacon interval must be positive")
	}
	if profile.Core.Name == "" || profile.Core.Version == "" || profile.Core.Digest == "" {
		return errors.New("core requires immutable name, version, and digest")
	}
	if err := validateArtifact(profile.CoreOutput); err != nil {
		return fmt.Errorf("coreOutput: %w", err)
	}
	if err := validateArtifact(profile.FinalOutput.Artifact); err != nil {
		return fmt.Errorf("finalOutput: %w", err)
	}
	if !formatKindMatches(profile.FinalOutput.Format, profile.FinalOutput.Artifact.Kind) {
		return errors.New("final output format and artifact kind disagree")
	}
	if err := validateOptions(profile.CoreOptions); err != nil {
		return fmt.Errorf("coreOptions: %w", err)
	}
	for _, transform := range profile.Transforms {
		if transform.Spec.Name == "" || transform.Spec.Version == "" || transform.Spec.Digest == "" {
			return errors.New("transform requires immutable name, version, and digest")
		}
		if err := validateOptions(transform.Options); err != nil {
			return fmt.Errorf("transform %s options: %w", transform.Spec.Name, err)
		}
	}
	seen := map[string]bool{}
	slotKinds := map[string]string{}
	for _, input := range profile.PerBuildInputs {
		if !inputPattern.MatchString(input.Name) || seen[input.Name] || input.Kind != "secret" && input.Kind != "input-artifact" ||
			(input.Kind == "input-artifact") != (input.Artifact != nil) {
			return errors.New("invalid or duplicate per-build input")
		}
		seen[input.Name] = true
		slotKinds[input.Name] = input.Kind
		if input.Artifact != nil {
			if err := validateArtifact(*input.Artifact); err != nil {
				return fmt.Errorf("input %s: %w", input.Name, err)
			}
		}
	}
	if err := validateOptionBindings(profile.CoreOptions, slotKinds); err != nil {
		return fmt.Errorf("coreOptions: %w", err)
	}
	for _, transform := range profile.Transforms {
		if err := validateOptionBindings(transform.Options, slotKinds); err != nil {
			return fmt.Errorf("transform %s: %w", transform.Spec.Name, err)
		}
	}
	return nil
}

func validateOptionBindings(options []buildplan.Option, slots map[string]string) error {
	for _, option := range options {
		value := bytes.TrimSpace(option.ValueJSON)
		if len(value) == 0 || value[0] != '{' {
			continue
		}
		var reference map[string]json.RawMessage
		if err := json.Unmarshal(value, &reference); err != nil {
			return fmt.Errorf("invalid input reference for %s: %w", option.Name, err)
		}
		kindJSON, hasKind := reference["kind"]
		nameJSON, hasName := reference["name"]
		if !hasKind && !hasName {
			continue // A catalog option may legitimately hold another object type.
		}
		var kind, name string
		if len(reference) != 2 || !hasKind || !hasName ||
			json.Unmarshal(kindJSON, &kind) != nil || json.Unmarshal(nameJSON, &name) != nil ||
			name == "" || (kind != "secret" && kind != "input-artifact") || slots[name] != kind {
			return fmt.Errorf("unbound per-build input reference in option %s", option.Name)
		}
	}
	return nil
}

func validTarget(target buildplan.Target) bool {
	switch target.OS {
	case "linux", "windows":
		return target.Arch == "386" || target.Arch == "amd64" || target.Arch == "arm64"
	case "darwin":
		return target.Arch == "amd64" || target.Arch == "arm64"
	}
	return false
}

func validateArtifact(artifact Artifact) error {
	allowed := false
	switch artifact.Kind {
	case "executable":
		allowed = isNativeContainer(artifact.Container) &&
			(artifact.GoBuildMode == "" || artifact.GoBuildMode == "default" || artifact.GoBuildMode == "exe" || artifact.GoBuildMode == "pie")
	case "service-executable":
		allowed = artifact.Container == "pe" && (artifact.GoBuildMode == "" || artifact.GoBuildMode == "default")
	case "shared-library":
		allowed = isNativeContainer(artifact.Container) && (artifact.GoBuildMode == "" || artifact.GoBuildMode == "c-shared")
	case "go-archive":
		allowed = artifact.Container == "ar" && artifact.GoBuildMode == "archive"
	case "c-archive":
		allowed = artifact.Container == "ar" && artifact.GoBuildMode == "c-archive"
	case "c-archive-bundle":
		allowed = artifact.Container == "zip" && artifact.GoBuildMode == "c-archive"
	case "go-plugin":
		allowed = (artifact.Container == "elf" || artifact.Container == "macho") && artifact.GoBuildMode == "plugin"
	case "go-shared-library":
		allowed = (artifact.Container == "elf" || artifact.Container == "macho") && artifact.GoBuildMode == "shared"
	case "shellcode", "stage":
		allowed = artifact.Container == "raw" && artifact.GoBuildMode == ""
	}
	if !allowed {
		return errors.New("incompatible artifact kind, container, or Go build mode")
	}
	seen := map[string]bool{}
	for _, symbol := range artifact.EntrySymbols {
		if symbol == "" || seen[symbol] {
			return errors.New("empty or duplicate entry symbol")
		}
		seen[symbol] = true
	}
	return nil
}

func isNativeContainer(container string) bool {
	return container == "pe" || container == "elf" || container == "macho"
}

func validProtocol(protocol string) bool {
	switch protocol {
	case "mtls", "http", "https", "dns", "wireguard", "named-pipe", "tcp-pivot":
		return true
	}
	return false
}

func formatKindMatches(format, kind string) bool {
	return map[string]string{"exe": "executable", "service": "service-executable", "shared": "shared-library", "archive": "c-archive-bundle", "shellcode": "shellcode", "stage": "stage"}[format] == kind
}

func validateOptions(options []buildplan.Option) error {
	seen := map[string]bool{}
	for _, option := range options {
		if option.Name == "" || seen[option.Name] || len(option.ValueJSON) == 0 || !json.Valid(option.ValueJSON) {
			return errors.New("invalid or duplicate option")
		}
		seen[option.Name] = true
	}
	return nil
}

func (profile Profile) Request(name string) buildplan.Request {
	request := buildplan.Request{
		Name: name, Target: profile.Target, Mode: profile.Mode,
		C2:               append([]buildplan.Endpoint(nil), profile.C2.Endpoints...),
		IncludeProtocols: append([]string(nil), profile.C2.IncludeProtocols...),
		C2Settings: &buildplan.C2Settings{
			ConnectionStrategy: profile.C2.ConnectionStrategy, ReconnectSeconds: profile.C2.ReconnectSeconds,
			PollTimeoutSeconds: profile.C2.PollTimeoutSeconds, MaxConnectionErrors: profile.C2.MaxConnectionErrors,
			HTTPC2Profile: profile.C2.HTTPC2Profile, WireguardKeyExchangePort: profile.C2.WireguardKeyExchangePort,
			WireguardTCPCommsPort: profile.C2.WireguardTCPCommsPort,
		},
		BeaconSettings: profile.Beacon, FinalFormat: profile.FinalOutput.Format,
		Core: profile.Core, CoreOptions: append([]buildplan.Option(nil), profile.CoreOptions...),
		Transforms: append([]buildplan.TransformRequest(nil), profile.Transforms...),
		Provider:   buildplan.ProviderPolicy{Mode: profile.Provider.Policy, Builder: profile.Provider.Builder},
	}
	core := profile.CoreOutput.PlanArtifact(profile.Target)
	request.CoreOutput = &core
	return request
}

func (artifact Artifact) PlanArtifact(target buildplan.Target) buildplan.Artifact {
	return buildplan.Artifact{Kind: artifact.Kind, Container: artifact.Container, Target: target,
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: append([]string(nil), artifact.EntrySymbols...)}
}

func ArtifactFromPlan(artifact buildplan.Artifact) Artifact {
	return Artifact{Kind: artifact.Kind, Container: artifact.Container, GoBuildMode: artifact.GoBuildMode,
		EntrySymbols: append([]string(nil), artifact.EntrySymbols...)}
}
