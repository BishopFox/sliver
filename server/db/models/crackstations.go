package models

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"math"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"gorm.io/gorm"
)

// Crackstation - History of crackstation jobs
type Crackstation struct {
	// ID = crackstation name
	ID                      UUID      `gorm:"primaryKey;type:uuid;"`
	CreatedAt               time.Time `gorm:"->;<-:create;"`
	OperatorName            string
	HashcatVersion          string `gorm:"size:128"`
	BenchmarkHashcatVersion string `gorm:"size:128"`
	BenchmarkSchemaVersion  uint32
	Tasks                   []CrackTask
	Benchmarks              []Benchmark
}

// BeforeCreate - GORM hook
func (c *Crackstation) BeforeCreate(tx *gorm.DB) (err error) {
	c.CreatedAt = time.Now()
	return nil
}

// Benchmark - Performance information about the crackstation
type Benchmark struct {
	ID             UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	CrackstationID UUID      `gorm:"type:uuid;"`
	HashType       int32
	PerSecondRate  uint64
}

// BeforeCreate - GORM hook
func (b *Benchmark) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID = NewUUID()
	b.CreatedAt = time.Now()
	return nil
}

// CrackFile - Performance information about the crackstation
type CrackFile struct {
	ID               UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt        time.Time `gorm:"->;<-:create;"`
	LastModified     time.Time
	Name             string
	UncompressedSize int64
	CompressedSize   int64
	Sha2_256         string
	Type             int32
	IsCompressed     bool
	IsComplete       bool

	Chunks []CrackFileChunk
}

// BeforeCreate - GORM hook
func (c *CrackFile) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = NewUUID()
	c.CreatedAt = time.Now()
	return nil
}

func (c *CrackFile) MaxN(chunkSize int64) uint32 {
	if chunkSize < 1 {
		panic("invalid chunk size")
	}
	return uint32(math.Ceil(float64(c.UncompressedSize) / float64(chunkSize)))
}

func (c *CrackFile) ToProtobuf() *clientpb.CrackFile {
	chunks := []*clientpb.CrackFileChunk{}
	for _, chunk := range c.Chunks {
		chunks = append(chunks, chunk.ToProtobuf())
	}
	return &clientpb.CrackFile{
		ID:               c.ID.String(),
		CreatedAt:        c.CreatedAt.Unix(),
		LastModified:     c.LastModified.Unix(),
		Name:             c.Name,
		UncompressedSize: c.UncompressedSize,
		CompressedSize:   c.CompressedSize,
		Sha2_256:         c.Sha2_256,
		Type:             clientpb.CrackFileType(c.Type),
		IsCompressed:     c.IsCompressed,
		Chunks:           chunks,
	}
}

// CrackFileChunk - Performance information about the crackstation
type CrackFileChunk struct {
	ID          UUID   `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackFileID UUID   `gorm:"type:uuid;uniqueIndex:idx_crack_file_chunk_n"`
	N           uint32 `gorm:"uniqueIndex:idx_crack_file_chunk_n"`
}

// BeforeCreate - GORM hook
func (c *CrackFileChunk) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == NilUUID() {
		c.ID = NewUUID()
	}
	return nil
}

func (c *CrackFileChunk) ToProtobuf() *clientpb.CrackFileChunk {
	return &clientpb.CrackFileChunk{
		ID: c.ID.String(),
		N:  c.N,
	}
}

// CrackJob - A crack job is a collection of one or more crack tasks, the
// crack job contains the parent command, whose keyspace may get broken
// up into multiple crack tasks and distributed to multiple crackstations
type CrackJob struct {
	ID           UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt    time.Time `gorm:"->;<-:create;"`
	UpdatedAt    time.Time
	CompletedAt  time.Time
	Err          string
	ResultFileID string
	Keyspace     string
	// RecoveredBytes bounds the cumulative canonical recovery payload accepted
	// across streaming result batches. It is internal queue accounting and is
	// intentionally not exposed in the operator protobuf.
	RecoveredBytes uint64
	HashcatVersion string `gorm:"size:128"`
	Tasks          []CrackTask
	Results        []CrackResult
	ResultCount    uint64 `gorm:"-"`

	Command CrackCommand // Parent command
}

func (c *CrackJob) Status() clientpb.CrackJobStatus {
	if c.CompletedAt.IsZero() {
		return clientpb.CrackJobStatus_IN_PROGRESS
	}
	if c.Err != "" {
		return clientpb.CrackJobStatus_FAILED
	}
	for _, task := range c.Tasks {
		if clientpb.CrackTaskState(task.State) == clientpb.CrackTaskState_CRACK_TASK_FAILED {
			return clientpb.CrackJobStatus_FAILED
		}
	}
	return clientpb.CrackJobStatus_COMPLETED
}

// BeforeCreate - GORM hook
func (c *CrackJob) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == NilUUID() {
		c.ID = NewUUID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	return nil
}

func (c *CrackJob) ToProtobuf() *clientpb.CrackJob {
	job := &clientpb.CrackJob{
		ID:           c.ID.String(),
		CreatedAt:    c.CreatedAt.UTC().Format(time.RFC3339),
		Status:       c.Status(),
		Err:          c.Err,
		Command:      c.Command.ToProtobuf(),
		ResultFileID: c.ResultFileID,
		Keyspace:     c.Keyspace,
		ResultCount:  c.ResultCount,
	}
	if len(c.Results) != 0 {
		job.ResultCount = uint64(len(c.Results))
	}
	if !c.UpdatedAt.IsZero() {
		job.UpdatedAt = c.UpdatedAt.Unix()
	}
	for index := range c.Tasks {
		job.Tasks = append(job.Tasks, c.Tasks[index].ToProtobuf())
	}
	for index := range c.Results {
		job.Results = append(job.Results, c.Results[index].ToProtobuf())
	}
	if !c.CompletedAt.IsZero() {
		job.CompletedAt = c.CompletedAt.UTC().Format(time.RFC3339)
	}
	return job
}

func (CrackJob) FromProtobuf(c *clientpb.CrackJob) *CrackJob {
	job := &CrackJob{}
	if c == nil {
		return job
	}
	jobID := ParseUUIDOrNil(c.ID)
	if jobID != NilUUID() {
		job.ID = jobID
	}
	if c.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, c.CreatedAt); err == nil {
			job.CreatedAt = createdAt
		}
	}
	if c.CompletedAt != "" {
		if completedAt, err := time.Parse(time.RFC3339, c.CompletedAt); err == nil {
			job.CompletedAt = completedAt
		}
	}
	job.Err = c.Err
	job.ResultFileID = c.ResultFileID
	job.Keyspace = c.Keyspace
	if c.UpdatedAt != 0 {
		job.UpdatedAt = time.Unix(c.UpdatedAt, 0)
	}
	if c.Command != nil {
		job.Command = *CrackCommand{}.FromProtobuf(c.Command)
		job.Command.CrackJobID = job.ID
	}
	return job
}

// CrackTask - An individual chunk of a job sent to a specific crackstation
type CrackTask struct {
	ID               UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackJobID       UUID      `gorm:"type:uuid;"`
	CrackstationID   UUID      `gorm:"type:uuid;"`
	CreatedAt        time.Time `gorm:"->;<-:create;"`
	UpdatedAt        time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
	LeaseExpiresAt   time.Time `gorm:"index:idx_crack_task_state_lease"`
	LastHeartbeatAt  time.Time
	Kind             int32
	State            int32 `gorm:"index:idx_crack_task_state_lease"`
	Attempt          uint32
	LeaseToken       string
	Keyspace         string
	LatestStatusJSON []byte
	RecoveredJSON    []byte
	ShardSkip        uint64
	ShardLimit       uint64
	Err              string
	Stdout           []byte
	Stderr           []byte
	ExitCode         int32
	StdoutTruncated  bool
	StderrTruncated  bool
	StdoutTotalBytes uint64
	StderrTotalBytes uint64

	Command CrackCommand
}

func (c *CrackTask) ToProtobuf() *clientpb.CrackTask {
	task := &clientpb.CrackTask{
		ID:               c.ID.String(),
		Err:              c.Err,
		Stdout:           c.Stdout,
		Stderr:           c.Stderr,
		ExitCode:         c.ExitCode,
		StdoutTruncated:  c.StdoutTruncated,
		StderrTruncated:  c.StderrTruncated,
		StdoutTotalBytes: c.StdoutTotalBytes,
		StderrTotalBytes: c.StderrTotalBytes,
		Command:          c.Command.ToProtobuf(),
		Kind:             clientpb.CrackTaskKind(c.Kind),
		State:            clientpb.CrackTaskState(c.State),
		Attempt:          c.Attempt,
		LeaseToken:       c.LeaseToken,
		Keyspace:         c.Keyspace,
		LatestStatusJSON: append([]byte(nil), c.LatestStatusJSON...),
		RecoveredJSON:    append([]byte(nil), c.RecoveredJSON...),
		ShardSkip:        c.ShardSkip,
		ShardLimit:       c.ShardLimit,
	}
	if c.CrackstationID != NilUUID() {
		task.HostUUID = c.CrackstationID.String()
	}
	if c.CrackJobID != NilUUID() {
		task.CrackJobID = c.CrackJobID.String()
	}
	if !c.UpdatedAt.IsZero() {
		task.UpdatedAt = c.UpdatedAt.Unix()
	}
	if !c.LeaseExpiresAt.IsZero() {
		task.LeaseExpiresAt = c.LeaseExpiresAt.Unix()
	}
	if !c.LastHeartbeatAt.IsZero() {
		task.LastHeartbeatAt = c.LastHeartbeatAt.Unix()
	}
	if !c.CreatedAt.IsZero() {
		task.CreatedAt = c.CreatedAt.Unix()
	}
	if !c.StartedAt.IsZero() {
		task.StartedAt = c.StartedAt.Unix()
	}
	if !c.CompletedAt.IsZero() {
		task.CompletedAt = c.CompletedAt.Unix()
	}
	return task
}

func (CrackTask) FromProtobuf(c *clientpb.CrackTask) *CrackTask {
	task := &CrackTask{}
	if c == nil {
		return task
	}
	task.ID = ParseUUIDOrNil(c.ID)
	task.CrackJobID = ParseUUIDOrNil(c.CrackJobID)
	task.CrackstationID = ParseUUIDOrNil(c.HostUUID)
	if c.CreatedAt != 0 {
		task.CreatedAt = time.Unix(c.CreatedAt, 0)
	}
	if c.StartedAt != 0 {
		task.StartedAt = time.Unix(c.StartedAt, 0)
	}
	if c.CompletedAt != 0 {
		task.CompletedAt = time.Unix(c.CompletedAt, 0)
	}
	task.Err = c.Err
	task.Stdout = c.Stdout
	task.Stderr = c.Stderr
	task.ExitCode = c.ExitCode
	task.StdoutTruncated = c.StdoutTruncated
	task.StderrTruncated = c.StderrTruncated
	task.StdoutTotalBytes = c.StdoutTotalBytes
	task.StderrTotalBytes = c.StderrTotalBytes
	task.Kind = int32(c.Kind)
	task.State = int32(c.State)
	task.Attempt = c.Attempt
	task.LeaseToken = c.LeaseToken
	if c.LeaseExpiresAt != 0 {
		task.LeaseExpiresAt = time.Unix(c.LeaseExpiresAt, 0)
	}
	if c.UpdatedAt != 0 {
		task.UpdatedAt = time.Unix(c.UpdatedAt, 0)
	}
	if c.LastHeartbeatAt != 0 {
		task.LastHeartbeatAt = time.Unix(c.LastHeartbeatAt, 0)
	}
	task.Keyspace = c.Keyspace
	task.LatestStatusJSON = append([]byte(nil), c.LatestStatusJSON...)
	task.RecoveredJSON = append([]byte(nil), c.RecoveredJSON...)
	task.ShardSkip = c.ShardSkip
	task.ShardLimit = c.ShardLimit
	if c.Command != nil {
		task.Command = *CrackCommand{}.FromProtobuf(c.Command)
	}
	return task
}

// BeforeCreate - GORM hook
func (c *CrackTask) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == NilUUID() {
		c.ID = NewUUID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	return nil
}

// CrackResult is one recovered hash/plaintext pair. Fingerprint is a stable,
// server-generated idempotency key and is intentionally not exposed over RPC.
type CrackResult struct {
	ID           UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt    time.Time `gorm:"->;<-:create;"`
	CrackJobID   UUID      `gorm:"type:uuid;index"`
	CrackTaskID  UUID      `gorm:"type:uuid;index"`
	CredentialID UUID      `gorm:"type:uuid;index"`
	Hash         string
	Plaintext    []byte
	Fingerprint  string `gorm:"uniqueIndex;size:64"`
}

// BeforeCreate initializes server-owned fields before a crack result is persisted.
func (c *CrackResult) BeforeCreate(_ *gorm.DB) error {
	if c.ID == NilUUID() {
		c.ID = NewUUID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	return nil
}

// ToProtobuf converts a persisted crack result to its RPC representation.
func (c *CrackResult) ToProtobuf() *clientpb.CrackResult {
	result := &clientpb.CrackResult{
		ID:          c.ID.String(),
		CrackJobID:  c.CrackJobID.String(),
		CrackTaskID: c.CrackTaskID.String(),
		Hash:        c.Hash,
		Plaintext:   append([]byte(nil), c.Plaintext...),
		CreatedAt:   c.CreatedAt.Unix(),
	}
	if c.CredentialID != NilUUID() {
		result.CredentialID = c.CredentialID.String()
	}
	return result
}

// CrackJobCredential records the exact credential rows selected for a job.
type CrackJobCredential struct {
	ID           UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackJobID   UUID `gorm:"type:uuid;uniqueIndex:idx_crack_job_credential"`
	CredentialID UUID `gorm:"type:uuid;uniqueIndex:idx_crack_job_credential"`
}

// BeforeCreate initializes server-owned fields before a job credential is persisted.
func (c *CrackJobCredential) BeforeCreate(_ *gorm.DB) error {
	if c.ID == NilUUID() {
		c.ID = NewUUID()
	}
	return nil
}

type CrackCommand struct {
	ID          UUID      `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	CrackTaskID UUID      `gorm:"type:uuid;"`
	CrackJobID  UUID      `gorm:"type:uuid;"`

	// FLAGS
	AttackMode             int32
	HashType               int32
	Hashes                 []string `gorm:"type:text;serializer:legacyjsonslice"`
	Quiet                  bool
	HexCharset             bool
	HexSalt                bool
	HexWordlist            bool
	Force                  bool
	DeprecatedCheckDisable bool
	Status                 bool
	StatusJSON             bool
	StatusTimer            uint32
	StdinTimeoutAbort      uint32
	MachineReadable        bool
	KeepGuessing           bool
	SelfTestDisable        bool
	Loopback               bool
	MarkovHcstat2          []byte
	MarkovDisable          bool
	MarkovClassic          bool
	MarkovInverse          bool
	MarkovThreshold        uint32
	Runtime                uint32
	Session                string
	Restore                bool
	RestoreDisable         bool
	RestoreFile            []byte
	Outfile                string
	OutfileFormat          []int32 `gorm:"type:text;serializer:legacyjsonslice"`
	OutfileAutohexDisable  bool
	OutfileCheckTimer      uint32
	WordlistAutohexDisable bool
	Separator              string
	Stdout                 bool
	Show                   bool
	Left                   bool
	Username               bool
	Remove                 bool
	RemoveTimer            uint32
	PotfileDisable         bool
	Potfile                []byte
	EncodingFrom           int32
	EncodingTo             int32
	DebugMode              uint32
	DebugFile              string
	InductionDir           string
	OutfileCheckDir        string
	LogfileDisable         bool
	HccapxMessagePair      uint32
	NonceErrorCorrections  uint32
	KeyboardLayoutMapping  []byte
	TruecryptKeyfiles      string
	VeracryptKeyfiles      string
	VeracryptPimStart      *uint32
	VeracryptPimStop       *uint32
	Benchmark              bool
	BenchmarkAll           bool
	SpeedOnly              bool
	ProgressOnly           bool
	SegmentSize            uint32
	BitmapMin              uint32
	BitmapMax              uint32
	CPUAffinity            []uint32 `gorm:"type:text;serializer:legacyjsonslice"`
	HookThreads            uint32
	HashInfo               bool
	// --example-hashes (66)
	BackendIgnoreCUDA         bool
	BackendIgnoreHip          bool
	BackendIgnoreMetal        bool
	BackendIgnoreOpenCL       bool
	BackendInfo               bool
	BackendDevices            []uint32 `gorm:"type:text;serializer:legacyjsonslice"`
	OpenCLDeviceTypes         []uint32 `gorm:"type:text;serializer:legacyjsonslice"`
	OptimizedKernelEnable     bool
	MultiplyAccelDisabled     bool
	WorkloadProfile           int32
	KernelAccel               uint32
	KernelLoops               uint32
	KernelThreads             uint32
	BackendVectorWidth        uint32
	SpinDamp                  uint32
	HwmonDisable              bool
	HwmonTempAbort            uint32
	ScryptTMTO                uint32
	Skip                      uint64
	Limit                     uint64
	Keyspace                  bool
	RuleLeft                  string
	RuleRight                 string
	RulesFile                 []byte
	GenerateRules             uint32
	GenerateRulesFunMin       uint32
	GenerateRulesFunMax       uint32
	GenerateRulesFuncSel      string
	GenerateRulesSeed         int32
	CustomCharset1            string
	CustomCharset2            string
	CustomCharset3            string
	CustomCharset4            string
	Identify                  string
	Increment                 bool
	IncrementMin              uint32
	IncrementMax              uint32
	SlowCandidates            bool
	BrainServer               bool
	BrainServerTimer          uint32
	BrainClient               bool
	BrainClientFeatures       string
	BrainHost                 string
	BrainPort                 uint32
	BrainPassword             string
	BrainSession              string
	BrainSessionWhitelist     string
	PipelineStats             bool
	TaskTimeBreakdown         bool
	MetalCompilerRuntime      uint32
	RestorePosition           bool
	OutfileJSON               bool
	DynamicX                  bool
	SeekDBPath                string
	BenchmarkMin              uint32
	BenchmarkMax              *uint32
	BridgeParameter1          string
	BridgeParameter2          string
	BridgeParameter3          string
	BridgeParameter4          string
	BackendDevicesVirtMulti   uint32
	BackendDevicesVirtHost    uint32
	TotalCandidates           bool
	Lookup                    string
	CustomCharset5            string
	CustomCharset6            string
	CustomCharset7            string
	CustomCharset8            string
	IncrementInverse          bool
	BypassDelay               *uint32
	BypassThreshold           *uint32
	BrainFeed                 bool
	ColorCracked              bool
	HashCopy                  bool
	EncryptWithPubkey         string
	IdentifyMode              bool
	PositionalArguments       []string `gorm:"serializer:json"`
	EncodingFromName          string
	EncodingToName            string
	HashInfoLevel             uint32
	BackendInfoLevel          uint32
	HccapxMessagePairV7       *uint32
	NonceErrorCorrectionsV7   *uint32
	ScryptTMTOV7              *uint32
	GenerateRulesSeedV7       *uint32
	BrainClientFeaturesV7     uint32
	BrainSessionV7            *uint32
	BrainSessionWhitelistV7   []uint32 `gorm:"serializer:json"`
	Stdin                     []byte
	AdviceDisable             bool
	HashMode                  *uint32
	RestoreShowCommand        bool
	BrainServerTimerV7        *uint32
	StatusTimerV7             *uint32
	StdinTimeoutAbortV7       *uint32
	OutfileCheckTimerV7       *uint32
	BitmapMinV7               *uint32
	BitmapMaxV7               *uint32
	HwmonTempAbortV7          *uint32
	RulesFilesV7              [][]byte `gorm:"serializer:json"`
	BrainPasswordV7           *string
	GenerateRulesFuncMinV7    *uint32
	GenerateRulesFuncMaxV7    *uint32
	CredentialIDs             []string `gorm:"serializer:json"`
	CredentialCollection      string
	IncludeCrackedCredentials bool
}

// BeforeCreate - GORM hook
func (c *CrackCommand) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = NewUUID()
	c.CreatedAt = time.Now()
	return nil
}

func (c *CrackCommand) ToProtobuf() *clientpb.CrackCommand {
	cmd := &clientpb.CrackCommand{}
	cmd.AttackMode = clientpb.CrackAttackMode(c.AttackMode)
	cmd.HashType = clientpb.HashType(c.HashType)
	cmd.Hashes = c.Hashes
	// --version
	// --help
	cmd.Quiet = c.Quiet
	cmd.HexCharset = c.HexCharset
	cmd.HexSalt = c.HexSalt
	cmd.HexWordlist = c.HexWordlist
	cmd.Force = c.Force
	cmd.DeprecatedCheckDisable = c.DeprecatedCheckDisable
	cmd.Status = c.Status
	cmd.StatusJSON = c.StatusJSON
	cmd.StatusTimer = c.StatusTimer
	cmd.StdinTimeoutAbort = c.StdinTimeoutAbort
	cmd.MachineReadable = c.MachineReadable
	cmd.KeepGuessing = c.KeepGuessing
	cmd.SelfTestDisable = c.SelfTestDisable
	cmd.Loopback = c.Loopback
	cmd.MarkovHcstat2 = c.MarkovHcstat2
	cmd.MarkovDisable = c.MarkovDisable
	cmd.MarkovClassic = c.MarkovClassic
	cmd.MarkovInverse = c.MarkovInverse
	cmd.MarkovThreshold = c.MarkovThreshold
	cmd.Runtime = c.Runtime
	cmd.Session = c.Session
	cmd.Restore = c.Restore
	cmd.RestoreDisable = c.RestoreDisable
	cmd.RestoreFile = c.RestoreFile
	cmd.Outfile = c.Outfile
	cmd.OutfileFormat = []clientpb.CrackOutfileFormat{}
	for _, f := range c.OutfileFormat {
		cmd.OutfileFormat = append(cmd.OutfileFormat, clientpb.CrackOutfileFormat(f))
	}
	cmd.OutfileAutohexDisable = c.OutfileAutohexDisable
	cmd.OutfileCheckTimer = c.OutfileCheckTimer
	cmd.WordlistAutohexDisable = c.WordlistAutohexDisable
	cmd.Separator = c.Separator
	cmd.Stdout = c.Stdout
	cmd.Show = c.Show
	cmd.Left = c.Left
	cmd.Username = c.Username
	cmd.Remove = c.Remove
	cmd.RemoveTimer = c.RemoveTimer
	cmd.PotfileDisable = c.PotfileDisable
	cmd.Potfile = c.Potfile
	cmd.EncodingFrom = clientpb.CrackEncoding(c.EncodingFrom)
	cmd.EncodingTo = clientpb.CrackEncoding(c.EncodingTo)
	cmd.DebugMode = c.DebugMode
	cmd.DebugFile = c.DebugFile
	cmd.InductionDir = c.InductionDir
	cmd.OutfileCheckDir = c.OutfileCheckDir
	cmd.LogfileDisable = c.LogfileDisable
	cmd.HccapxMessagePair = c.HccapxMessagePair
	cmd.NonceErrorCorrections = c.NonceErrorCorrections
	cmd.KeyboardLayoutMapping = c.KeyboardLayoutMapping
	cmd.TruecryptKeyfiles = c.TruecryptKeyfiles
	cmd.VeracryptKeyfiles = c.VeracryptKeyfiles
	cmd.VeracryptPimStart = c.VeracryptPimStart
	cmd.VeracryptPimStop = c.VeracryptPimStop
	cmd.Benchmark = c.Benchmark
	cmd.BenchmarkAll = c.BenchmarkAll
	cmd.SpeedOnly = c.SpeedOnly
	cmd.ProgressOnly = c.ProgressOnly
	cmd.SegmentSize = c.SegmentSize
	cmd.BitmapMin = c.BitmapMin
	cmd.BitmapMax = c.BitmapMax
	cmd.CPUAffinity = c.CPUAffinity
	cmd.HookThreads = c.HookThreads
	cmd.HashInfo = c.HashInfo
	// --example-hashes (66)
	cmd.BackendIgnoreCUDA = c.BackendIgnoreCUDA
	cmd.BackendIgnoreHip = c.BackendIgnoreHip
	cmd.BackendIgnoreMetal = c.BackendIgnoreMetal
	cmd.BackendIgnoreOpenCL = c.BackendIgnoreOpenCL
	cmd.BackendInfo = c.BackendInfo
	cmd.BackendDevices = c.BackendDevices
	cmd.OpenCLDeviceTypes = c.OpenCLDeviceTypes
	cmd.OptimizedKernelEnable = c.OptimizedKernelEnable
	cmd.MultiplyAccelDisabled = c.MultiplyAccelDisabled
	cmd.WorkloadProfile = clientpb.CrackWorkloadProfile(c.WorkloadProfile)
	cmd.KernelAccel = c.KernelAccel
	cmd.KernelLoops = c.KernelLoops
	cmd.KernelThreads = c.KernelThreads
	cmd.BackendVectorWidth = c.BackendVectorWidth
	cmd.SpinDamp = c.SpinDamp
	cmd.HwmonDisable = c.HwmonDisable
	cmd.HwmonTempAbort = c.HwmonTempAbort
	cmd.ScryptTMTO = c.ScryptTMTO
	cmd.Skip = c.Skip
	cmd.Limit = c.Limit
	cmd.Keyspace = c.Keyspace
	cmd.RuleLeft = c.RuleLeft
	cmd.RuleRight = c.RuleRight
	cmd.RulesFile = c.RulesFile
	cmd.GenerateRules = c.GenerateRules
	cmd.GenerateRulesFunMin = c.GenerateRulesFunMin
	cmd.GenerateRulesFunMax = c.GenerateRulesFunMax
	cmd.GenerateRulesFuncSel = c.GenerateRulesFuncSel
	cmd.GenerateRulesSeed = c.GenerateRulesSeed
	cmd.CustomCharset1 = c.CustomCharset1
	cmd.CustomCharset2 = c.CustomCharset2
	cmd.CustomCharset3 = c.CustomCharset3
	cmd.CustomCharset4 = c.CustomCharset4
	cmd.Identify = c.Identify
	cmd.Increment = c.Increment
	cmd.IncrementMin = c.IncrementMin
	cmd.IncrementMax = c.IncrementMax
	cmd.SlowCandidates = c.SlowCandidates
	cmd.BrainServer = c.BrainServer
	cmd.BrainServerTimer = c.BrainServerTimer
	cmd.BrainClient = c.BrainClient
	cmd.BrainClientFeatures = c.BrainClientFeatures
	cmd.BrainHost = c.BrainHost
	cmd.BrainPort = c.BrainPort
	cmd.BrainPassword = c.BrainPassword
	cmd.BrainSession = c.BrainSession
	cmd.BrainSessionWhitelist = c.BrainSessionWhitelist
	cmd.PipelineStats = c.PipelineStats
	cmd.TaskTimeBreakdown = c.TaskTimeBreakdown
	cmd.MetalCompilerRuntime = c.MetalCompilerRuntime
	cmd.RestorePosition = c.RestorePosition
	cmd.OutfileJSON = c.OutfileJSON
	cmd.DynamicX = c.DynamicX
	cmd.SeekDBPath = c.SeekDBPath
	cmd.BenchmarkMin = c.BenchmarkMin
	cmd.BenchmarkMax = c.BenchmarkMax
	cmd.BridgeParameter1 = c.BridgeParameter1
	cmd.BridgeParameter2 = c.BridgeParameter2
	cmd.BridgeParameter3 = c.BridgeParameter3
	cmd.BridgeParameter4 = c.BridgeParameter4
	cmd.BackendDevicesVirtMulti = c.BackendDevicesVirtMulti
	cmd.BackendDevicesVirtHost = c.BackendDevicesVirtHost
	cmd.TotalCandidates = c.TotalCandidates
	cmd.Lookup = c.Lookup
	cmd.CustomCharset5 = c.CustomCharset5
	cmd.CustomCharset6 = c.CustomCharset6
	cmd.CustomCharset7 = c.CustomCharset7
	cmd.CustomCharset8 = c.CustomCharset8
	cmd.IncrementInverse = c.IncrementInverse
	cmd.BypassDelay = c.BypassDelay
	cmd.BypassThreshold = c.BypassThreshold
	cmd.BrainFeed = c.BrainFeed
	cmd.ColorCracked = c.ColorCracked
	cmd.HashCopy = c.HashCopy
	cmd.EncryptWithPubkey = c.EncryptWithPubkey
	cmd.IdentifyMode = c.IdentifyMode
	cmd.PositionalArguments = c.PositionalArguments
	cmd.EncodingFromName = c.EncodingFromName
	cmd.EncodingToName = c.EncodingToName
	cmd.HashInfoLevel = c.HashInfoLevel
	cmd.BackendInfoLevel = c.BackendInfoLevel
	cmd.HccapxMessagePairV7 = c.HccapxMessagePairV7
	cmd.NonceErrorCorrectionsV7 = c.NonceErrorCorrectionsV7
	cmd.ScryptTMTOV7 = c.ScryptTMTOV7
	cmd.GenerateRulesSeedV7 = c.GenerateRulesSeedV7
	cmd.BrainClientFeaturesV7 = c.BrainClientFeaturesV7
	cmd.BrainSessionV7 = c.BrainSessionV7
	cmd.BrainSessionWhitelistV7 = c.BrainSessionWhitelistV7
	cmd.Stdin = c.Stdin
	cmd.AdviceDisable = c.AdviceDisable
	cmd.HashMode = c.HashMode
	cmd.RestoreShowCommand = c.RestoreShowCommand
	cmd.BrainServerTimerV7 = c.BrainServerTimerV7
	cmd.StatusTimerV7 = c.StatusTimerV7
	cmd.StdinTimeoutAbortV7 = c.StdinTimeoutAbortV7
	cmd.OutfileCheckTimerV7 = c.OutfileCheckTimerV7
	cmd.BitmapMinV7 = c.BitmapMinV7
	cmd.BitmapMaxV7 = c.BitmapMaxV7
	cmd.HwmonTempAbortV7 = c.HwmonTempAbortV7
	cmd.RulesFilesV7 = c.RulesFilesV7
	cmd.BrainPasswordV7 = c.BrainPasswordV7
	cmd.GenerateRulesFuncMinV7 = c.GenerateRulesFuncMinV7
	cmd.GenerateRulesFuncMaxV7 = c.GenerateRulesFuncMaxV7
	cmd.CredentialIDs = append([]string(nil), c.CredentialIDs...)
	cmd.CredentialCollection = c.CredentialCollection
	cmd.IncludeCrackedCredentials = c.IncludeCrackedCredentials
	return cmd
}

func (CrackCommand) FromProtobuf(c *clientpb.CrackCommand) *CrackCommand {
	cmd := &CrackCommand{}
	cmd.AttackMode = int32(c.AttackMode)
	cmd.HashType = int32(c.HashType)
	cmd.Hashes = c.Hashes
	// --version
	// --help
	cmd.Quiet = c.Quiet
	cmd.HexCharset = c.HexCharset
	cmd.HexSalt = c.HexSalt
	cmd.HexWordlist = c.HexWordlist
	cmd.Force = c.Force
	cmd.DeprecatedCheckDisable = c.DeprecatedCheckDisable
	cmd.Status = c.Status
	cmd.StatusJSON = c.StatusJSON
	cmd.StatusTimer = c.StatusTimer
	cmd.StdinTimeoutAbort = c.StdinTimeoutAbort
	cmd.MachineReadable = c.MachineReadable
	cmd.KeepGuessing = c.KeepGuessing
	cmd.SelfTestDisable = c.SelfTestDisable
	cmd.Loopback = c.Loopback
	cmd.MarkovHcstat2 = c.MarkovHcstat2
	cmd.MarkovDisable = c.MarkovDisable
	cmd.MarkovClassic = c.MarkovClassic
	cmd.MarkovInverse = c.MarkovInverse
	cmd.MarkovThreshold = c.MarkovThreshold
	cmd.Runtime = c.Runtime
	cmd.Session = c.Session
	cmd.Restore = c.Restore
	cmd.RestoreDisable = c.RestoreDisable
	cmd.RestoreFile = c.RestoreFile
	cmd.Outfile = c.Outfile
	cmd.OutfileFormat = []int32{}
	for _, f := range c.OutfileFormat {
		cmd.OutfileFormat = append(cmd.OutfileFormat, int32(f))
	}
	cmd.OutfileAutohexDisable = c.OutfileAutohexDisable
	cmd.OutfileCheckTimer = c.OutfileCheckTimer
	cmd.WordlistAutohexDisable = c.WordlistAutohexDisable
	cmd.Separator = c.Separator
	cmd.Stdout = c.Stdout
	cmd.Show = c.Show
	cmd.Left = c.Left
	cmd.Username = c.Username
	cmd.Remove = c.Remove
	cmd.RemoveTimer = c.RemoveTimer
	cmd.PotfileDisable = c.PotfileDisable
	cmd.Potfile = c.Potfile
	cmd.EncodingFrom = int32(c.EncodingFrom)
	cmd.EncodingTo = int32(c.EncodingTo)
	cmd.DebugMode = c.DebugMode
	cmd.DebugFile = c.DebugFile
	cmd.InductionDir = c.InductionDir
	cmd.OutfileCheckDir = c.OutfileCheckDir
	cmd.LogfileDisable = c.LogfileDisable
	cmd.HccapxMessagePair = c.HccapxMessagePair
	cmd.NonceErrorCorrections = c.NonceErrorCorrections
	cmd.KeyboardLayoutMapping = c.KeyboardLayoutMapping
	cmd.TruecryptKeyfiles = c.TruecryptKeyfiles
	cmd.VeracryptKeyfiles = c.VeracryptKeyfiles
	cmd.VeracryptPimStart = c.VeracryptPimStart
	cmd.VeracryptPimStop = c.VeracryptPimStop
	cmd.Benchmark = c.Benchmark
	cmd.BenchmarkAll = c.BenchmarkAll
	cmd.SpeedOnly = c.SpeedOnly
	cmd.ProgressOnly = c.ProgressOnly
	cmd.SegmentSize = c.SegmentSize
	cmd.BitmapMin = c.BitmapMin
	cmd.BitmapMax = c.BitmapMax
	cmd.CPUAffinity = c.CPUAffinity
	cmd.HookThreads = c.HookThreads
	cmd.HashInfo = c.HashInfo
	// --example-hashes (66)
	cmd.BackendIgnoreCUDA = c.BackendIgnoreCUDA
	cmd.BackendIgnoreHip = c.BackendIgnoreHip
	cmd.BackendIgnoreMetal = c.BackendIgnoreMetal
	cmd.BackendIgnoreOpenCL = c.BackendIgnoreOpenCL
	cmd.BackendInfo = c.BackendInfo
	cmd.BackendDevices = c.BackendDevices
	cmd.OpenCLDeviceTypes = c.OpenCLDeviceTypes
	cmd.OptimizedKernelEnable = c.OptimizedKernelEnable
	cmd.MultiplyAccelDisabled = c.MultiplyAccelDisabled
	cmd.WorkloadProfile = int32(c.WorkloadProfile)
	cmd.KernelAccel = c.KernelAccel
	cmd.KernelLoops = c.KernelLoops
	cmd.KernelThreads = c.KernelThreads
	cmd.BackendVectorWidth = c.BackendVectorWidth
	cmd.SpinDamp = c.SpinDamp
	cmd.HwmonDisable = c.HwmonDisable
	cmd.HwmonTempAbort = c.HwmonTempAbort
	cmd.ScryptTMTO = c.ScryptTMTO
	cmd.Skip = c.Skip
	cmd.Limit = c.Limit
	cmd.Keyspace = c.Keyspace
	cmd.RuleLeft = c.RuleLeft
	cmd.RuleRight = c.RuleRight
	cmd.RulesFile = c.RulesFile
	cmd.GenerateRules = c.GenerateRules
	cmd.GenerateRulesFunMin = c.GenerateRulesFunMin
	cmd.GenerateRulesFunMax = c.GenerateRulesFunMax
	cmd.GenerateRulesFuncSel = c.GenerateRulesFuncSel
	cmd.GenerateRulesSeed = c.GenerateRulesSeed
	cmd.CustomCharset1 = c.CustomCharset1
	cmd.CustomCharset2 = c.CustomCharset2
	cmd.CustomCharset3 = c.CustomCharset3
	cmd.CustomCharset4 = c.CustomCharset4
	cmd.Identify = c.Identify
	cmd.Increment = c.Increment
	cmd.IncrementMin = c.IncrementMin
	cmd.IncrementMax = c.IncrementMax
	cmd.SlowCandidates = c.SlowCandidates
	cmd.BrainServer = c.BrainServer
	cmd.BrainServerTimer = c.BrainServerTimer
	cmd.BrainClient = c.BrainClient
	cmd.BrainClientFeatures = c.BrainClientFeatures
	cmd.BrainHost = c.BrainHost
	cmd.BrainPort = c.BrainPort
	cmd.BrainPassword = c.BrainPassword
	cmd.BrainSession = c.BrainSession
	cmd.BrainSessionWhitelist = c.BrainSessionWhitelist
	cmd.PipelineStats = c.PipelineStats
	cmd.TaskTimeBreakdown = c.TaskTimeBreakdown
	cmd.MetalCompilerRuntime = c.MetalCompilerRuntime
	cmd.RestorePosition = c.RestorePosition
	cmd.OutfileJSON = c.OutfileJSON
	cmd.DynamicX = c.DynamicX
	cmd.SeekDBPath = c.SeekDBPath
	cmd.BenchmarkMin = c.BenchmarkMin
	cmd.BenchmarkMax = c.BenchmarkMax
	cmd.BridgeParameter1 = c.BridgeParameter1
	cmd.BridgeParameter2 = c.BridgeParameter2
	cmd.BridgeParameter3 = c.BridgeParameter3
	cmd.BridgeParameter4 = c.BridgeParameter4
	cmd.BackendDevicesVirtMulti = c.BackendDevicesVirtMulti
	cmd.BackendDevicesVirtHost = c.BackendDevicesVirtHost
	cmd.TotalCandidates = c.TotalCandidates
	cmd.Lookup = c.Lookup
	cmd.CustomCharset5 = c.CustomCharset5
	cmd.CustomCharset6 = c.CustomCharset6
	cmd.CustomCharset7 = c.CustomCharset7
	cmd.CustomCharset8 = c.CustomCharset8
	cmd.IncrementInverse = c.IncrementInverse
	cmd.BypassDelay = c.BypassDelay
	cmd.BypassThreshold = c.BypassThreshold
	cmd.BrainFeed = c.BrainFeed
	cmd.ColorCracked = c.ColorCracked
	cmd.HashCopy = c.HashCopy
	cmd.EncryptWithPubkey = c.EncryptWithPubkey
	cmd.IdentifyMode = c.IdentifyMode
	cmd.PositionalArguments = c.PositionalArguments
	cmd.EncodingFromName = c.EncodingFromName
	cmd.EncodingToName = c.EncodingToName
	cmd.HashInfoLevel = c.HashInfoLevel
	cmd.BackendInfoLevel = c.BackendInfoLevel
	cmd.HccapxMessagePairV7 = c.HccapxMessagePairV7
	cmd.NonceErrorCorrectionsV7 = c.NonceErrorCorrectionsV7
	cmd.ScryptTMTOV7 = c.ScryptTMTOV7
	cmd.GenerateRulesSeedV7 = c.GenerateRulesSeedV7
	cmd.BrainClientFeaturesV7 = c.BrainClientFeaturesV7
	cmd.BrainSessionV7 = c.BrainSessionV7
	cmd.BrainSessionWhitelistV7 = c.BrainSessionWhitelistV7
	cmd.Stdin = c.Stdin
	cmd.AdviceDisable = c.AdviceDisable
	cmd.HashMode = c.HashMode
	cmd.RestoreShowCommand = c.RestoreShowCommand
	cmd.BrainServerTimerV7 = c.BrainServerTimerV7
	cmd.StatusTimerV7 = c.StatusTimerV7
	cmd.StdinTimeoutAbortV7 = c.StdinTimeoutAbortV7
	cmd.OutfileCheckTimerV7 = c.OutfileCheckTimerV7
	cmd.BitmapMinV7 = c.BitmapMinV7
	cmd.BitmapMaxV7 = c.BitmapMaxV7
	cmd.HwmonTempAbortV7 = c.HwmonTempAbortV7
	cmd.RulesFilesV7 = c.RulesFilesV7
	cmd.BrainPasswordV7 = c.BrainPasswordV7
	cmd.GenerateRulesFuncMinV7 = c.GenerateRulesFuncMinV7
	cmd.GenerateRulesFuncMaxV7 = c.GenerateRulesFuncMaxV7
	cmd.CredentialIDs = append([]string(nil), c.CredentialIDs...)
	cmd.CredentialCollection = c.CredentialCollection
	cmd.IncludeCrackedCredentials = c.IncludeCrackedCredentials
	return cmd
}
