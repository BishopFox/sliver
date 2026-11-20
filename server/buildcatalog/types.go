package buildcatalog

import (
	"encoding/json"
	"time"
)

// Reference identifies one immutable spec definition. Empty version or digest
// fields are permitted in a lookup, but the lookup must resolve uniquely.
type Reference struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

// Target identifies an operating system and architecture.
type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Artifact identifies the contents of a build step independently from its
// filename. GoBuildMode is empty for non-Go and transformed artifacts.
type Artifact struct {
	Kind         string   `json:"kind"`
	Container    string   `json:"container"`
	GoBuildMode  string   `json:"goBuildMode,omitempty"`
	EntrySymbols []string `json:"entrySymbols,omitempty"`
}

// Toolchain names the driver and requirements for a capability.
type Toolchain struct {
	Driver       string   `json:"driver"`
	Requirements []string `json:"requirements"`
}

// Resources estimates the memory needed to execute a capability.
type Resources struct {
	EstimatedPeakMemoryBytes uint64 `json:"estimatedPeakMemoryBytes"`
}

// OptionDefinition describes one option and its allowed values.
type OptionDefinition struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Required    bool            `json:"required"`
	Description string          `json:"description,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
	Values      []string        `json:"values,omitempty"`
	Minimum     *float64        `json:"minimum,omitempty"`
	Maximum     *float64        `json:"maximum,omitempty"`
}

// Runtime identifies the protocol and commands supported by a core.
type Runtime struct {
	Protocol       string   `json:"protocol"`
	CommandSource  string   `json:"commandSource"`
	CommandCatalog string   `json:"commandCatalog,omitempty"`
	Commands       []string `json:"commands,omitempty"`
}

// Executor identifies the installed integration that implements a spec.
type Executor struct {
	ID string `json:"id"`
}

// CoreSource describes preparation performed by an installed core executor.
// Copy is byte preserving; a renderer is an explicit, trusted opt-in.
type CoreSource struct {
	Mode     string `json:"mode"`
	Renderer string `json:"renderer,omitempty"`
}

// CoreBuilder selects a Magefile inside the installed executor's bundle.
// Magefile is a bundle-relative directory, never a host filesystem path.
type CoreBuilder struct {
	Engine   string `json:"engine"`
	Magefile string `json:"magefile"`
	Target   string `json:"target"`
}

// CoreOutput describes the produced artifact and supported output formats.
type CoreOutput struct {
	Artifact        Artifact `json:"artifact"`
	OperatorFormats []string `json:"operatorFormats"`
}

// CoreCapability describes a supported combination of target, runtime mode, and output.
type CoreCapability struct {
	Targets   []Target   `json:"targets"`
	Mode      string     `json:"mode"`
	Protocols []string   `json:"protocols"`
	Output    CoreOutput `json:"output"`
	Toolchain Toolchain  `json:"toolchain"`
	Resources *Resources `json:"resources,omitempty"`
}

// TransformCapability describes a supported input and output pairing for a target.
type TransformCapability struct {
	Targets             []Target   `json:"targets"`
	Input               Artifact   `json:"input"`
	Output              Artifact   `json:"output"`
	Toolchain           Toolchain  `json:"toolchain"`
	Resources           *Resources `json:"resources,omitempty"`
	RequiresEntrySymbol bool       `json:"requiresEntrySymbol,omitempty"`
}

// CoreSpec declares an immutable core definition and its capabilities.
type CoreSpec struct {
	APIVersion   string             `json:"apiVersion"`
	Kind         string             `json:"kind"`
	Metadata     Reference          `json:"metadata"`
	Runtime      Runtime            `json:"runtime"`
	Executor     Executor           `json:"executor"`
	Source       *CoreSource        `json:"source,omitempty"`
	Builder      *CoreBuilder       `json:"builder,omitempty"`
	Options      []OptionDefinition `json:"options"`
	Capabilities []CoreCapability   `json:"capabilities"`
}

// TransformSpec declares an immutable transform definition and its capabilities.
type TransformSpec struct {
	APIVersion   string                `json:"apiVersion"`
	Kind         string                `json:"kind"`
	Metadata     Reference             `json:"metadata"`
	Executor     Executor              `json:"executor"`
	Options      []OptionDefinition    `json:"options"`
	Capabilities []TransformCapability `json:"capabilities"`
}

// Origin records where a catalog definition was obtained.
type Origin struct {
	Source   string    `json:"source"`
	Provider string    `json:"provider,omitempty"`
	LastSeen time.Time `json:"lastSeen"`
}

// ProviderStatus records the availability of a spec provider.
type ProviderStatus struct {
	Name     string    `json:"name"`
	Online   bool      `json:"online"`
	LastSeen time.Time `json:"lastSeen"`
}

// Snapshot is a copy of a catalog definition and its current provider state.
type Snapshot struct {
	Kind      string
	Reference Reference
	RawYAML   []byte
	Core      *CoreSpec
	Transform *TransformSpec
	Sources   []Origin
	Providers []ProviderStatus
	Available bool
}

// Query filters catalog snapshots by spec, target, and availability.
type Query struct {
	Kind          string
	Name          string
	TargetOS      string
	TargetArch    string
	AvailableOnly bool
}
