package buildprofile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("build profile not found")
var ErrRevision = errors.New("build profile revision must increase")

type Document struct {
	Profile        Profile `json:"-"`
	YAML           []byte  `json:"yaml"`
	Migrated       bool    `json:"migrated,omitempty"`
	LegacyConfigID string  `json:"legacyConfigID,omitempty"`
	LegacyConfig   []byte  `json:"legacyConfig,omitempty"`
}

type index struct {
	Version    int                 `json:"version"`
	Documents  map[string]Document `json:"documents"`
	Tombstones map[string]bool     `json:"tombstones"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data index
}

// OpenStore keeps canonical YAML in one atomic index under the server root.
// Names are map keys, never filesystem paths.
func OpenStore(root string) (*Store, error) {
	directory := filepath.Join(root, "build-profiles")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	store := &Store{path: filepath.Join(directory, "profiles.json"), data: index{
		Version: 1, Documents: map[string]Document{}, Tombstones: map[string]bool{},
	}}
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("decode build profiles: %w", err)
	}
	if store.data.Version != 1 {
		return nil, fmt.Errorf("unsupported build profile index version %d", store.data.Version)
	}
	if store.data.Documents == nil {
		store.data.Documents = map[string]Document{}
	}
	if store.data.Tombstones == nil {
		store.data.Tombstones = map[string]bool{}
	}
	for name, document := range store.data.Documents {
		profile, err := Parse(document.YAML)
		if err != nil || profile.Metadata.Name != name {
			return nil, fmt.Errorf("invalid stored build profile %q: %v", name, err)
		}
		document.Profile = profile
		store.data.Documents[name] = document
	}
	return store, nil
}

func (store *Store) Get(name string) (Document, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok := store.data.Documents[name]
	if !ok {
		return Document{}, ErrNotFound
	}
	return cloneDocument(document)
}

func (store *Store) IsTombstoned(name string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.data.Tombstones[name]
}

func (store *Store) List() ([]Document, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	names := make([]string, 0, len(store.data.Documents))
	for name := range store.data.Documents {
		names = append(names, name)
	}
	sort.Strings(names)
	documents := make([]Document, 0, len(names))
	for _, name := range names {
		copy, err := cloneDocument(store.data.Documents[name])
		if err != nil {
			return nil, err
		}
		documents = append(documents, copy)
	}
	return documents, nil
}

// Import creates or edits a user-authored profile. Repeated identical imports
// are idempotent; a changed document must increment its revision.
func (store *Store) Import(profile Profile) (Document, error) {
	canonical, err := CanonicalYAML(profile)
	if err != nil {
		return Document{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	name := profile.Metadata.Name
	if current, ok := store.data.Documents[name]; ok {
		if string(current.YAML) == string(canonical) {
			return cloneDocument(current)
		}
		if profile.Metadata.Revision <= current.Profile.Metadata.Revision {
			return Document{}, ErrRevision
		}
	}
	parsed, err := Parse(canonical)
	if err != nil {
		return Document{}, err
	}
	document := Document{Profile: parsed, YAML: canonical}
	previous, hadPrevious := store.data.Documents[name]
	oldTombstone := store.data.Tombstones[name]
	store.data.Documents[name] = document
	delete(store.data.Tombstones, name)
	if err := store.save(); err != nil {
		if hadPrevious {
			store.data.Documents[name] = previous
		} else {
			delete(store.data.Documents, name)
		}
		if oldTombstone {
			store.data.Tombstones[name] = true
		}
		return Document{}, err
	}
	return cloneDocument(document)
}

// UpsertLegacy never overwrites an authored YAML profile or revives a deletion.
// When update is false, an existing migrated snapshot is left untouched.
func (store *Store) UpsertLegacy(profile Profile, configID string, config []byte, update bool) (Document, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	name := profile.Metadata.Name
	if store.data.Tombstones[name] {
		return Document{}, ErrNotFound
	}
	if current, ok := store.data.Documents[name]; ok {
		if !current.Migrated || !update {
			return cloneDocument(current)
		}
		profile.Metadata.Revision = current.Profile.Metadata.Revision
		unchanged, err := CanonicalYAML(profile)
		if err != nil {
			return Document{}, err
		}
		if string(unchanged) == string(current.YAML) && current.LegacyConfigID == configID && string(current.LegacyConfig) == string(config) {
			return cloneDocument(current)
		}
		profile.Metadata.Revision = current.Profile.Metadata.Revision + 1
	}
	canonical, err := CanonicalYAML(profile)
	if err != nil {
		return Document{}, err
	}
	parsed, err := Parse(canonical)
	if err != nil {
		return Document{}, err
	}
	document := Document{Profile: parsed, YAML: canonical, Migrated: true, LegacyConfigID: configID, LegacyConfig: append([]byte(nil), config...)}
	previous, hadPrevious := store.data.Documents[name]
	store.data.Documents[name] = document
	if err := store.save(); err != nil {
		if hadPrevious {
			store.data.Documents[name] = previous
		} else {
			delete(store.data.Documents, name)
		}
		return Document{}, err
	}
	return cloneDocument(document)
}

// Delete retains a tombstone so a stale legacy row cannot be backfilled again.
func (store *Store) Delete(name string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	old, existed := store.data.Documents[name]
	oldTombstone := store.data.Tombstones[name]
	delete(store.data.Documents, name)
	store.data.Tombstones[name] = true
	if err := store.save(); err != nil {
		if existed {
			store.data.Documents[name] = old
		}
		if !oldTombstone {
			delete(store.data.Tombstones, name)
		}
		return err
	}
	return nil
}

// Restore compensates a failed linked legacy-row deletion. A nil document
// clears the tombstone so the still-present legacy row can be backfilled.
func (store *Store) Restore(name string, document *Document) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous, hadPrevious := store.data.Documents[name]
	oldTombstone := store.data.Tombstones[name]
	if document == nil {
		delete(store.data.Documents, name)
	} else {
		copy, err := cloneDocument(*document)
		if err != nil {
			return err
		}
		store.data.Documents[name] = copy
	}
	delete(store.data.Tombstones, name)
	if err := store.save(); err != nil {
		if hadPrevious {
			store.data.Documents[name] = previous
		} else {
			delete(store.data.Documents, name)
		}
		if oldTombstone {
			store.data.Tombstones[name] = true
		}
		return err
	}
	return nil
}

func cloneDocument(document Document) (Document, error) {
	clone, err := Parse(document.YAML)
	if err != nil {
		return Document{}, fmt.Errorf("stored build profile was invalid after validation: %w", err)
	}
	document.Profile = clone
	document.YAML = append([]byte(nil), document.YAML...)
	document.LegacyConfig = append([]byte(nil), document.LegacyConfig...)
	return document, nil
}

func (store *Store) save() error {
	encoded, err := json.MarshalIndent(store.data, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(store.path)
	file, err := os.CreateTemp(directory, ".profiles-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(encoded); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), store.path); err != nil {
		return err
	}
	dir, err := os.Open(directory)
	if err == nil {
		defer dir.Close()
		_ = dir.Sync()
	}
	return nil
}
