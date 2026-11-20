// Package buildcatalog loads immutable build specifications and tracks their providers.
package buildcatalog

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed builtin/*.yaml
var builtinSpecs embed.FS

var (
	// ErrNotFound indicates that no spec matches a reference.
	ErrNotFound = errors.New("build spec not found")
	// ErrAmbiguous indicates that a partial reference matches more than one spec.
	ErrAmbiguous = errors.New("build spec reference is ambiguous")
	// ErrConflict indicates that the name and version already have a different definition.
	ErrConflict = errors.New("build spec name and version conflict")
)

var builtinExecutorIDs = []string{
	"builtin-sliver-go-v1",
	"minimal-sliver-v1-go",
	"donut-v1",
	"beignet-v1",
	"malasada-v1",
	"shikata-ga-nai-v1",
	"xor-v1",
	"xor-dynamic-v1",
	"stage-frame-v1",
	"stage-compress-v1",
	"stage-aes-v1",
	"stage-rc4-v1",
	"pe-metadata-v1",
}

// InstalledExecutorIDs returns the executor integrations shipped with the
// server. Open always allows these IDs in addition to its installed argument.
func InstalledExecutorIDs() []string {
	return append([]string(nil), builtinExecutorIDs...)
}

type record struct {
	spec      parsedSpec
	sources   map[string]Origin
	providers map[string]ProviderStatus
}

type persistedRecord struct {
	Kind      string           `json:"kind"`
	Reference Reference        `json:"reference"`
	RawYAML   []byte           `json:"rawYAML"`
	Sources   []Origin         `json:"sources"`
	Providers []ProviderStatus `json:"providers"`
}

type persistedCatalog struct {
	Version int               `json:"version"`
	Records []persistedRecord `json:"records"`
}

// Catalog owns immutable spec definitions and mutable provider status. Its
// snapshots are copied before returning to callers.
type Catalog struct {
	mu        sync.RWMutex
	path      string
	installed map[string]struct{}
	records   map[string]*record
}

// Open loads shipped specs, stored snapshots, and administrator YAML from
// <rootDir>/build-catalog/admin. Dropped remote connections start offline.
func Open(rootDir string, installed []string) (*Catalog, error) {
	if rootDir == "" {
		return nil, errors.New("catalog root directory is empty")
	}
	directory := filepath.Join(rootDir, "build-catalog")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("create catalog directory: %w", err)
	}
	catalog := &Catalog{
		path:      filepath.Join(directory, "snapshots.json"),
		installed: make(map[string]struct{}),
		records:   make(map[string]*record),
	}
	for _, id := range append(InstalledExecutorIDs(), installed...) {
		if !identifierRE.MatchString(id) {
			return nil, fmt.Errorf("invalid installed executor ID %q", id)
		}
		catalog.installed[id] = struct{}{}
	}
	if err := catalog.loadBuiltins(); err != nil {
		return nil, err
	}
	if err := catalog.load(); err != nil {
		return nil, err
	}
	if err := catalog.loadAdministratorSpecs(filepath.Join(directory, "admin")); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (catalog *Catalog) loadBuiltins() error {
	entries, err := fs.ReadDir(builtinSpecs, "builtin")
	if err != nil {
		return fmt.Errorf("read built-in spec directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		raw, err := builtinSpecs.ReadFile("builtin/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read built-in %s: %w", entry.Name(), err)
		}
		spec, err := parseSpec(raw)
		if err != nil {
			return fmt.Errorf("invalid built-in %s: %w", entry.Name(), err)
		}
		if err := catalog.checkInstalled(spec); err != nil {
			return fmt.Errorf("invalid built-in %s: %w", entry.Name(), err)
		}
		rec, err := catalog.upsert(spec)
		if err != nil {
			return err
		}
		rec.sources["builtin"] = Origin{Source: "builtin", Provider: "server"}
		rec.providers["server"] = ProviderStatus{Name: "server", Online: true}
	}
	return nil
}

func (catalog *Catalog) loadAdministratorSpecs(adminDir string) error {
	if err := os.MkdirAll(adminDir, 0700); err != nil {
		return fmt.Errorf("create administrator spec directory: %w", err)
	}
	adminEntries, err := os.ReadDir(adminDir)
	if err != nil {
		return fmt.Errorf("read administrator spec directory: %w", err)
	}
	for _, entry := range adminEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(adminDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read administrator spec %s: %w", entry.Name(), err)
		}
		spec, err := parseSpec(raw)
		if err != nil {
			return fmt.Errorf("import administrator spec %s: %w", entry.Name(), err)
		}
		if err := catalog.checkInstalled(spec); err != nil {
			stored := catalog.records[recordKey(spec.kind, spec.reference)]
			if stored != nil && stored.spec.reference.Digest == spec.reference.Digest {
				if _, known := stored.sources["administrator\x00server"]; known {
					// An integration can be removed without erasing its
					// previously imported immutable definition.
					continue
				}
			}
			return fmt.Errorf("import administrator spec %s: %w", entry.Name(), err)
		}
		if _, err := catalog.ImportAdministrator(raw); err != nil {
			return fmt.Errorf("import administrator spec %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func recordKey(_ string, ref Reference) string {
	return ref.Name + "\x00" + ref.Version
}

func (catalog *Catalog) upsert(spec parsedSpec) (*record, error) {
	key := recordKey(spec.kind, spec.reference)
	if existing, ok := catalog.records[key]; ok {
		if existing.spec.kind != spec.kind || existing.spec.reference.Digest != spec.reference.Digest {
			return nil, fmt.Errorf("%w: %s %s@%s", ErrConflict, spec.kind, spec.reference.Name, spec.reference.Version)
		}
		return existing, nil
	}
	rec := &record{spec: spec, sources: make(map[string]Origin), providers: make(map[string]ProviderStatus)}
	catalog.records[key] = rec
	return rec, nil
}

func (catalog *Catalog) checkInstalled(spec parsedSpec) error {
	id := ""
	if spec.core != nil {
		id = spec.core.Executor.ID
	} else {
		id = spec.transform.Executor.ID
	}
	if _, ok := catalog.installed[id]; !ok {
		return fmt.Errorf("executor %q is not installed", id)
	}
	return nil
}

// ImportAdministrator adds an installed server spec. The raw YAML is retained
// unchanged, and conflicting content at the same name/version is rejected.
func (catalog *Catalog) ImportAdministrator(raw []byte) (Snapshot, error) {
	spec, err := parseSpec(raw)
	if err != nil {
		return Snapshot{}, err
	}
	if err := catalog.checkInstalled(spec); err != nil {
		return Snapshot{}, err
	}
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	return catalog.importSpec(spec, "administrator", "server", time.Now().UTC())
}

// AdvertiseRemote stores an exact builder claim. A later disconnect changes
// provider status, never the immutable YAML definition.
func (catalog *Catalog) AdvertiseRemote(provider string, raw []byte, seen time.Time) (Snapshot, error) {
	if provider == "" || provider == "server" {
		return Snapshot{}, fmt.Errorf("invalid remote provider %q", provider)
	}
	spec, err := parseSpec(raw)
	if err != nil {
		return Snapshot{}, err
	}
	if err := catalog.checkInstalled(spec); err != nil {
		return Snapshot{}, err
	}
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	return catalog.importSpec(spec, "remote", provider, seen)
}

// AdvertiseRemoteBatch replaces a builder's active claims as one transaction.
// Definitions omitted from the new advertisement remain stored but offline.
// A malformed or conflicting member changes none of the builder's entries.
func (catalog *Catalog) AdvertiseRemoteBatch(provider string, raws [][]byte, seen time.Time) error {
	specs, err := catalog.parseRemoteBatch(provider, raws)
	if err != nil {
		return err
	}
	if seen.IsZero() {
		seen = time.Now().UTC()
	}
	seen = seen.UTC()
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	previous := catalog.records
	replacement := make(map[string]*record, len(previous)+len(specs))
	for key, rec := range previous {
		if status, exists := rec.providers[provider]; exists {
			offline := cloneRecord(rec)
			status.Online = false
			offline.providers[provider] = status
			replacement[key] = offline
		} else {
			replacement[key] = rec
		}
	}
	for _, spec := range specs {
		key := recordKey(spec.kind, spec.reference)
		rec, exists := replacement[key]
		if exists {
			if rec.spec.kind != spec.kind || rec.spec.reference.Digest != spec.reference.Digest {
				return fmt.Errorf("%w: %s %s@%s", ErrConflict, spec.kind, spec.reference.Name, spec.reference.Version)
			}
			rec = cloneRecord(rec)
		} else {
			rec = &record{spec: spec, sources: make(map[string]Origin), providers: make(map[string]ProviderStatus)}
		}
		rec.sources["remote\x00"+provider] = Origin{Source: "remote", Provider: provider, LastSeen: seen}
		rec.providers[provider] = ProviderStatus{Name: provider, Online: true, LastSeen: seen}
		replacement[key] = rec
	}
	catalog.records = replacement
	if err := catalog.save(); err != nil {
		catalog.records = previous
		return err
	}
	return nil
}

// PreviewRemoteBatch validates the exact candidate definitions without
// publishing them. A registration can check its claims against these
// snapshots before it makes a builder schedulable or changes catalog state.
// AdvertiseRemoteBatch repeats the conflict check under its write lock.
func (catalog *Catalog) PreviewRemoteBatch(provider string, raws [][]byte) (map[Reference]Snapshot, error) {
	specs, err := catalog.parseRemoteBatch(provider, raws)
	if err != nil {
		return nil, err
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	preview := make(map[Reference]Snapshot, len(specs))
	seen := make(map[string]parsedSpec, len(specs))
	for _, spec := range specs {
		key := recordKey(spec.kind, spec.reference)
		if prior, exists := seen[key]; exists && (prior.kind != spec.kind || prior.reference.Digest != spec.reference.Digest) {
			return nil, fmt.Errorf("%w: %s %s@%s", ErrConflict, spec.kind, spec.reference.Name, spec.reference.Version)
		}
		seen[key] = spec
		rec := catalog.records[key]
		if rec != nil {
			if rec.spec.kind != spec.kind || rec.spec.reference.Digest != spec.reference.Digest {
				return nil, fmt.Errorf("%w: %s %s@%s", ErrConflict, spec.kind, spec.reference.Name, spec.reference.Version)
			}
			preview[spec.reference] = snapshot(rec)
			continue
		}
		preview[spec.reference] = snapshot(&record{
			spec:      spec,
			sources:   map[string]Origin{"remote\x00" + provider: {Source: "remote", Provider: provider}},
			providers: map[string]ProviderStatus{provider: {Name: provider, Online: true}},
		})
	}
	return preview, nil
}

func (catalog *Catalog) parseRemoteBatch(provider string, raws [][]byte) ([]parsedSpec, error) {
	if provider == "" || provider == "server" {
		return nil, fmt.Errorf("invalid remote provider %q", provider)
	}
	specs := make([]parsedSpec, 0, len(raws))
	for index, raw := range raws {
		spec, err := parseSpec(raw)
		if err != nil {
			return nil, fmt.Errorf("remote spec %d: %w", index, err)
		}
		if err := catalog.checkInstalled(spec); err != nil {
			return nil, fmt.Errorf("remote spec %d: %w", index, err)
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func (catalog *Catalog) importSpec(spec parsedSpec, source, provider string, seen time.Time) (Snapshot, error) {
	key := recordKey(spec.kind, spec.reference)
	old, existed := catalog.records[key]
	if existed && (old.spec.kind != spec.kind || old.spec.reference.Digest != spec.reference.Digest) {
		return Snapshot{}, fmt.Errorf("%w: %s %s@%s", ErrConflict, spec.kind, spec.reference.Name, spec.reference.Version)
	}
	var rec *record
	if existed {
		rec = cloneRecord(old)
	} else {
		rec = &record{spec: spec, sources: make(map[string]Origin), providers: make(map[string]ProviderStatus)}
	}
	if seen.IsZero() {
		seen = time.Now().UTC()
	}
	seen = seen.UTC()
	rec.sources[source+"\x00"+provider] = Origin{Source: source, Provider: provider, LastSeen: seen}
	rec.providers[provider] = ProviderStatus{Name: provider, Online: true, LastSeen: seen}
	catalog.records[key] = rec
	if err := catalog.save(); err != nil {
		if existed {
			catalog.records[key] = old
		} else {
			delete(catalog.records, key)
		}
		return Snapshot{}, err
	}
	return snapshot(rec), nil
}

// SetRemoteOnline changes every advertised definition for a builder. Offline
// definitions remain available for inspection and persist across restarts.
func (catalog *Catalog) SetRemoteOnline(provider string, online bool, seen time.Time) error {
	if provider == "" || provider == "server" {
		return fmt.Errorf("invalid remote provider %q", provider)
	}
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	previous := make(map[string]*record)
	for key, rec := range catalog.records {
		status, ok := rec.providers[provider]
		if !ok {
			continue
		}
		previous[key] = rec
		replacement := cloneRecord(rec)
		status.Online = online
		if !seen.IsZero() {
			status.LastSeen = seen.UTC()
		}
		replacement.providers[provider] = status
		catalog.records[key] = replacement
	}
	if len(previous) == 0 {
		return nil
	}
	if err := catalog.save(); err != nil {
		for key, rec := range previous {
			catalog.records[key] = rec
		}
		return err
	}
	return nil
}

func cloneRecord(rec *record) *record {
	copyRec := &record{spec: rec.spec, sources: make(map[string]Origin, len(rec.sources)), providers: make(map[string]ProviderStatus, len(rec.providers))}
	for key, source := range rec.sources {
		copyRec.sources[key] = source
	}
	for key, provider := range rec.providers {
		copyRec.providers[key] = provider
	}
	return copyRec
}

// List returns sorted immutable snapshots. Target filters match at least one
// declared capability; availability is independently reported.
func (catalog *Catalog) List(query Query) []Snapshot {
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	results := make([]Snapshot, 0, len(catalog.records))
	for _, rec := range catalog.records {
		if query.Kind != "" && rec.spec.kind != query.Kind || query.Name != "" && rec.spec.reference.Name != query.Name {
			continue
		}
		if !matchesTarget(rec.spec, query.TargetOS, query.TargetArch) {
			continue
		}
		value := snapshot(rec)
		if query.AvailableOnly && !value.Available {
			continue
		}
		results = append(results, value)
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Reference.Name != b.Reference.Name {
			return a.Reference.Name < b.Reference.Name
		}
		if a.Reference.Version != b.Reference.Version {
			return a.Reference.Version < b.Reference.Version
		}
		return a.Reference.Digest < b.Reference.Digest
	})
	return results
}

// Get resolves an exact or partial reference. A partial reference succeeds
// only when it identifies one definition across both spec kinds.
func (catalog *Catalog) Get(reference Reference) (Snapshot, error) {
	if reference.Name == "" {
		return Snapshot{}, errors.New("spec name is required")
	}
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	var match *record
	for _, rec := range catalog.records {
		candidate := rec.spec.reference
		if candidate.Name != reference.Name || reference.Version != "" && candidate.Version != reference.Version || reference.Digest != "" && candidate.Digest != reference.Digest {
			continue
		}
		if match != nil {
			return Snapshot{}, fmt.Errorf("%w: %s", ErrAmbiguous, reference.Name)
		}
		match = rec
	}
	if match == nil {
		return Snapshot{}, fmt.Errorf("%w: %s", ErrNotFound, reference.Name)
	}
	return snapshot(match), nil
}

func matchesTarget(spec parsedSpec, osName, arch string) bool {
	if osName == "" && arch == "" {
		return true
	}
	matches := func(targets []Target) bool {
		for _, target := range targets {
			if (osName == "" || target.OS == osName) && (arch == "" || target.Arch == arch) {
				return true
			}
		}
		return false
	}
	if spec.core != nil {
		for _, capability := range spec.core.Capabilities {
			if matches(capability.Targets) {
				return true
			}
		}
	} else {
		for _, capability := range spec.transform.Capabilities {
			if matches(capability.Targets) {
				return true
			}
		}
	}
	return false
}

func snapshot(rec *record) Snapshot {
	value := Snapshot{
		Kind:      rec.spec.kind,
		Reference: rec.spec.reference,
		RawYAML:   append([]byte(nil), rec.spec.raw...),
	}
	if rec.spec.core != nil {
		value.Core = cloneJSON(rec.spec.core)
	} else {
		value.Transform = cloneJSON(rec.spec.transform)
	}
	for _, source := range rec.sources {
		value.Sources = append(value.Sources, source)
	}
	sort.Slice(value.Sources, func(i, j int) bool {
		if value.Sources[i].Source != value.Sources[j].Source {
			return value.Sources[i].Source < value.Sources[j].Source
		}
		return value.Sources[i].Provider < value.Sources[j].Provider
	})
	for _, provider := range rec.providers {
		value.Providers = append(value.Providers, provider)
		value.Available = value.Available || provider.Online
	}
	sort.Slice(value.Providers, func(i, j int) bool { return value.Providers[i].Name < value.Providers[j].Name })
	return value
}

func cloneJSON[T any](original *T) *T {
	data, _ := json.Marshal(original)
	var clone T
	_ = json.Unmarshal(data, &clone)
	return &clone
}

func (catalog *Catalog) load() error {
	data, err := os.ReadFile(catalog.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read catalog snapshots: %w", err)
	}
	var persisted persistedCatalog
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&persisted); err != nil || persisted.Version != 1 {
		return fmt.Errorf("decode catalog snapshots: invalid or unsupported snapshot version: %v", err)
	}
	for _, item := range persisted.Records {
		if err := catalog.loadRecord(item); err != nil {
			return err
		}
	}
	return nil
}

func (catalog *Catalog) loadRecord(item persistedRecord) error {
	spec, err := parseSpec(item.RawYAML)
	if err != nil {
		return fmt.Errorf("invalid stored spec: %w", err)
	}
	if spec.kind != item.Kind || spec.reference != item.Reference {
		return errors.New("stored spec index does not match its YAML content")
	}
	rec, err := catalog.upsert(spec)
	if err != nil {
		return err
	}
	for _, source := range item.Sources {
		if source.Source != "administrator" && source.Source != "remote" {
			return fmt.Errorf("invalid persisted spec source %q", source.Source)
		}
		rec.sources[source.Source+"\x00"+source.Provider] = source
	}
	for _, provider := range item.Providers {
		if provider.Name == "server" {
			if _, ok := catalog.installed[executorID(spec)]; ok {
				rec.providers["server"] = ProviderStatus{Name: "server", Online: true, LastSeen: provider.LastSeen}
			}
			continue
		}
		provider.Online = false
		rec.providers[provider.Name] = provider
	}
	return nil
}

func executorID(spec parsedSpec) string {
	if spec.core != nil {
		return spec.core.Executor.ID
	}
	return spec.transform.Executor.ID
}

func (catalog *Catalog) persistedSnapshots() persistedCatalog {
	stored := persistedCatalog{Version: 1}
	for _, rec := range catalog.records {
		item := persistedRecord{Kind: rec.spec.kind, Reference: rec.spec.reference, RawYAML: rec.spec.raw}
		for _, source := range rec.sources {
			if source.Source != "builtin" {
				item.Sources = append(item.Sources, source)
			}
		}
		if len(item.Sources) == 0 {
			continue
		}
		sort.Slice(item.Sources, func(i, j int) bool {
			if item.Sources[i].Source != item.Sources[j].Source {
				return item.Sources[i].Source < item.Sources[j].Source
			}
			return item.Sources[i].Provider < item.Sources[j].Provider
		})
		for _, provider := range rec.providers {
			item.Providers = append(item.Providers, provider)
		}
		sort.Slice(item.Providers, func(i, j int) bool { return item.Providers[i].Name < item.Providers[j].Name })
		stored.Records = append(stored.Records, item)
	}
	sort.Slice(stored.Records, func(i, j int) bool {
		a, b := stored.Records[i], stored.Records[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Reference.Name != b.Reference.Name {
			return a.Reference.Name < b.Reference.Name
		}
		return a.Reference.Version < b.Reference.Version
	})
	return stored
}

func (catalog *Catalog) save() error {
	data, err := json.MarshalIndent(catalog.persistedSnapshots(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode catalog snapshots: %w", err)
	}
	directory := filepath.Dir(catalog.path)
	file, err := os.CreateTemp(directory, ".snapshots-*.tmp")
	if err != nil {
		return fmt.Errorf("create catalog snapshot: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
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
	if err := os.Rename(file.Name(), catalog.path); err != nil {
		return fmt.Errorf("replace catalog snapshots: %w", err)
	}
	return nil
}
