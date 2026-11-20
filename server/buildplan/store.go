package buildplan

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"
)

const recordFile = "record.json"
const finalFile = "final.bin" // Legacy records created before numbered revisions.

var ErrArtifactNotReady = errors.New("build final artifact is not ready")

var recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,127}$`)

// Store persists immutable plans, progress, and final bytes in a private
// server directory. Each record has its own directory, so ID reservation is
// atomic even if more than one server process opens the same root.
type Store struct {
	root string
	mu   sync.Mutex
}

// OpenStore opens a server-controlled record directory. Files and newly
// created directories use owner-only permissions.
func OpenStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("empty build record root")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("build record root is not a directory: %s", root)
	}
	return &Store{root: root}, nil
}

func (s *Store) Create(plan Plan) (Record, error) {
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return Record{}, err
	}
	return s.CreateWithID(hex.EncodeToString(randomID[:]), plan)
}

// CreateWithID lets the compatibility Generate path use its existing
// ImplantBuild.ID, so Regenerate can find the frozen final artifact directly.
func (s *Store) CreateWithID(id string, plan Plan) (Record, error) {
	if err := validateRecordID(id); err != nil {
		return Record{}, err
	}
	if len(plan.Steps) == 0 {
		return Record{}, errors.New("cannot store a plan without steps")
	}
	planCopy, err := clonePlan(plan)
	if err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.recordDir(id)
	if err := os.Mkdir(path, 0o700); err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	record := Record{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		State:     BuildPending,
		Plan:      planCopy,
		Steps:     make([]StepRecord, len(planCopy.Steps)),
	}
	for i, step := range planCopy.Steps {
		record.Steps[i] = StepRecord{ID: step.ID, State: StepPending}
	}
	if err := s.writeRecord(record); err != nil {
		_ = os.Remove(path)
		return Record{}, err
	}
	return cloneRecord(record)
}

type StepUpdate struct {
	State             StepState
	Progress          int
	Failure           string
	ToolchainIdentity string
	Artifact          *ArtifactInfo
	Selection         *StepSelection
}

// UpdateStep records execution progress without changing the frozen plan.
func (s *Store) UpdateStep(id string, index int, update StepUpdate) (Record, error) {
	if err := validateRecordID(id); err != nil {
		return Record{}, err
	}
	if update.Progress < 0 || update.Progress > 100 {
		return Record{}, errors.New("step progress must be between 0 and 100")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return Record{}, err
	}
	if record.State == BuildComplete || record.State == BuildFailed || record.State == BuildCancelled {
		return Record{}, fmt.Errorf("build %s is terminal", id)
	}
	if index < 0 || index >= len(record.Steps) {
		return Record{}, fmt.Errorf("step index %d out of range", index)
	}
	step := &record.Steps[index]
	if err := checkTransition(step.State, update.State); err != nil {
		return Record{}, err
	}
	if update.State == StepComplete && update.Progress != 100 {
		return Record{}, errors.New("completed step must have 100 percent progress")
	}
	if update.State == StepFailed && update.Failure == "" {
		return Record{}, errors.New("failed step requires failure detail")
	}
	if update.Artifact != nil {
		if err := validateArtifactInfo(*update.Artifact, record.Plan.Steps[index].Output); err != nil {
			return Record{}, err
		}
	}
	now := time.Now().UTC()
	if update.State == StepRunning && step.StartedAt == nil {
		step.StartedAt = &now
	}
	if update.State == StepComplete || update.State == StepFailed || update.State == StepCancelled {
		step.FinishedAt = &now
	}
	step.State = update.State
	step.Progress = update.Progress
	step.Failure = update.Failure
	if update.ToolchainIdentity != "" {
		step.ToolchainIdentity = update.ToolchainIdentity
	}
	if update.Artifact != nil {
		artifact := *update.Artifact
		step.Artifact = &artifact
	}
	if update.Selection != nil {
		selection := *update.Selection
		step.Selection = &selection
	}
	if update.State == StepFailed {
		record.State = BuildFailed
		record.Failure = update.Failure
	} else if update.State == StepCancelled {
		record.State = BuildCancelled
	} else {
		record.State = BuildRunning
	}
	record.UpdatedAt = now
	if err := s.writeRecord(record); err != nil {
		return Record{}, err
	}
	return cloneRecord(record)
}

// Complete writes final bytes first, then atomically publishes the completed
// record. Regenerate reads this byte stream, never a recompiled plan.
func (s *Store) Complete(id string, artifact ArtifactInfo, finalBytes []byte) (Record, error) {
	if err := validateRecordID(id); err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return Record{}, err
	}
	if record.State == BuildComplete || record.State == BuildFailed || record.State == BuildCancelled {
		return Record{}, fmt.Errorf("build %s is terminal", id)
	}
	for _, step := range record.Steps {
		if step.State != StepComplete {
			return Record{}, fmt.Errorf("cannot complete build with %s step %s", step.State, step.ID)
		}
	}
	if err := validateArtifactInfo(artifact, record.Plan.FinalArtifact); err != nil {
		return Record{}, err
	}
	if err := validateFinalBytes(artifact, finalBytes); err != nil {
		return Record{}, err
	}
	if err := writeAtomic(s.finalPath(id, 1), finalBytes); err != nil {
		return Record{}, err
	}
	artifactCopy := artifact
	now := time.Now().UTC()
	record.Final = &artifactCopy
	record.FinalRevision = 1
	record.FinalHistory = append(record.FinalHistory, FinalRevision{Number: 1, UpdatedAt: now, Artifact: artifactCopy})
	record.State = BuildComplete
	record.UpdatedAt = now
	if err := s.writeRecord(record); err != nil {
		return Record{}, err
	}
	return cloneRecord(record)
}

// ReplaceFinal publishes a new revision of completed bytes after a legacy
// post-build mutation such as metadata spoofing. Earlier revisions remain on
// disk, and the original request and resolved plan are unchanged.
func (s *Store) ReplaceFinal(id string, artifact ArtifactInfo, finalBytes []byte) (Record, error) {
	if err := validateRecordID(id); err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return Record{}, err
	}
	if record.State != BuildComplete || record.Final == nil {
		return Record{}, ErrArtifactNotReady
	}
	if err := validateArtifactInfo(artifact, record.Plan.FinalArtifact); err != nil {
		return Record{}, err
	}
	if err := validateFinalBytes(artifact, finalBytes); err != nil {
		return Record{}, err
	}
	nextRevision := record.FinalRevision + 1
	if nextRevision == 1 { // A record created before numbered revisions.
		nextRevision = 2
	}
	if err := writeAtomic(s.finalPath(id, nextRevision), finalBytes); err != nil {
		return Record{}, err
	}
	artifactCopy := artifact
	now := time.Now().UTC()
	record.Final = &artifactCopy
	record.FinalRevision = nextRevision
	record.FinalHistory = append(record.FinalHistory,
		FinalRevision{Number: nextRevision, UpdatedAt: now, Artifact: artifactCopy})
	record.UpdatedAt = now
	if err := s.writeRecord(record); err != nil {
		return Record{}, err
	}
	return cloneRecord(record)
}

func (s *Store) Get(id string) (Record, error) {
	if err := validateRecordID(id); err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return Record{}, err
	}
	return cloneRecord(record)
}

// FinalBytes verifies the persisted bytes against the frozen final metadata.
func (s *Store) FinalBytes(id string) ([]byte, ArtifactInfo, error) {
	if err := validateRecordID(id); err != nil {
		return nil, ArtifactInfo{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return nil, ArtifactInfo{}, err
	}
	if record.State != BuildComplete || record.Final == nil {
		return nil, ArtifactInfo{}, ErrArtifactNotReady
	}
	data, err := os.ReadFile(s.finalPath(id, record.FinalRevision))
	if err != nil {
		return nil, ArtifactInfo{}, err
	}
	hash := sha256.Sum256(data)
	if int64(len(data)) != record.Final.Size || hex.EncodeToString(hash[:]) != record.Final.SHA256 {
		return nil, ArtifactInfo{}, fmt.Errorf("stored final artifact checksum mismatch for %s", id)
	}
	return data, *record.Final, nil
}

func (s *Store) recordDir(id string) string { return filepath.Join(s.root, id) }

func (s *Store) finalPath(id string, revision int) string {
	if revision == 0 {
		return filepath.Join(s.recordDir(id), finalFile)
	}
	return filepath.Join(s.recordDir(id), fmt.Sprintf("final-%d.bin", revision))
}

func (s *Store) readRecord(id string) (Record, error) {
	data, err := os.ReadFile(filepath.Join(s.recordDir(id), recordFile))
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, err
	}
	if record.ID != id {
		return Record{}, fmt.Errorf("record ID mismatch for %s", id)
	}
	return record, nil
}

func (s *Store) writeRecord(record Record) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.recordDir(record.ID), recordFile), data)
}

func writeAtomic(path string, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func validateRecordID(id string) error {
	if !recordIDPattern.MatchString(id) {
		return fmt.Errorf("invalid build record ID %q", id)
	}
	return nil
}

func validateArtifactInfo(info ArtifactInfo, expected Artifact) error {
	if !artifactEqual(info.Type, expected) {
		return errors.New("artifact type does not match frozen plan")
	}
	if info.Name == "" || filepath.Base(info.Name) != info.Name {
		return errors.New("artifact name must be a basename")
	}
	if info.Size < 0 {
		return errors.New("artifact size cannot be negative")
	}
	if len(info.SHA256) != 64 {
		return errors.New("artifact SHA-256 must be lowercase hex")
	}
	decoded, err := hex.DecodeString(info.SHA256)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != info.SHA256 {
		return errors.New("artifact SHA-256 must be lowercase hex")
	}
	return nil
}

func validateFinalBytes(artifact ArtifactInfo, finalBytes []byte) error {
	computed := sha256.Sum256(finalBytes)
	if artifact.Size != int64(len(finalBytes)) || artifact.SHA256 != hex.EncodeToString(computed[:]) {
		return errors.New("final artifact size or SHA-256 does not match bytes")
	}
	return nil
}

func artifactEqual(a, b Artifact) bool {
	return a.Kind == b.Kind && a.Container == b.Container &&
		a.Target == b.Target && a.GoBuildMode == b.GoBuildMode &&
		slices.Equal(a.EntrySymbols, b.EntrySymbols)
}

func checkTransition(from, to StepState) error {
	switch from {
	case StepPending:
		if to == StepRunning || to == StepFailed || to == StepCancelled {
			return nil
		}
	case StepRunning:
		if to == StepRunning || to == StepComplete || to == StepFailed || to == StepCancelled {
			return nil
		}
	}
	return fmt.Errorf("invalid step transition %s -> %s", from, to)
}

func clonePlan(plan Plan) (Plan, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return Plan{}, err
	}
	var result Plan
	err = json.Unmarshal(data, &result)
	return result, err
}

func cloneRecord(record Record) (Record, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return Record{}, err
	}
	var result Record
	err = json.Unmarshal(data, &result)
	return result, err
}
