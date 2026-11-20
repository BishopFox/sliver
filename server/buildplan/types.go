package buildplan

import (
	"encoding/json"
	"fmt"
	"time"
)

// Target identifies an implant operating system and architecture.
type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Reference identifies one immutable catalog definition.
type Reference struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

// Artifact identifies bytes without relying on their filename.
type Artifact struct {
	Kind         string   `json:"kind"`
	Container    string   `json:"container"`
	Target       Target   `json:"target"`
	GoBuildMode  string   `json:"goBuildMode,omitempty"`
	EntrySymbols []string `json:"entrySymbols,omitempty"`
}

// Toolchain records both the declared compiler requirements and, once known,
// the identity of the selected installed toolchain.
type Toolchain struct {
	Driver       string   `json:"driver"`
	Requirements []string `json:"requirements,omitempty"`
	Identity     string   `json:"identity,omitempty"`
}

type Endpoint struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

// Runtime settings are optional for existing neutral callers. Profiles carry
// them explicitly so the frozen plan retains all authored Sliver C2 timing.
type C2Settings struct {
	ConnectionStrategy       string `json:"connectionStrategy"`
	ReconnectSeconds         uint64 `json:"reconnectSeconds"`
	PollTimeoutSeconds       uint64 `json:"pollTimeoutSeconds"`
	MaxConnectionErrors      uint32 `json:"maxConnectionErrors"`
	HTTPC2Profile            string `json:"httpC2Profile"`
	WireguardKeyExchangePort uint32 `json:"wireguardKeyExchangePort,omitempty"`
	WireguardTCPCommsPort    uint32 `json:"wireguardTCPCommsPort,omitempty"`
}

type BeaconSettings struct {
	IntervalSeconds uint64 `json:"intervalSeconds"`
	JitterSeconds   uint64 `json:"jitterSeconds"`
}

// Option holds a typed JSON value. Its name and value are checked against the
// immutable spec at plan time; it is never interpreted as an executable command.
type Option struct {
	Name      string          `json:"name"`
	ValueJSON json.RawMessage `json:"value"`
}

type ProviderPolicy struct {
	Mode    string `json:"mode"`
	Builder string `json:"builder,omitempty"`
}

// ProfileSnapshot freezes the canonical definition selected for a build.
// Migrated builds also retain the full legacy config through compatibility
// execution; this YAML is its operator-visible definition at build time.
type ProfileSnapshot struct {
	Name               string `json:"name"`
	Revision           uint64 `json:"revision"`
	YAML               []byte `json:"yaml"`
	SHA256             string `json:"sha256"`
	Migrated           bool   `json:"migrated,omitempty"`
	LegacyConfigSHA256 string `json:"legacyConfigSHA256,omitempty"`
	EncoderOverride    string `json:"encoderOverride,omitempty"`
}

type TransformRequest struct {
	Spec    Reference `json:"spec"`
	Options []Option  `json:"options,omitempty"`
}

// Request is the neutral, serializable request retained with every plan.
// Missing core name resolves to builtin/sliver. A specified version or digest
// must resolve uniquely before any compile begins.
type Request struct {
	Name             string             `json:"name,omitempty"`
	Target           Target             `json:"target"`
	Mode             string             `json:"mode"`
	C2               []Endpoint         `json:"c2,omitempty"`
	C2Settings       *C2Settings        `json:"c2Settings,omitempty"`
	BeaconSettings   *BeaconSettings    `json:"beaconSettings,omitempty"`
	IncludeProtocols []string           `json:"includeProtocols,omitempty"`
	FinalFormat      string             `json:"finalFormat"`
	Core             Reference          `json:"core"`
	CoreOutput       *Artifact          `json:"coreOutput,omitempty"`
	CoreOptions      []Option           `json:"coreOptions,omitempty"`
	Transforms       []TransformRequest `json:"transforms,omitempty"`
	Provider         ProviderPolicy     `json:"provider"`
	ProfileSnapshot  *ProfileSnapshot   `json:"profileSnapshot,omitempty"`
}

// Step is an immutable resolved operation. The provider is a concrete server
// or connected builder, while the chosen catalog spec remains immutable even
// if that provider later disconnects.
type Step struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Spec      Reference `json:"spec"`
	Provider  string    `json:"provider"`
	Input     *Artifact `json:"input,omitempty"`
	Output    Artifact  `json:"output"`
	Toolchain Toolchain `json:"toolchain"`
	Options   []Option  `json:"options,omitempty"`
}

type Plan struct {
	Request       Request  `json:"request"`
	Steps         []Step   `json:"steps"`
	FinalArtifact Artifact `json:"finalArtifact"`
	SHA256        string   `json:"sha256"`
}

type StepState string

const (
	StepPending   StepState = "pending"
	StepRunning   StepState = "running"
	StepComplete  StepState = "complete"
	StepFailed    StepState = "failed"
	StepCancelled StepState = "cancelled"
)

type BuildState string

const (
	BuildPending   BuildState = "pending"
	BuildRunning   BuildState = "running"
	BuildComplete  BuildState = "complete"
	BuildFailed    BuildState = "failed"
	BuildCancelled BuildState = "cancelled"
)

// ArtifactInfo describes bytes actually produced by a step. SHA256 covers the
// full stored byte stream, including any ZIP container or final transform.
type ArtifactInfo struct {
	Type   Artifact `json:"type"`
	Name   string   `json:"name"`
	Size   int64    `json:"size"`
	SHA256 string   `json:"sha256"`
}

type StepRecord struct {
	ID                string         `json:"id"`
	State             StepState      `json:"state"`
	Progress          int            `json:"progress"`
	Failure           string         `json:"failure,omitempty"`
	ToolchainIdentity string         `json:"toolchainIdentity,omitempty"`
	Artifact          *ArtifactInfo  `json:"artifact,omitempty"`
	Selection         *StepSelection `json:"selection,omitempty"`
	StartedAt         *time.Time     `json:"startedAt,omitempty"`
	FinishedAt        *time.Time     `json:"finishedAt,omitempty"`
}

// StepSelection freezes the capacity report used for one remote assignment.
// Memory is recorded in bytes as reported, and as rounded decimal MB used for
// ranking. ReservedMemoryBytes is capacity already committed before this step.
type StepSelection struct {
	Builder                 string               `json:"builder"`
	ReportedAt              time.Time            `json:"reportedAt"`
	AvailableMemoryBytes    uint64               `json:"availableMemoryBytes"`
	ReservedMemoryBytes     uint64               `json:"reservedMemoryBytes"`
	RoundedAvailableMB      uint64               `json:"roundedAvailableMB"`
	EffectiveCPUCores       uint32               `json:"effectiveCpuCores"`
	RequiredPeakMemoryBytes uint64               `json:"requiredPeakMemoryBytes"`
	Decision                string               `json:"decision"`
	Candidates              []SelectionCandidate `json:"candidates,omitempty"`
}

// SelectionCandidate preserves each eligible builder in the order considered
// when a remote step was reserved.
type SelectionCandidate struct {
	Name                 string    `json:"name"`
	ReportedAt           time.Time `json:"reportedAt"`
	AvailableMemoryBytes uint64    `json:"availableMemoryBytes"`
	ReservedMemoryBytes  uint64    `json:"reservedMemoryBytes"`
	RoundedAvailableMB   uint64    `json:"roundedAvailableMB"`
	EffectiveCPUCores    uint32    `json:"effectiveCpuCores"`
	ToolchainIdentity    string    `json:"toolchainIdentity"`
}

// FinalRevision records an update to completed bytes. Compatibility metadata
// spoofing creates a new revision while preserving the immutable build plan.
type FinalRevision struct {
	Number    int          `json:"number"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Artifact  ArtifactInfo `json:"artifact"`
}

// Record freezes both the original neutral request and the resolved plan.
// Later catalog/profile edits cannot rewrite the plan or completed bytes.
type Record struct {
	ID            string          `json:"id"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	State         BuildState      `json:"state"`
	Plan          Plan            `json:"plan"`
	Steps         []StepRecord    `json:"steps"`
	Failure       string          `json:"failure,omitempty"`
	Final         *ArtifactInfo   `json:"final,omitempty"`
	FinalRevision int             `json:"finalRevision,omitempty"`
	FinalHistory  []FinalRevision `json:"finalHistory,omitempty"`
}

type ErrorCode string

const (
	CodeInvalidRequest       ErrorCode = "invalid-request"
	CodeUnknownSpec          ErrorCode = "unknown-spec"
	CodeAmbiguousSpec        ErrorCode = "ambiguous-spec"
	CodeIncompatible         ErrorCode = "incompatible-capability"
	CodeAmbiguousRoute       ErrorCode = "ambiguous-route"
	CodeUnavailableProvider  ErrorCode = "unavailable-provider"
	CodeUnsupportedExecution ErrorCode = "unsupported-execution"
)

// PlanError is suitable for an RPC status message without exposing catalog
// internals or falling back to an executable on unknown output formats.
type PlanError struct {
	Code   ErrorCode
	Step   string
	Detail string
}

func (e *PlanError) Error() string {
	if e.Step == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	}
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Step, e.Detail)
}
