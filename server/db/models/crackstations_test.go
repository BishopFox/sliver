package models

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyCrackCommandSlices struct {
	ID                UUID     `gorm:"primaryKey;type:uuid;"`
	Hashes            []string `gorm:"type:text"`
	OutfileFormat     []int32  `gorm:"type:integer[]"`
	CPUAffinity       []uint32 `gorm:"type:integer[]"`
	BackendDevices    []uint32 `gorm:"type:integer[]"`
	OpenCLDeviceTypes []uint32 `gorm:"type:integer[]"`
}

func (legacyCrackCommandSlices) TableName() string {
	return "crack_commands"
}

func TestCrackCommandExtendedFieldsProtobufRoundTrip(t *testing.T) {
	hccapxMessagePairV7 := uint32(0)
	nonceErrorCorrectionsV7 := uint32(89)
	scryptTMTOV7 := uint32(97)
	generateRulesSeedV7 := uint32(101)
	hashMode := uint32(0)
	veracryptPimStart := uint32(17)
	veracryptPimStop := uint32(29)
	benchmarkMax := uint32(67)
	bypassDelay := uint32(79)
	bypassThreshold := uint32(83)
	brainSessionV7 := uint32(113)
	brainServerTimerV7 := uint32(0)
	statusTimerV7 := uint32(0)
	stdinTimeoutAbortV7 := uint32(0)
	outfileCheckTimerV7 := uint32(0)
	bitmapMinV7 := uint32(0)
	bitmapMaxV7 := uint32(0)
	hwmonTempAbortV7 := uint32(0)
	brainPasswordV7 := ""

	want := &CrackCommand{
		MarkovHcstat2:           []byte{0x00, 0x01, 0xff},
		RestoreFile:             []byte("restore data"),
		Outfile:                 "results.out",
		OutfileFormat:           []int32{1, 3},
		Potfile:                 []byte("hash:plain"),
		DebugFile:               "debug.log",
		InductionDir:            "induction",
		OutfileCheckDir:         "outfiles",
		KeyboardLayoutMapping:   []byte("keymap"),
		TruecryptKeyfiles:       "truecrypt.keys",
		VeracryptKeyfiles:       "veracrypt.keys",
		VeracryptPimStart:       &veracryptPimStart,
		VeracryptPimStop:        &veracryptPimStop,
		RuleLeft:                "l",
		RuleRight:               "r",
		RulesFile:               []byte(":\n$1"),
		PipelineStats:           true,
		TaskTimeBreakdown:       true,
		MetalCompilerRuntime:    41,
		RestorePosition:         true,
		OutfileJSON:             true,
		DynamicX:                true,
		SeekDBPath:              "hashcat.db",
		BenchmarkMin:            53,
		BenchmarkMax:            &benchmarkMax,
		BridgeParameter1:        "bridge-one",
		BridgeParameter2:        "bridge-two",
		BridgeParameter3:        "bridge-three",
		BridgeParameter4:        "bridge-four",
		BackendDevicesVirtMulti: 71,
		BackendDevicesVirtHost:  73,
		TotalCandidates:         true,
		Lookup:                  "lookup-value",
		CustomCharset5:          "charset-five",
		CustomCharset6:          "charset-six",
		CustomCharset7:          "charset-seven",
		CustomCharset8:          "charset-eight",
		IncrementInverse:        true,
		BypassDelay:             &bypassDelay,
		BypassThreshold:         &bypassThreshold,
		BrainFeed:               true,
		ColorCracked:            true,
		HashCopy:                true,
		EncryptWithPubkey:       "age1example",
		IdentifyMode:            true,
		PositionalArguments:     []string{"hashes.txt", "words.txt", "?d?d?d?d"},
		EncodingFromName:        "utf-8",
		EncodingToName:          "utf-16le",
		HashInfoLevel:           103,
		BackendInfoLevel:        107,
		HccapxMessagePairV7:     &hccapxMessagePairV7,
		NonceErrorCorrectionsV7: &nonceErrorCorrectionsV7,
		ScryptTMTOV7:            &scryptTMTOV7,
		GenerateRulesSeedV7:     &generateRulesSeedV7,
		BrainClientFeaturesV7:   109,
		BrainSessionV7:          &brainSessionV7,
		BrainSessionWhitelistV7: []uint32{127, 131},
		Stdin:                   []byte("candidate one\ncandidate two\x00"),
		AdviceDisable:           true,
		HashMode:                &hashMode,
		RestoreShowCommand:      true,
		BrainServerTimerV7:      &brainServerTimerV7,
		StatusTimerV7:           &statusTimerV7,
		StdinTimeoutAbortV7:     &stdinTimeoutAbortV7,
		OutfileCheckTimerV7:     &outfileCheckTimerV7,
		BitmapMinV7:             &bitmapMinV7,
		BitmapMaxV7:             &bitmapMaxV7,
		HwmonTempAbortV7:        &hwmonTempAbortV7,
		RulesFilesV7:            [][]byte{[]byte(":"), []byte("$1")},
		BrainPasswordV7:         &brainPasswordV7,
		GenerateRulesFuncMinV7:  &statusTimerV7,
		GenerateRulesFuncMaxV7:  &statusTimerV7,
	}

	got := (CrackCommand{}).FromProtobuf(want.ToProtobuf())
	if !reflect.DeepEqual(got, want) {
		t.Fatal("CrackCommand protobuf round trip mismatch")
	}
}

//nolint:gocyclo // The round trip verifies every persisted task field and protobuf representation together.
func TestCrackTaskProtobufRoundTrip(t *testing.T) {
	want := &CrackTask{
		ID:               ParseUUIDOrNil("3f2a6f33-586f-4af4-9f86-175db84bf2ab"),
		CrackstationID:   ParseUUIDOrNil("bd7d24e8-ea46-40ce-8755-cb5400970cca"),
		CreatedAt:        time.Unix(1_725_000_001, 0).UTC(),
		StartedAt:        time.Unix(1_725_000_002, 0).UTC(),
		CompletedAt:      time.Unix(1_725_000_003, 0).UTC(),
		Err:              "hashcat exited unsuccessfully",
		Stdout:           []byte("stdout bytes\x00"),
		Stderr:           []byte("stderr bytes\xff"),
		ExitCode:         -2,
		StdoutTruncated:  true,
		StderrTruncated:  true,
		StdoutTotalBytes: 30,
		StderrTotalBytes: 40,
		Command: CrackCommand{
			Quiet:         true,
			OutfileFormat: []int32{2},
		},
	}

	protobuf := want.ToProtobuf()
	if protobuf.ID != want.ID.String() {
		t.Fatalf("protobuf ID = %q, want %q", protobuf.ID, want.ID.String())
	}
	if protobuf.HostUUID != want.CrackstationID.String() {
		t.Fatalf("protobuf HostUUID = %q, want %q", protobuf.HostUUID, want.CrackstationID.String())
	}

	got := (CrackTask{}).FromProtobuf(protobuf)
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.CrackstationID != want.CrackstationID {
		t.Errorf("CrackstationID = %q, want %q", got.CrackstationID, want.CrackstationID)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if !got.StartedAt.Equal(want.StartedAt) {
		t.Errorf("StartedAt = %s, want %s", got.StartedAt, want.StartedAt)
	}
	if !got.CompletedAt.Equal(want.CompletedAt) {
		t.Errorf("CompletedAt = %s, want %s", got.CompletedAt, want.CompletedAt)
	}
	if got.Err != want.Err {
		t.Errorf("Err = %q, want %q", got.Err, want.Err)
	}
	if !bytes.Equal(got.Stdout, want.Stdout) {
		t.Errorf("Stdout = %q, want %q", got.Stdout, want.Stdout)
	}
	if !bytes.Equal(got.Stderr, want.Stderr) {
		t.Errorf("Stderr = %q, want %q", got.Stderr, want.Stderr)
	}
	if got.ExitCode != want.ExitCode {
		t.Errorf("ExitCode = %d, want %d", got.ExitCode, want.ExitCode)
	}
	if got.StdoutTruncated != want.StdoutTruncated || got.StderrTruncated != want.StderrTruncated {
		t.Errorf("truncation flags = (%v, %v), want (%v, %v)", got.StdoutTruncated, got.StderrTruncated, want.StdoutTruncated, want.StderrTruncated)
	}
	if got.StdoutTotalBytes != want.StdoutTotalBytes || got.StderrTotalBytes != want.StderrTotalBytes {
		t.Errorf("total bytes = (%d, %d), want (%d, %d)", got.StdoutTotalBytes, got.StderrTotalBytes, want.StdoutTotalBytes, want.StderrTotalBytes)
	}
	if !reflect.DeepEqual(got.Command, want.Command) {
		t.Error("CrackTask command protobuf round trip mismatch")
	}
}

func TestCrackTaskZeroTimestampsRemainUnset(t *testing.T) {
	protobuf := (&CrackTask{}).ToProtobuf()
	if protobuf.CreatedAt != 0 || protobuf.StartedAt != 0 || protobuf.CompletedAt != 0 {
		t.Fatalf("zero model timestamps encoded as (%d, %d, %d)", protobuf.CreatedAt, protobuf.StartedAt, protobuf.CompletedAt)
	}

	got := (CrackTask{}).FromProtobuf(&clientpb.CrackTask{})
	if !got.CreatedAt.IsZero() || !got.StartedAt.IsZero() || !got.CompletedAt.IsZero() {
		t.Fatalf("zero protobuf timestamps decoded as (%s, %s, %s)", got.CreatedAt, got.StartedAt, got.CompletedAt)
	}
}

func TestCrackCommandSQLiteRepeatedFieldsRoundTrip(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "crack-command.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := database.AutoMigrate(&CrackCommand{}); err != nil {
		t.Fatalf("migrate CrackCommand: %v", err)
	}
	want := &CrackCommand{
		Hashes:                  []string{"hash-one", "hash,two"},
		OutfileFormat:           []int32{1, 3},
		CPUAffinity:             []uint32{1, 4},
		BackendDevices:          []uint32{2, 5},
		OpenCLDeviceTypes:       []uint32{1, 2, 4},
		PositionalArguments:     []string{"hashes,2026.txt", "wordlist.txt"},
		BrainSessionWhitelistV7: []uint32{0x10, 0x20},
		RulesFilesV7:            [][]byte{[]byte(":"), []byte("$1")},
	}
	if err := database.Create(want).Error; err != nil {
		t.Fatalf("create CrackCommand: %v", err)
	}

	got := &CrackCommand{}
	if err := database.First(got, "id = ?", want.ID).Error; err != nil {
		t.Fatalf("load CrackCommand: %v", err)
	}
	if !reflect.DeepEqual(got.Hashes, want.Hashes) {
		t.Fatalf("Hashes = %#v, want %#v", got.Hashes, want.Hashes)
	}
	if !reflect.DeepEqual(got.OutfileFormat, want.OutfileFormat) {
		t.Fatalf("OutfileFormat = %#v, want %#v", got.OutfileFormat, want.OutfileFormat)
	}
	if !reflect.DeepEqual(got.CPUAffinity, want.CPUAffinity) {
		t.Fatalf("CPUAffinity = %#v, want %#v", got.CPUAffinity, want.CPUAffinity)
	}
	if !reflect.DeepEqual(got.BackendDevices, want.BackendDevices) {
		t.Fatalf("BackendDevices = %#v, want %#v", got.BackendDevices, want.BackendDevices)
	}
	if !reflect.DeepEqual(got.OpenCLDeviceTypes, want.OpenCLDeviceTypes) {
		t.Fatalf("OpenCLDeviceTypes = %#v, want %#v", got.OpenCLDeviceTypes, want.OpenCLDeviceTypes)
	}
	if !reflect.DeepEqual(got.PositionalArguments, want.PositionalArguments) {
		t.Fatalf("PositionalArguments = %#v, want %#v", got.PositionalArguments, want.PositionalArguments)
	}
	if !reflect.DeepEqual(got.BrainSessionWhitelistV7, want.BrainSessionWhitelistV7) {
		t.Fatalf("BrainSessionWhitelistV7 = %#v, want %#v", got.BrainSessionWhitelistV7, want.BrainSessionWhitelistV7)
	}
	if !reflect.DeepEqual(got.RulesFilesV7, want.RulesFilesV7) {
		t.Fatalf("RulesFilesV7 = %#v, want %#v", got.RulesFilesV7, want.RulesFilesV7)
	}
}

func TestCrackTopPollingIndexes(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "crack-top-indexes.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := database.AutoMigrate(&CrackJob{}, &CrackTask{}, &CrackCommand{}); err != nil {
		t.Fatalf("migrate crack top models: %v", err)
	}

	type indexColumn struct {
		Seq  int
		Name string
	}
	var jobColumns []indexColumn
	if err := database.Raw("PRAGMA index_info('idx_crack_jobs_top')").Scan(&jobColumns).Error; err != nil {
		t.Fatalf("inspect crack job polling index: %v", err)
	}
	jobIndexNames := make([]string, 0, len(jobColumns))
	for _, column := range jobColumns {
		jobIndexNames = append(jobIndexNames, column.Name)
	}
	if want := []string{"completed_at", "created_at"}; !reflect.DeepEqual(jobIndexNames, want) {
		t.Fatalf("crack job polling index columns = %#v, want %#v", jobIndexNames, want)
	}

	var taskColumns []indexColumn
	if err := database.Raw("PRAGMA index_info('idx_crack_tasks_job')").Scan(&taskColumns).Error; err != nil {
		t.Fatalf("inspect crack task polling index: %v", err)
	}
	taskIndexNames := make([]string, 0, len(taskColumns))
	for _, column := range taskColumns {
		taskIndexNames = append(taskIndexNames, column.Name)
	}
	if want := []string{"crack_job_id", "state"}; !reflect.DeepEqual(taskIndexNames, want) {
		t.Fatalf("crack task polling index columns = %#v, want %#v", taskIndexNames, want)
	}

	var commandColumns []indexColumn
	if err := database.Raw("PRAGMA index_info('idx_crack_commands_job')").Scan(&commandColumns).Error; err != nil {
		t.Fatalf("inspect crack command polling index: %v", err)
	}
	commandIndexNames := make([]string, 0, len(commandColumns))
	for _, column := range commandColumns {
		commandIndexNames = append(commandIndexNames, column.Name)
	}
	if want := []string{"crack_job_id"}; !reflect.DeepEqual(commandIndexNames, want) {
		t.Fatalf("crack command polling index columns = %#v, want %#v", commandIndexNames, want)
	}
}

//nolint:gocyclo // The migration test compares every legacy scalar slice with its restored representation.
func TestCrackCommandLegacyScalarSlicesSurviveMigration(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-crack-command.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := database.AutoMigrate(&legacyCrackCommandSlices{}); err != nil {
		t.Fatalf("migrate legacy CrackCommand: %v", err)
	}
	legacy := &legacyCrackCommandSlices{
		ID:                NewUUID(),
		Hashes:            []string{"legacy-hash"},
		OutfileFormat:     []int32{3},
		CPUAffinity:       []uint32{4},
		BackendDevices:    []uint32{5},
		OpenCLDeviceTypes: []uint32{2},
	}
	if err := database.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy CrackCommand: %v", err)
	}
	if err := database.AutoMigrate(&CrackCommand{}); err != nil {
		t.Fatalf("migrate current CrackCommand: %v", err)
	}

	got := &CrackCommand{}
	if err := database.First(got, "id = ?", legacy.ID).Error; err != nil {
		t.Fatalf("load migrated CrackCommand: %v", err)
	}
	if !reflect.DeepEqual(got.Hashes, legacy.Hashes) {
		t.Fatalf("Hashes = %#v, want %#v", got.Hashes, legacy.Hashes)
	}
	if !reflect.DeepEqual(got.OutfileFormat, legacy.OutfileFormat) {
		t.Fatalf("OutfileFormat = %#v, want %#v", got.OutfileFormat, legacy.OutfileFormat)
	}
	if !reflect.DeepEqual(got.CPUAffinity, legacy.CPUAffinity) {
		t.Fatalf("CPUAffinity = %#v, want %#v", got.CPUAffinity, legacy.CPUAffinity)
	}
	if !reflect.DeepEqual(got.BackendDevices, legacy.BackendDevices) {
		t.Fatalf("BackendDevices = %#v, want %#v", got.BackendDevices, legacy.BackendDevices)
	}
	if !reflect.DeepEqual(got.OpenCLDeviceTypes, legacy.OpenCLDeviceTypes) {
		t.Fatalf("OpenCLDeviceTypes = %#v, want %#v", got.OpenCLDeviceTypes, legacy.OpenCLDeviceTypes)
	}

	// PostgreSQL casts the old integer[] columns to text using brace-array
	// syntax when AutoMigrate changes them to text. Exercise that transitional
	// representation without requiring a PostgreSQL service in unit tests.
	if err := database.Model(&CrackCommand{}).Where("id = ?", legacy.ID).Updates(map[string]interface{}{
		"outfile_format":       "{1,3}",
		"cpu_affinity":         "{4,6}",
		"backend_devices":      "{5,7}",
		"open_cl_device_types": "{2,4}",
	}).Error; err != nil {
		t.Fatalf("write PostgreSQL-style transitional values: %v", err)
	}
	postgresStyle := &CrackCommand{}
	if err := database.First(postgresStyle, "id = ?", legacy.ID).Error; err != nil {
		t.Fatalf("load PostgreSQL-style transitional values: %v", err)
	}
	if !reflect.DeepEqual(postgresStyle.OutfileFormat, []int32{1, 3}) ||
		!reflect.DeepEqual(postgresStyle.CPUAffinity, []uint32{4, 6}) ||
		!reflect.DeepEqual(postgresStyle.BackendDevices, []uint32{5, 7}) ||
		!reflect.DeepEqual(postgresStyle.OpenCLDeviceTypes, []uint32{2, 4}) {
		t.Fatalf("PostgreSQL-style transitional values were not decoded: %#v", postgresStyle)
	}
}
