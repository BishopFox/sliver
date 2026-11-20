package buildscheduler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/server/buildplan"
)

// DefaultStatusTTL bounds how long a builder's last capacity report may be
// used for a new assignment. Reports are timestamped by the server.
const DefaultStatusTTL = 30 * time.Second

// A single builder process cannot have more effective logical cores than this.
// It also bounds bogus status reports before they affect scheduler ordering.
const MaxEffectiveCPUCores = 65_536

var (
	ErrDuplicateBuilder = errors.New("builder is already connected")
	ErrUnknownBuilder   = errors.New("builder is not registered")
	ErrNoBuilder        = errors.New("no fresh compatible builder has available capacity")
)

// Registration contains only immutable support facts for one connection.
// Dynamic status reports cannot add a spec or capability to this definition.
type Registration struct {
	Name  string
	Specs []SpecSupport
}

type SpecSupport struct {
	Reference    buildplan.Reference
	Capabilities []Capability
}

// Capability is one exact target and artifact tuple. Input is nil for a core.
type Capability struct {
	Target    buildplan.Target
	Mode      string
	Protocols []string
	Input     *buildplan.Artifact
	Output    buildplan.Artifact
	Toolchain Toolchain
}

type Toolchain struct {
	Driver       string
	Requirements []string
}

// ToolchainStatus is a live report. A capability is dispatchable only when
// its matching installed toolchain is currently available.
type ToolchainStatus struct {
	Target       buildplan.Target
	Driver       string
	Requirements []string
	Identity     string
	Available    bool
}

type Status struct {
	EffectiveCPUCores    int
	AvailableMemoryBytes uint64
	Toolchains           []ToolchainStatus
}

// Requirement describes the already resolved step, not a requested route.
// All fields must match an immutable registered capability.
type Requirement struct {
	Spec      buildplan.Reference
	Target    buildplan.Target
	Mode      string
	Protocols []string
	Input     *buildplan.Artifact
	Output    buildplan.Artifact
	Toolchain Toolchain
}

type Candidate struct {
	Name                 string
	ReportedAt           time.Time
	ReportAge            time.Duration
	EffectiveCPUCores    int
	ReportedMemoryBytes  uint64
	ReservedMemoryBytes  uint64
	AvailableMemoryBytes uint64
	RoundedAvailableMB   uint64
	ToolchainIdentity    string
}

// Decision freezes the report and ordered candidates considered for a step.
// It can be stored alongside the resulting build step for diagnostics.
type Decision struct {
	Selected        string
	Pin             string
	PeakMemoryBytes uint64
	Candidates      []Candidate
}

type Reservation struct {
	ID       string
	Builder  string
	Decision Decision
}

// InventoryEntry combines immutable support with the latest live report.
// Connected and Fresh are separate so operators can distinguish stale data.
type InventoryEntry struct {
	Name                 string
	Connected            bool
	Fresh                bool
	Exclusive            bool
	ReportedAt           time.Time
	ReportAge            time.Duration
	EffectiveCPUCores    int
	ReportedMemoryBytes  uint64
	ReservedMemoryBytes  uint64
	AvailableMemoryBytes uint64
	RoundedAvailableMB   uint64
	Specs                []SpecSupport
	Toolchains           []ToolchainStatus
}

type builderState struct {
	registration Registration
	status       Status
	reportedAt   time.Time
	connected    bool
	reserved     uint64
	exclusive    bool
}

type reservationState struct {
	builder   string
	bytes     uint64
	exclusive bool
}

// Scheduler serializes report updates, selection, and capacity reservations.
type Scheduler struct {
	mu           sync.Mutex
	ttl          time.Duration
	builders     map[string]*builderState
	reservations map[string]reservationState
}

func New(ttl time.Duration) *Scheduler {
	if ttl <= 0 {
		ttl = DefaultStatusTTL
	}
	return &Scheduler{ttl: ttl, builders: make(map[string]*builderState), reservations: make(map[string]reservationState)}
}

// Register binds immutable support to a connection and accepts its first
// server-timestamped status report. A connected name cannot be replaced.
func (s *Scheduler) Register(reg Registration, status Status, now time.Time) error {
	if err := validateRegistration(reg); err != nil {
		return err
	}
	if err := validateStatus(status); err != nil {
		return err
	}
	if s == nil {
		return errors.New("nil builder scheduler")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old := s.builders[reg.Name]; old != nil && old.connected {
		return ErrDuplicateBuilder
	}
	s.builders[reg.Name] = &builderState{
		registration: cloneRegistration(reg), status: cloneStatus(status), reportedAt: now, connected: true,
	}
	return nil
}

// UpdateStatus changes only capacity and installed toolchain availability.
func (s *Scheduler) UpdateStatus(name string, status Status, now time.Time) error {
	if err := validateStatus(status); err != nil {
		return err
	}
	if s == nil {
		return ErrUnknownBuilder
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.builders[name]
	if state == nil || !state.connected {
		return ErrUnknownBuilder
	}
	state.status = cloneStatus(status)
	state.reportedAt = now
	return nil
}

// Disconnect removes the builder from selection immediately and cancels its
// outstanding capacity reservations. The immutable support remains visible
// in inventory until a new registration replaces it.
func (s *Scheduler) Disconnect(name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if state := s.builders[name]; state != nil {
		state.connected = false
		state.reserved = 0
		state.exclusive = false
	}
	for id, reservation := range s.reservations {
		if reservation.builder == name {
			delete(s.reservations, id)
		}
	}
}

// CanRun checks exact support, fresh status, and current toolchain availability.
// Capacity is checked atomically when SelectAndReserve is called.
func (s *Scheduler) CanRun(name string, requirement Requirement, now time.Time) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.builders[name]
	return state != nil && !state.exclusive && remaining(state) > 0 && s.canRun(state, requirement, now)
}

// Candidates lists compatible fresh builders in scheduler order using memory
// remaining after active reservations. A pin narrows the list to one builder.
func (s *Scheduler) Candidates(requirement Requirement, pin string, now time.Time) []Candidate {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidates(requirement, pin, now, 0, false)
}

// Eligible applies the same capacity check used for assignment without
// mutating reservations. A zero estimate means an exclusive single step.
func (s *Scheduler) Eligible(requirement Requirement, pin string, peakBytes uint64, now time.Time) []Candidate {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidates(requirement, pin, now, peakBytes, peakBytes == 0)
}

// SelectAndReserve selects and commits capacity under one lock. If a spec has
// no memory estimate, the selected builder is reserved exclusively. The returned
// decision includes the report used for ranking and the competing candidates.
func (s *Scheduler) SelectAndReserve(requirement Requirement, pin string, peakBytes uint64, now time.Time) (Reservation, error) {
	if s == nil {
		return Reservation{}, ErrNoBuilder
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exclusive := peakBytes == 0
	candidates := s.candidates(requirement, pin, now, peakBytes, exclusive)
	if len(candidates) == 0 {
		return Reservation{}, ErrNoBuilder
	}
	id, err := randomID()
	if err != nil {
		return Reservation{}, fmt.Errorf("create builder reservation: %w", err)
	}
	selected := candidates[0].Name
	state := s.builders[selected]
	if exclusive {
		peakBytes = remaining(state)
		state.exclusive = true
	}
	state.reserved += peakBytes // capacity was checked under the same lock.
	s.reservations[id] = reservationState{builder: selected, bytes: peakBytes, exclusive: exclusive}
	return Reservation{ID: id, Builder: selected, Decision: Decision{
		Selected: selected, Pin: pin, PeakMemoryBytes: peakBytes, Candidates: candidates,
	}}, nil
}

// Release is idempotent and returns whether a live reservation was removed.
func (s *Scheduler) Release(id string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	reservation, ok := s.reservations[id]
	if !ok {
		return false
	}
	delete(s.reservations, id)
	if state := s.builders[reservation.builder]; state != nil {
		state.reserved -= reservation.bytes
		if reservation.exclusive {
			state.exclusive = false
		}
	}
	return true
}

func (s *Scheduler) Inventory(now time.Time) []InventoryEntry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := make([]InventoryEntry, 0, len(s.builders))
	for name, state := range s.builders {
		available := remaining(state)
		entries = append(entries, InventoryEntry{
			Name: name, Connected: state.connected, Fresh: s.fresh(state, now), Exclusive: state.exclusive,
			ReportedAt: state.reportedAt, ReportAge: reportAge(state.reportedAt, now),
			EffectiveCPUCores:   state.status.EffectiveCPUCores,
			ReportedMemoryBytes: state.status.AvailableMemoryBytes,
			ReservedMemoryBytes: state.reserved, AvailableMemoryBytes: available,
			RoundedAvailableMB: roundedDecimalMB(available),
			Specs:              cloneSpecs(state.registration.Specs), Toolchains: cloneToolchains(state.status.Toolchains),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}

func (s *Scheduler) candidates(requirement Requirement, pin string, now time.Time, peakBytes uint64, exclusive bool) []Candidate {
	result := make([]Candidate, 0, len(s.builders))
	for name, state := range s.builders {
		if pin != "" && pin != name || !s.canRun(state, requirement, now) || state.exclusive || exclusive && state.reserved > 0 {
			continue
		}
		available := remaining(state)
		if available == 0 || available < peakBytes {
			continue
		}
		identity := toolchainIdentity(state.status.Toolchains, requirement.Target, requirement.Toolchain)
		result = append(result, Candidate{
			Name: name, ReportedAt: state.reportedAt, ReportAge: reportAge(state.reportedAt, now),
			EffectiveCPUCores:   state.status.EffectiveCPUCores,
			ReportedMemoryBytes: state.status.AvailableMemoryBytes, ReservedMemoryBytes: state.reserved,
			AvailableMemoryBytes: available, RoundedAvailableMB: roundedDecimalMB(available),
			ToolchainIdentity: identity,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RoundedAvailableMB != result[j].RoundedAvailableMB {
			return result[i].RoundedAvailableMB > result[j].RoundedAvailableMB
		}
		if result[i].EffectiveCPUCores != result[j].EffectiveCPUCores {
			return result[i].EffectiveCPUCores > result[j].EffectiveCPUCores
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func (s *Scheduler) canRun(state *builderState, requirement Requirement, now time.Time) bool {
	if !s.fresh(state, now) {
		return false
	}
	if toolchainIdentity(state.status.Toolchains, requirement.Target, requirement.Toolchain) == "" {
		return false
	}
	for _, spec := range state.registration.Specs {
		if spec.Reference != requirement.Spec {
			continue
		}
		for _, cap := range spec.Capabilities {
			if cap.Target == requirement.Target && cap.Mode == requirement.Mode &&
				allRequirementsPresent(requirement.Protocols, cap.Protocols) &&
				artifactPointerSupports(cap.Input, requirement.Input) &&
				artifactSupports(cap.Output, requirement.Output) && toolchainEqual(cap.Toolchain, requirement.Toolchain) {
				return true
			}
		}
	}
	return false
}

func (s *Scheduler) fresh(state *builderState, now time.Time) bool {
	return state.connected && !state.reportedAt.IsZero() && reportAge(state.reportedAt, now) <= s.ttl
}

func reportAge(reportedAt, now time.Time) time.Duration {
	age := now.Sub(reportedAt)
	if age < 0 {
		return 0
	}
	return age
}

func remaining(state *builderState) uint64 {
	if state.status.AvailableMemoryBytes <= state.reserved {
		return 0
	}
	return state.status.AvailableMemoryBytes - state.reserved
}

// roundedDecimalMB uses a decimal megabyte (1,000,000 bytes) and half-up
// rounding without floating-point conversion or uint64 overflow.
func roundedDecimalMB(bytes uint64) uint64 {
	mb := bytes / 1_000_000
	if bytes%1_000_000 >= 500_000 {
		mb++
	}
	return mb
}

func toolchainIdentity(statuses []ToolchainStatus, target buildplan.Target, required Toolchain) string {
	for _, status := range statuses {
		if status.Available && status.Target == target && status.Driver == required.Driver &&
			allRequirementsPresent(required.Requirements, status.Requirements) && status.Identity != "" {
			return status.Identity
		}
	}
	return ""
}

func allRequirementsPresent(required, present []string) bool {
	for _, need := range required {
		if !slices.Contains(present, need) {
			return false
		}
	}
	return true
}

func toolchainEqual(a, b Toolchain) bool {
	return a.Driver == b.Driver && sameStringSet(a.Requirements, b.Requirements)
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	a = slices.Clone(a)
	b = slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func artifactPointerSupports(a, b *buildplan.Artifact) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return artifactSupports(*a, *b)
}

// The spec's empty entry-symbol list means the resolved plan may fill symbols
// from validated options or carry them through a same-type transform.
func artifactSupports(a, b buildplan.Artifact) bool {
	return a.Kind == b.Kind && a.Container == b.Container && a.Target == b.Target &&
		a.GoBuildMode == b.GoBuildMode && (len(a.EntrySymbols) == 0 || sameStringSet(a.EntrySymbols, b.EntrySymbols))
}

func validateRegistration(reg Registration) error {
	if reg.Name == "" || reg.Name == "server" || len(reg.Name) > 128 ||
		strings.TrimSpace(reg.Name) != reg.Name || strings.ContainsAny(reg.Name, "\x00\r\n") {
		return errors.New("builder requires a valid non-server name")
	}
	if len(reg.Specs) == 0 {
		return errors.New("builder must advertise at least one exact spec")
	}
	seen := make(map[buildplan.Reference]bool)
	for _, spec := range reg.Specs {
		ref := spec.Reference
		if ref.Name == "" || ref.Version == "" || !validDigest(ref.Digest) {
			return fmt.Errorf("invalid exact spec reference %q@%q", ref.Name, ref.Version)
		}
		if seen[ref] {
			return fmt.Errorf("duplicate spec %s@%s", ref.Name, ref.Version)
		}
		seen[ref] = true
		if len(spec.Capabilities) == 0 {
			return fmt.Errorf("spec %s has no capability tuples", ref.Name)
		}
		for _, cap := range spec.Capabilities {
			if cap.Target.OS == "" || cap.Target.Arch == "" || cap.Output.Kind == "" || cap.Output.Container == "" || cap.Toolchain.Driver == "" || cap.Input == nil && cap.Mode == "" {
				return fmt.Errorf("spec %s has an incomplete capability tuple", ref.Name)
			}
		}
	}
	return nil
}

func validDigest(digest string) bool {
	if len(digest) != len("sha256:")+64 || digest[:len("sha256:")] != "sha256:" {
		return false
	}
	for _, ch := range digest[len("sha256:"):] {
		if ch < '0' || ch > '9' && ch < 'a' || ch > 'f' {
			return false
		}
	}
	return true
}

func validateStatus(status Status) error {
	if status.EffectiveCPUCores <= 0 || status.EffectiveCPUCores > MaxEffectiveCPUCores {
		return errors.New("builder status has an impossible effective CPU core count")
	}
	for _, toolchain := range status.Toolchains {
		if toolchain.Target.OS == "" || toolchain.Target.Arch == "" || toolchain.Driver == "" || toolchain.Available && toolchain.Identity == "" {
			return errors.New("builder status has invalid toolchain")
		}
	}
	return nil
}

func cloneRegistration(reg Registration) Registration {
	return Registration{Name: reg.Name, Specs: cloneSpecs(reg.Specs)}
}

func cloneSpecs(specs []SpecSupport) []SpecSupport {
	result := make([]SpecSupport, len(specs))
	for i, spec := range specs {
		result[i].Reference = spec.Reference
		result[i].Capabilities = make([]Capability, len(spec.Capabilities))
		for j, cap := range spec.Capabilities {
			result[i].Capabilities[j] = cap
			result[i].Capabilities[j].Protocols = slices.Clone(cap.Protocols)
			if cap.Input != nil {
				input := cloneArtifact(*cap.Input)
				result[i].Capabilities[j].Input = &input
			}
			result[i].Capabilities[j].Output = cloneArtifact(cap.Output)
			result[i].Capabilities[j].Toolchain.Requirements = slices.Clone(cap.Toolchain.Requirements)
		}
	}
	return result
}

func cloneArtifact(artifact buildplan.Artifact) buildplan.Artifact {
	artifact.EntrySymbols = slices.Clone(artifact.EntrySymbols)
	return artifact
}

func cloneStatus(status Status) Status {
	status.Toolchains = cloneToolchains(status.Toolchains)
	return status
}

func cloneToolchains(toolchains []ToolchainStatus) []ToolchainStatus {
	result := make([]ToolchainStatus, len(toolchains))
	for i, toolchain := range toolchains {
		result[i] = toolchain
		result[i].Requirements = slices.Clone(toolchain.Requirements)
	}
	return result
}

func randomID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
