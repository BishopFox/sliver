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
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Crackstation - History of crackstation jobs
type Crackstation struct {
	// ID = crackstation name
	ID         uuid.UUID `gorm:"primaryKey;type:uuid;"`
	CreatedAt  time.Time `gorm:"->;<-:create;"`
	Tasks      []CrackTask
	Benchmarks []Benchmark
}

// BeforeCreate - GORM hook
func (c *Crackstation) BeforeCreate(tx *gorm.DB) (err error) {
	c.CreatedAt = time.Now()
	return nil
}

// Benchmark - Performance information about the crackstation
type Benchmark struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	CrackstationID uuid.UUID `gorm:"type:uuid;"`
	HashType       int32
	PerSecondRate  uint64
}

// BeforeCreate - GORM hook
func (b *Benchmark) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	b.CreatedAt = time.Now()
	return nil
}

// CrackFile - Performance information about the crackstation
type CrackFile struct {
	ID               uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt        time.Time `gorm:"->;<-:create;"`
	LastModified     time.Time
	Name             string
	UncompressedSize int64
	Sha2_256         string
	Type             int32
	IsCompressed     bool
	IsComplete       bool

	Chunks []CrackFileChunk
}

// BeforeCreate - GORM hook
func (c *CrackFile) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
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
		Sha2_256:         c.Sha2_256,
		Type:             clientpb.CrackFileType(c.Type),
		IsCompressed:     c.IsCompressed,
		Chunks:           chunks,
	}
}

// CrackFileChunk - Performance information about the crackstation
type CrackFileChunk struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackFileID uuid.UUID `gorm:"type:uuid;"`
	N           uint32
}

// BeforeCreate - GORM hook
func (c *CrackFileChunk) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
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
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	CompletedAt time.Time
	Tasks       []CrackTask

	Command CrackCommand // Parent command
}

func (c *CrackJob) Status() clientpb.CrackJobStatus {
	if c.CompletedAt.IsZero() {
		return clientpb.CrackJobStatus_IN_PROGRESS
	}
	for _, task := range c.Tasks {
		if task.CompletedAt.IsZero() {
			return clientpb.CrackJobStatus_IN_PROGRESS
		}
	}
	return clientpb.CrackJobStatus_COMPLETED
}

// CrackTask - An individual chunk of a job sent to a specific crackstation
type CrackTask struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackstationID uuid.UUID `gorm:"type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	StartedAt      time.Time
	CompletedAt    time.Time

	Command CrackCommand
}

func (c *CrackTask) ToProtobuf() *clientpb.CrackTask {
	task := &clientpb.CrackTask{}
	task.CreatedAt = c.CreatedAt.Unix()
	task.StartedAt = c.StartedAt.Unix()
	task.CompletedAt = c.CompletedAt.Unix()
	task.Command = c.Command.ToProtobuf()
	return task
}

func (CrackTask) FromProtobuf(c *clientpb.CrackTask) *CrackTask {
	task := &CrackTask{}
	task.CreatedAt = time.Unix(c.CreatedAt, 0)
	task.StartedAt = time.Unix(c.StartedAt, 0)
	task.CompletedAt = time.Unix(c.CompletedAt, 0)
	task.Command = (*CrackCommand{}.FromProtobuf(c.Command))
	return task
}

// BeforeCreate - GORM hook
func (c *CrackTask) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

type CrackCommand struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	CrackTaskID uuid.UUID `gorm:"type:uuid;"`

	// FLAGS
	AttackMode             int32
	HashType               int32
	Hashes                 []string `gorm:"type:text"`
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
	// MarkovHcstat2          []byte
	MarkovDisable   bool
	MarkovClassic   bool
	MarkovInverse   bool
	MarkovThreshold uint32
	Runtime         uint32
	Session         string
	Restore         bool
	RestoreDisable  bool
	// RestoreFile            []byte
	// --outfile FILE (28)
	OutfileFormat          []int32 `gorm:"type:integer[]"`
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
	// Potfile                []byte
	EncodingFrom int32
	EncodingTo   int32
	DebugMode    uint32
	// --debug-file FILE (45)
	// --induction-dir DIR (46)
	// --outfile-check-dir DIR (47)
	LogfileDisable        bool
	HccapxMessagePair     uint32
	NonceErrorCorrections uint32
	// KeyboardLayoutMapping []byte
	// --truecrypt-keyfiles FILE (52)
	// --veracrypt-keyfiles FILE (53)
	// --veracrypt-pim-start PIM (54)
	// --veracrypt-pim-stop PIM (55)
	Benchmark    bool
	BenchmarkAll bool
	SpeedOnly    bool
	ProgressOnly bool
	SegmentSize  uint32
	BitmapMin    uint32
	BitmapMax    uint32
	CPUAffinity  []uint32 `gorm:"type:integer[]"`
	HookThreads  uint32
	HashInfo     bool
	// --example-hashes (66)
	BackendIgnoreCUDA     bool
	BackendIgnoreHip      bool
	BackendIgnoreMetal    bool
	BackendIgnoreOpenCL   bool
	BackendInfo           bool
	BackendDevices        []uint32 `gorm:"type:integer[]"`
	OpenCLDeviceTypes     []uint32 `gorm:"type:integer[]"`
	OptimizedKernelEnable bool
	MultiplyAccelDisabled bool
	WorkloadProfile       int32
	KernelAccel           uint32
	KernelLoops           uint32
	KernelThreads         uint32
	BackendVectorWidth    uint32
	SpinDamp              uint32
	HwmonDisable          bool
	HwmonTempAbort        uint32
	ScryptTMTO            uint32
	Skip                  uint64
	Limit                 uint64
	Keyspace              bool
	// --rule-left (88)
	// --rule-right (89)
	// RulesFile             []byte
	GenerateRules         uint32
	GenerateRulesFunMin   uint32
	GenerateRulesFunMax   uint32
	GenerateRulesFuncSel  string
	GenerateRulesSeed     int32
	CustomCharset1        string
	CustomCharset2        string
	CustomCharset3        string
	CustomCharset4        string
	Identify              string
	Increment             bool
	IncrementMin          uint32
	IncrementMax          uint32
	SlowCandidates        bool
	BrainServer           bool
	BrainServerTimer      uint32
	BrainClient           bool
	BrainClientFeatures   string
	BrainHost             string
	BrainPort             uint32
	BrainPassword         string
	BrainSession          string
	BrainSessionWhitelist string
}

// BeforeCreate - GORM hook
func (c *CrackCommand) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
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
	// cmd.MarkovHcstat2 = c.MarkovHcstat2
	cmd.MarkovDisable = c.MarkovDisable
	cmd.MarkovClassic = c.MarkovClassic
	cmd.MarkovInverse = c.MarkovInverse
	cmd.MarkovThreshold = c.MarkovThreshold
	cmd.Runtime = c.Runtime
	cmd.Session = c.Session
	cmd.Restore = c.Restore
	cmd.RestoreDisable = c.RestoreDisable
	// cmd.RestoreFile = c.RestoreFile
	// --outfile FILE (28)
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
	// cmd.Potfile = c.Potfile
	cmd.EncodingFrom = clientpb.CrackEncoding(c.EncodingFrom)
	cmd.EncodingTo = clientpb.CrackEncoding(c.EncodingTo)
	cmd.DebugMode = c.DebugMode
	// --debug-file FILE (45)
	// --induction-dir DIR (46)
	// --outfile-check-dir DIR (47)
	cmd.LogfileDisable = c.LogfileDisable
	cmd.HccapxMessagePair = c.HccapxMessagePair
	cmd.NonceErrorCorrections = c.NonceErrorCorrections
	// cmd.KeyboardLayoutMapping = c.KeyboardLayoutMapping
	// --truecrypt-keyfiles FILE (52)
	// --veracrypt-keyfiles FILE (53)
	// --veracrypt-pim-start PIM (54)
	// --veracrypt-pim-stop PIM (55)
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
	// --rule-left (88)
	// --rule-right (89)
	// cmd.RulesFile = c.RulesFile
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
	// cmd.MarkovHcstat2 = c.MarkovHcstat2
	cmd.MarkovDisable = c.MarkovDisable
	cmd.MarkovClassic = c.MarkovClassic
	cmd.MarkovInverse = c.MarkovInverse
	cmd.MarkovThreshold = c.MarkovThreshold
	cmd.Runtime = c.Runtime
	cmd.Session = c.Session
	cmd.Restore = c.Restore
	cmd.RestoreDisable = c.RestoreDisable
	// cmd.RestoreFile = c.RestoreFile
	// --outfile FILE (28)
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
	// cmd.Potfile = c.Potfile
	cmd.EncodingFrom = int32(c.EncodingFrom)
	cmd.EncodingTo = int32(c.EncodingTo)
	cmd.DebugMode = c.DebugMode
	// --debug-file FILE (45)
	// --induction-dir DIR (46)
	// --outfile-check-dir DIR (47)
	cmd.LogfileDisable = c.LogfileDisable
	cmd.HccapxMessagePair = c.HccapxMessagePair
	cmd.NonceErrorCorrections = c.NonceErrorCorrections
	// cmd.KeyboardLayoutMapping = c.KeyboardLayoutMapping
	// --truecrypt-keyfiles FILE (52)
	// --veracrypt-keyfiles FILE (53)
	// --veracrypt-pim-start PIM (54)
	// --veracrypt-pim-stop PIM (55)
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
	// --rule-left (88)
	// --rule-right (89)
	// cmd.RulesFile = c.RulesFile
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
	return cmd
}
