package rpc

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCrackTopJobCancellationOverridesFailedTaskFallback(t *testing.T) {
	now := time.Now()
	job := &models.CrackJob{ID: models.NewUUID(), CompletedAt: now, CancelledAt: now}
	if got := crackTopJobToProtobuf(job, true).GetStatus(); got != clientpb.CrackJobStatus_CANCELLED {
		t.Fatalf("cancelled job with failed task status = %s, want CANCELLED", got)
	}
}

//nolint:gocyclo // Active-job inclusion, bounded history, ordering, and redaction form one snapshot contract.
func TestCrackTopIncludesEveryActiveJobAndBoundsHistory(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	base := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)

	const activeCount = crackTopHistoryLimit + 1
	activeJobs := make([]models.CrackJob, activeCount)
	activeIDs := make(map[string]struct{}, activeCount)
	for index := range activeJobs {
		activeJobs[index] = models.CrackJob{
			ID:        models.NewUUID(),
			CreatedAt: base.Add(time.Duration(index) * time.Second),
			UpdatedAt: base.Add(time.Duration(index) * time.Second),
		}
		activeIDs[activeJobs[index].ID.String()] = struct{}{}
	}
	activeJobs[0].PausedAt = base.Add(time.Hour)
	pausedID := activeJobs[0].ID.String()
	if err := database.Omit("Tasks", "Command", "Results").Create(&activeJobs).Error; err != nil {
		t.Fatalf("create active crack jobs: %v", err)
	}

	const historyCount = crackTopHistoryLimit + 5
	historyJobs := make([]models.CrackJob, historyCount)
	for index := range historyJobs {
		createdAt := base.Add(time.Duration(activeCount+index) * time.Second)
		historyJobs[index] = models.CrackJob{
			ID:          models.NewUUID(),
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
			CompletedAt: createdAt.Add(time.Minute),
		}
	}
	// The oldest-created job completed most recently. Recent history must be
	// selected by completion time, not creation time.
	historyJobs[0].CompletedAt = base.Add(48 * time.Hour)
	if err := database.Omit("Tasks", "Command", "Results").Create(&historyJobs).Error; err != nil {
		t.Fatalf("create crack job history: %v", err)
	}
	failedHistoryTask := &models.CrackTask{
		CrackJobID:       historyJobs[0].ID,
		State:            int32(clientpb.CrackTaskState_CRACK_TASK_FAILED),
		LatestStatusJSON: []byte(`{"progress":[1,2],"devices":[{"speed":1234}]}`),
	}
	if err := database.Omit("Command").Create(failedHistoryTask).Error; err != nil {
		t.Fatalf("create failed history task: %v", err)
	}

	snapshot, err := (&Server{}).CrackTop(t.Context(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("load crack top snapshot: %v", err)
	}
	if got, want := len(snapshot.GetJobs()), activeCount+crackTopHistoryLimit; got != want {
		t.Fatalf("snapshot job count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(snapshot.GetJobs()))
	for index, job := range snapshot.GetJobs() {
		if _, duplicate := seen[job.GetID()]; duplicate {
			t.Fatalf("snapshot contains duplicate job %q", job.GetID())
		}
		seen[job.GetID()] = struct{}{}
		if index > 0 && snapshot.GetJobs()[index-1].GetCreatedAt() < job.GetCreatedAt() {
			t.Fatalf("snapshot jobs are not ordered newest first at index %d", index)
		}
	}
	for id := range activeIDs {
		if _, ok := seen[id]; !ok {
			t.Fatalf("active job %s was omitted", id)
		}
	}
	for _, job := range snapshot.GetJobs() {
		if job.GetID() == pausedID && job.GetStatus() != clientpb.CrackJobStatus_PAUSED {
			t.Fatalf("paused active job status = %s, want PAUSED", job.GetStatus())
		}
	}
	for index, job := range historyJobs {
		_, included := seen[job.ID.String()]
		wantIncluded := index == 0 || index >= historyCount-(crackTopHistoryLimit-1)
		if included != wantIncluded {
			t.Fatalf("history job %d inclusion = %v, want %v", index, included, wantIncluded)
		}
	}
	for _, job := range snapshot.GetJobs() {
		if job.GetID() != historyJobs[0].ID.String() {
			continue
		}
		if job.GetStatus() != clientpb.CrackJobStatus_FAILED {
			t.Fatalf("recent failed history status = %s, want FAILED", job.GetStatus())
		}
		if len(job.GetTasks()) != 0 {
			t.Fatalf("terminal history included task telemetry: %#v", job.GetTasks())
		}
	}
}

//nolint:gocyclo // This is the wire-level redaction and completeness contract for the dashboard snapshot.
func TestCrackTopReturnsLeanTelemetryCountsAndConnectedStations(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hashMode := uint32(22000)
	job := createQueueTestJob(t, database, models.CrackCommand{
		AttackMode:          int32(clientpb.CrackAttackMode_BRUTEFORCE),
		HashType:            int32(clientpb.HashType_MD5),
		HashMode:            &hashMode,
		Hashes:              []string{"secret-parent-hash"},
		PositionalArguments: []string{"secret-parent-target"},
	})
	job.ResultFileID = "secret-result-file-id"
	if err := database.Model(job).Update("result_file_id", job.ResultFileID).Error; err != nil {
		t.Fatalf("set result file id: %v", err)
	}

	stationID := models.NewUUID()
	latestStatus := []byte(`{"progress":[25,100],"target":"secret-status-target","devices":[{"device_id":1,"device_name":"secret-device-name","speed":1234,"temp":65,"util":98},{"device_id":2,"speed":"66","temp":"unavailable","util":0}]}`)
	now := time.Now().UTC().Truncate(time.Second)
	task := &models.CrackTask{
		CrackJobID:       job.ID,
		CrackstationID:   stationID,
		CreatedAt:        now.Add(-time.Minute),
		UpdatedAt:        now,
		StartedAt:        now.Add(-30 * time.Second),
		LeaseExpiresAt:   now.Add(time.Minute),
		LastHeartbeatAt:  now,
		Kind:             int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		State:            int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		Attempt:          3,
		LeaseToken:       "secret-lease-token",
		Keyspace:         "100",
		LatestStatusJSON: latestStatus,
		RecoveredJSON:    []byte(`[{"hash":"secret-recovered-hash","plaintext":"secret-recovered-plaintext"}]`),
		ShardSkip:        10,
		ShardLimit:       90,
		Stdout:           []byte("secret-stdout"),
		Stderr:           []byte("secret-stderr"),
	}
	if err := database.Omit("Command").Create(task).Error; err != nil {
		t.Fatalf("create crack task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{
		CrackTaskID:         task.ID,
		Hashes:              []string{"secret-task-hash"},
		PositionalArguments: []string{"secret-task-target"},
	}).Error; err != nil {
		t.Fatalf("create task command: %v", err)
	}
	results := []models.CrackResult{
		{CrackJobID: job.ID, CrackTaskID: task.ID, Hash: "secret-result-hash-1", Plaintext: []byte("secret-plaintext-1"), Fingerprint: strings.Repeat("1", 64)},
		{CrackJobID: job.ID, CrackTaskID: task.ID, Hash: "secret-result-hash-2", Plaintext: []byte("secret-plaintext-2"), Fingerprint: strings.Repeat("2", 64)},
	}
	if err := database.Create(&results).Error; err != nil {
		t.Fatalf("create crack results: %v", err)
	}

	stationIDText := stationID.String()
	station := core.NewCrackstation(&clientpb.Crackstation{
		ID:           "station-record-id",
		HostUUID:     stationIDText,
		Name:         "online-station",
		Benchmarks:   map[int32]uint64{int32(clientpb.HashType_MD5): 987654321},
		OperatorName: "secret-station-operator",
		OpenCL:       []*clientpb.OpenCLBackendInfo{{Name: "secret-backend-name"}},
	})
	station.UpdateStatus(&clientpb.CrackstationStatus{
		HostUUID:          stationIDText,
		Name:              "online-station",
		State:             clientpb.States_CRACKING,
		CurrentCrackJobID: job.ID.String(),
	})
	if err := core.AddCrackstation(station); err != nil {
		t.Fatalf("add connected crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(stationIDText) })

	before := time.Now().Add(-time.Second).Unix()
	snapshot, err := (&Server{}).CrackTop(t.Context(), &commonpb.Empty{})
	after := time.Now().Add(time.Second).Unix()
	if err != nil {
		t.Fatalf("load crack top snapshot: %v", err)
	}
	if snapshot.GetObservedAt() < before || snapshot.GetObservedAt() > after {
		t.Fatalf("observed timestamp = %d, want [%d,%d]", snapshot.GetObservedAt(), before, after)
	}
	if len(snapshot.GetCrackstations()) != 1 {
		t.Fatalf("connected crackstations = %#v", snapshot.GetCrackstations())
	}
	gotStation := snapshot.GetCrackstations()[0]
	if gotStation.GetID() != "station-record-id" || gotStation.GetName() != "online-station" || gotStation.GetHostUUID() != stationIDText ||
		!gotStation.GetStatusAvailable() || gotStation.GetState() != clientpb.States_CRACKING || gotStation.GetCurrentCrackJobID() != job.ID.String() {
		t.Fatalf("lean connected crackstation = %#v", gotStation)
	}
	if len(snapshot.GetJobs()) != 1 || len(snapshot.GetJobs()[0].GetTasks()) != 1 {
		t.Fatalf("snapshot jobs = %#v", snapshot.GetJobs())
	}
	gotJob := snapshot.GetJobs()[0]
	if gotJob.GetResultCount() != 2 {
		t.Fatalf("result count = %d, want 2", gotJob.GetResultCount())
	}
	if gotJob.GetAttackMode() != clientpb.CrackAttackMode_BRUTEFORCE || gotJob.GetHashType() != clientpb.HashType_MD5 || gotJob.GetHashMode() != hashMode {
		t.Fatalf("parent mode metadata = %#v", gotJob)
	}
	gotTask := gotJob.GetTasks()[0]
	if gotTask.GetHostUUID() != stationIDText || gotTask.GetState() != clientpb.CrackTaskState_CRACK_TASK_RUNNING ||
		gotTask.GetAttempt() != task.Attempt || gotTask.GetShardSkip() != task.ShardSkip || gotTask.GetShardLimit() != task.ShardLimit ||
		gotTask.GetProgressCurrent() != "25" || gotTask.GetProgressTotal() != "100" || !gotTask.GetStatusAvailable() || gotTask.GetStatusParseError() {
		t.Fatalf("task telemetry metadata = %#v", gotTask)
	}
	if len(gotTask.GetDevices()) != 2 || gotTask.GetDevices()[0].GetID() != "1" ||
		gotTask.GetDevices()[0].GetSpeed() != 1234 || !gotTask.GetDevices()[0].GetSpeedAvailable() ||
		gotTask.GetDevices()[0].GetTemperature() != 65 || !gotTask.GetDevices()[0].GetTemperatureAvailable() ||
		gotTask.GetDevices()[1].GetSpeed() != 66 || !gotTask.GetDevices()[1].GetSpeedAvailable() ||
		gotTask.GetDevices()[1].GetTemperatureAvailable() || !gotTask.GetDevices()[1].GetUtilizationAvailable() {
		t.Fatalf("parsed device telemetry = %#v", gotTask.GetDevices())
	}

	serialized, err := protojson.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	for _, secret := range []string{
		"secret-parent-hash", "secret-parent-target", "secret-result-file-id", "secret-lease-token",
		"secret-task-hash", "secret-task-target", "secret-status-target", "secret-device-name",
		"secret-recovered-hash", "secret-recovered-plaintext", "secret-stdout", "secret-stderr",
		"secret-result-hash", "secret-plaintext", "secret-station-operator", "secret-backend-name", "987654321",
	} {
		if strings.Contains(string(serialized), secret) {
			t.Fatalf("snapshot contains secret or heavy field %q: %s", secret, serialized)
		}
	}
}

//nolint:gocyclo // The cases jointly verify status availability, parsing, and progress trust boundaries.
func TestCrackTopStatusAvailabilityAndParseErrors(t *testing.T) {
	missing := crackTopTaskToProtobuf(&models.CrackTask{}, "")
	if missing.GetStatusAvailable() || missing.GetStatusParseError() {
		t.Fatalf("missing status flags = available %v parse-error %v", missing.GetStatusAvailable(), missing.GetStatusParseError())
	}

	malformed := crackTopTaskToProtobuf(&models.CrackTask{LatestStatusJSON: []byte(`{"progress":`)}, "")
	if malformed.GetStatusAvailable() || !malformed.GetStatusParseError() {
		t.Fatalf("malformed status flags = available %v parse-error %v", malformed.GetStatusAvailable(), malformed.GetStatusParseError())
	}

	invalidOptionalMetric := crackTopTaskToProtobuf(&models.CrackTask{LatestStatusJSON: []byte(`{"progress":[1,2],"devices":[{"speed":10,"temp":"n/a"}]}`)}, "")
	if !invalidOptionalMetric.GetStatusAvailable() || invalidOptionalMetric.GetStatusParseError() ||
		len(invalidOptionalMetric.GetDevices()) != 1 || !invalidOptionalMetric.GetDevices()[0].GetSpeedAvailable() ||
		invalidOptionalMetric.GetDevices()[0].GetTemperatureAvailable() {
		t.Fatalf("optional metric status = %#v", invalidOptionalMetric)
	}
	missingSpeed := crackTopTaskToProtobuf(&models.CrackTask{LatestStatusJSON: []byte(`{"devices":[{"temp":60}]}`)}, "")
	if !missingSpeed.GetStatusAvailable() || !missingSpeed.GetStatusParseError() {
		t.Fatalf("missing speed status = %#v", missingSpeed)
	}

	deviceJSON := strings.Repeat(`{"speed":1},`, crackTopMaxDevicesPerTask) + `{"speed":1}`
	truncated := crackTopTaskToProtobuf(&models.CrackTask{LatestStatusJSON: []byte(`{"devices":[` + deviceJSON + `]}`)}, "")
	if !truncated.GetStatusAvailable() || !truncated.GetStatusParseError() || len(truncated.GetDevices()) != crackTopMaxDevicesPerTask {
		t.Fatalf("truncated device status = %#v", truncated)
	}

	exponential := crackTopTaskToProtobuf(&models.CrackTask{LatestStatusJSON: []byte(`{"progress":["1e1000000","1e1000000"],"devices":[{"speed":10}]}`)}, "")
	if !exponential.GetStatusAvailable() || !exponential.GetStatusParseError() ||
		len(exponential.GetDevices()) != 1 || !exponential.GetDevices()[0].GetSpeedAvailable() {
		t.Fatalf("exponential progress was not rejected with bounded device parsing: %#v", exponential)
	}
}

func TestCrackTopProgressNormalizationIsAmplificationSafeAndVersionConservative(t *testing.T) {
	absolute := crackTopTaskToProtobuf(&models.CrackTask{
		ShardSkip:        1_000_000,
		ShardLimit:       20_000_000,
		LatestStatusJSON: []byte(`{"progress":[9000372736,21000000000],"devices":[{"speed":1234}]}`),
	}, "v7.1.2")
	if !absolute.GetProgressExact() {
		t.Fatalf("official v7.1.2 progress is not exact: %#v", absolute)
	}
	got, ok := new(big.Rat).SetString(absolute.GetProgressCurrent() + "/" + absolute.GetProgressTotal())
	if !ok {
		t.Fatalf("normalized progress pair is invalid: %q/%q", absolute.GetProgressCurrent(), absolute.GetProgressTotal())
	}
	want := new(big.Rat).SetFrac(big.NewInt(8_000_372_736), big.NewInt(20_000_000_000))
	if got.Cmp(want) != 0 {
		t.Fatalf("normalized amplified progress = %s, want %s", got.RatString(), want.RatString())
	}

	for _, version := range []string{"v7.1.3", "v7.1.2-custom", "git-24d67b5", "unknown", ""} {
		result := crackTopTaskToProtobuf(&models.CrackTask{
			ShardSkip:        1_000_000,
			ShardLimit:       20_000_000,
			LatestStatusJSON: []byte(`{"progress":[5898240000,20000000000],"devices":[{"speed":1234}]}`),
		}, version)
		if result.GetProgressExact() {
			t.Errorf("version %q guessed nonzero-skip progress semantics", version)
		}
		if !result.GetStatusAvailable() || result.GetStatusParseError() || len(result.GetDevices()) != 1 || !result.GetDevices()[0].GetSpeedAvailable() {
			t.Errorf("version %q lost valid non-progress telemetry: %#v", version, result)
		}
	}

	zeroSkip := crackTopTaskToProtobuf(&models.CrackTask{
		ShardLimit:       20_000_000,
		LatestStatusJSON: []byte(`{"progress":[5898240000,20000000000]}`),
	}, "git-24d67b5")
	if !zeroSkip.GetProgressExact() {
		t.Fatalf("zero-skip progress should be unambiguous: %#v", zeroSkip)
	}
}

func TestCrackTopAbsoluteProgressVersionAcceptsOnlyKnownOfficialTags(t *testing.T) {
	for _, version := range []string{"v6.2.6", "6.2.6", "v7.1.2", "7.1.2"} {
		if !crackTopAbsoluteProgressVersion(version) {
			t.Errorf("known official version %q was not recognized", version)
		}
	}
	for _, version := range []string{"", "unknown", "v7.1.3", "v7.1.2-custom", "v7.1.2+git", "hashcat v7.1.2"} {
		if crackTopAbsoluteProgressVersion(version) {
			t.Errorf("unknown or changed version %q was treated as absolute", version)
		}
	}
}
