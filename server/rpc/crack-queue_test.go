package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func TestSplitCrackKeyspaceWeightedExactAndDeterministic(t *testing.T) {
	stations := []weightedCrackstation{{HostUUID: "b", Rate: 1}, {HostUUID: "a", Rate: 3}}
	want := []crackShard{{HostUUID: "a", Skip: 0, Limit: 8}, {HostUUID: "b", Skip: 8, Limit: 2}}
	got := splitCrackKeyspace(10, stations)
	if !slices.Equal(got, want) {
		t.Fatalf("split = %#v, want %#v", got, want)
	}
	reversed := []weightedCrackstation{{HostUUID: "a", Rate: 3}, {HostUUID: "b", Rate: 1}}
	if gotAgain := splitCrackKeyspace(10, reversed); !slices.Equal(gotAgain, want) {
		t.Fatalf("reordered split = %#v, want %#v", gotAgain, want)
	}
	zeroAndDuplicate := splitCrackKeyspace(1, []weightedCrackstation{
		{HostUUID: "b", Rate: 0}, {HostUUID: "a", Rate: 0}, {HostUUID: "a", Rate: 9},
	})
	if len(zeroAndDuplicate) != 1 || zeroAndDuplicate[0].HostUUID != "a" || zeroAndDuplicate[0].Limit != 1 {
		t.Fatalf("zero/duplicate split = %#v", zeroAndDuplicate)
	}
	if got := splitCrackKeyspace(99, nil); got != nil {
		t.Fatalf("empty station split = %#v, want nil", got)
	}
}

func TestEffectiveCrackRange(t *testing.T) {
	tests := []struct {
		keyspace, skip, limit uint64
		wantStart, wantEnd    uint64
	}{
		{100, 0, 0, 0, 100},
		{100, 10, 40, 10, 50},
		{100, 90, 50, 90, 100},
		{100, 200, 10, 100, 100},
	}
	for _, test := range tests {
		start, end := effectiveCrackRange(test.keyspace, test.skip, test.limit)
		if start != test.wantStart || end != test.wantEnd {
			t.Fatalf("range(%d,%d,%d) = [%d,%d), want [%d,%d)", test.keyspace, test.skip, test.limit, start, end, test.wantStart, test.wantEnd)
		}
	}
}

func addQueueTestStation(t *testing.T, database *gorm.DB, idText string, rates map[int32]uint64) *core.Crackstation {
	return addQueueTestStationVersion(t, database, idText, "hashcat-test-v1", rates)
}

func addQueueTestStationVersion(t *testing.T, database *gorm.DB, idText string, hashcatVersion string, rates map[int32]uint64) *core.Crackstation {
	t.Helper()
	id := models.ParseUUIDOrNil(idText)
	if id == models.NilUUID() {
		t.Fatalf("invalid test station UUID %q", idText)
	}
	normalizedVersion := normalizeHashcatVersion(hashcatVersion)
	if err := database.Omit("Tasks", "Benchmarks").Create(&models.Crackstation{
		ID:                      id,
		HashcatVersion:          normalizedVersion,
		BenchmarkHashcatVersion: normalizedVersion,
		BenchmarkSchemaVersion:  crackBenchmarkSchemaVersion,
	}).Error; err != nil {
		t.Fatalf("create crackstation: %v", err)
	}
	for hashType, rate := range rates {
		if err := database.Create(&models.Benchmark{CrackstationID: id, HashType: hashType, PerSecondRate: rate}).Error; err != nil {
			t.Fatalf("create benchmark: %v", err)
		}
	}
	station := core.NewCrackstation(&clientpb.Crackstation{HostUUID: id.String(), OperatorName: "queue-test", HashcatVersion: normalizedVersion})
	if err := core.AddCrackstation(station); err != nil {
		t.Fatalf("add crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(id.String()) })
	return station
}

func createQueueTestJob(t *testing.T, database *gorm.DB, command models.CrackCommand) *models.CrackJob {
	t.Helper()
	job := &models.CrackJob{UpdatedAt: time.Now()}
	if err := database.Omit("Tasks", "Command", "Results").Create(job).Error; err != nil {
		t.Fatalf("create crack job: %v", err)
	}
	command.CrackJobID = job.ID
	if err := database.Create(&command).Error; err != nil {
		t.Fatalf("create parent command: %v", err)
	}
	job.Command = command
	return job
}

//nolint:gocyclo // Benchmark gating, single-task leasing, and persisted state form one scheduler scenario.
func TestSchedulerWaitsForBenchmarkAndLeasesOnlyOneTask(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	job := createQueueTestJob(t, database, models.CrackCommand{})
	for index := 0; index < 2; index++ {
		if err := database.Create(&models.CrackTask{
			CrackJobID: job.ID,
			Kind:       int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
			State:      int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
		}).Error; err != nil {
			t.Fatalf("create queued task: %v", err)
		}
	}
	legacy := &models.CrackTask{State: int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)}
	if err := database.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy task: %v", err)
	}
	benchmarkTask := &models.CrackTask{Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_BENCHMARK), State: int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)}
	if err := database.Create(benchmarkTask).Error; err != nil {
		t.Fatalf("create reserved benchmark task: %v", err)
	}
	activeBenchmarkTask := &models.CrackTask{
		CrackstationID: models.ParseUUIDOrNil(station.HostUUID),
		Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_BENCHMARK),
		State:          int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:        1,
		LeaseToken:     "reserved-benchmark-token",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(activeBenchmarkTask).Error; err != nil {
		t.Fatalf("create active reserved benchmark task: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule without benchmark: %v", err)
	}
	var leased int64
	if err := database.Model(&models.CrackTask{}).
		Where("state = ? AND kind IN ?", int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), schedulableCrackTaskKinds).
		Count(&leased).Error; err != nil {
		t.Fatalf("count leased tasks: %v", err)
	}
	if leased != 0 {
		t.Fatalf("leased tasks before benchmark = %d, want 0", leased)
	}
	if err := database.Create(&models.Benchmark{CrackstationID: models.ParseUUIDOrNil(station.HostUUID), HashType: 0, PerSecondRate: 1}).Error; err != nil {
		t.Fatalf("create benchmark: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule after benchmark: %v", err)
	}
	if err := database.Model(&models.CrackTask{}).
		Where("state = ? AND kind IN ?", int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), schedulableCrackTaskKinds).
		Count(&leased).Error; err != nil {
		t.Fatalf("count leased tasks: %v", err)
	}
	if leased != 1 {
		t.Fatalf("leased tasks = %d, want 1", leased)
	}
	select {
	case event := <-station.Events:
		assignment := crackTaskAssignment{}
		if err := json.Unmarshal(event.Data, &assignment); err != nil {
			t.Fatalf("decode task assignment: %v", err)
		}
		if assignment.TaskID == "" || assignment.HostUUID != station.HostUUID || assignment.Attempt != 1 || assignment.LeaseToken == "" {
			t.Fatalf("task assignment = %#v", assignment)
		}
	case <-time.After(time.Second):
		t.Fatal("task assignment event was not dispatched")
	}
	var gotLegacy models.CrackTask
	if err := database.First(&gotLegacy, "id = ?", legacy.ID).Error; err != nil {
		t.Fatalf("load legacy task: %v", err)
	}
	if gotLegacy.Attempt != 0 || gotLegacy.CrackstationID != models.NilUUID() {
		t.Fatalf("legacy unspecified task was dispatched: %#v", gotLegacy)
	}
	var gotBenchmark models.CrackTask
	if err := database.First(&gotBenchmark, "id = ?", benchmarkTask.ID).Error; err != nil {
		t.Fatalf("load reserved benchmark task: %v", err)
	}
	if gotBenchmark.Attempt != 0 || gotBenchmark.CrackstationID != models.NilUUID() {
		t.Fatalf("reserved benchmark task was dispatched: %#v", gotBenchmark)
	}
	var gotActiveBenchmark models.CrackTask
	if err := database.First(&gotActiveBenchmark, "id = ?", activeBenchmarkTask.ID).Error; err != nil {
		t.Fatalf("load active reserved benchmark task: %v", err)
	}
	if gotActiveBenchmark.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || gotActiveBenchmark.Attempt != 1 || gotActiveBenchmark.LeaseToken != "reserved-benchmark-token" {
		t.Fatalf("active reserved benchmark task was modified: %#v", gotActiveBenchmark)
	}
}

//nolint:gocyclo // The test follows legacy benchmark rejection through freshness repair and task leasing.
func TestSchedulerRejectsLegacyBenchmarkUntilFreshnessMarkerMatches(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hostID := models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111")
	if err := database.Omit("Tasks", "Benchmarks").Create(&models.Crackstation{ID: hostID, HashcatVersion: "hashcat-v1"}).Error; err != nil {
		t.Fatalf("create legacy crackstation: %v", err)
	}
	if err := database.Create(&models.Benchmark{CrackstationID: hostID, HashType: 100, PerSecondRate: 100}).Error; err != nil {
		t.Fatalf("create legacy benchmark: %v", err)
	}
	station := core.NewCrackstation(&clientpb.Crackstation{HostUUID: hostID.String(), OperatorName: "queue-test", HashcatVersion: "hashcat-v1"})
	if err := core.AddCrackstation(station); err != nil {
		t.Fatalf("add legacy runtime crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(hostID.String()) })
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100})
	task := &models.CrackTask{CrackJobID: job.ID, Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE), State: int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create queued task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
		t.Fatalf("create queued task command: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule with legacy benchmark: %v", err)
	}
	if err := database.First(task, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load task after legacy schedule: %v", err)
	}
	if task.State != int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED) || task.Attempt != 0 {
		t.Fatalf("legacy benchmark scheduled task: %#v", task)
	}
	if err := database.Model(&models.Crackstation{}).Where("id = ?", hostID).Updates(map[string]interface{}{
		"benchmark_schema_version": crackBenchmarkSchemaVersion, "benchmark_hashcat_version": "hashcat-v1",
	}).Error; err != nil {
		t.Fatalf("stamp benchmark freshness marker: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule with fresh benchmark: %v", err)
	}
	if err := database.First(task, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load task after fresh schedule: %v", err)
	}
	if task.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || task.Attempt != 1 || task.CrackstationID != hostID {
		t.Fatalf("fresh benchmark did not schedule task: %#v", task)
	}
}

//nolint:gocyclo // The table validates benchmark eligibility and shard weights across selected attack modes.
func TestSchedulerAndShardWeightsRequirePositiveSelectedModeBenchmark(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	unsupported := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{200: 900})
	supported := addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", map[int32]uint64{100: 100})
	addQueueTestStation(t, database, "33333333-3333-4333-8333-333333333333", map[int32]uint64{100: 0})
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
	keyspaceTask := &models.CrackTask{
		CrackJobID: job.ID,
		Kind:       int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
		State:      int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
	}
	if err := database.Create(keyspaceTask).Error; err != nil {
		t.Fatalf("create keyspace task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: keyspaceTask.ID, HashType: 100, Keyspace: true}).Error; err != nil {
		t.Fatalf("create keyspace command: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule keyspace task: %v", err)
	}
	if err := database.Preload("Command").First(keyspaceTask, "id = ?", keyspaceTask.ID).Error; err != nil {
		t.Fatalf("load leased keyspace task: %v", err)
	}
	if keyspaceTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || keyspaceTask.CrackstationID.String() != supported.HostUUID {
		t.Fatalf("keyspace task leased to %s in state %d, want supported station %s", keyspaceTask.CrackstationID, keyspaceTask.State, supported.HostUUID)
	}
	weights, err := stationWeights(database, 100, "hashcat-test-v1")
	if err != nil {
		t.Fatalf("load station weights: %v", err)
	}
	if !slices.Equal(weights, []weightedCrackstation{{HostUUID: supported.HostUUID, Rate: 100}}) {
		t.Fatalf("selected-mode station weights = %#v", weights)
	}

	completion := keyspaceTask.ToProtobuf()
	completion.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	completion.CompletedAt = time.Now().Unix()
	completion.Keyspace = "100"
	if _, err := updateLeasedCrackTask(completion); err != nil {
		t.Fatalf("complete keyspace task: %v", err)
	}
	var shard models.CrackTask
	if err := database.Preload("Command").First(&shard, "crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Error; err != nil {
		t.Fatalf("load crack shard: %v", err)
	}
	if shard.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || shard.CrackstationID.String() != supported.HostUUID {
		t.Fatalf("crack shard leased to %s in state %d, want supported station %s", shard.CrackstationID, shard.State, supported.HostUUID)
	}

	core.RemoveCrackstation(supported.HostUUID)
	if err := requeueCrackstationTasks(supported.HostUUID); err != nil {
		t.Fatalf("requeue disconnected station: %v", err)
	}
	if err := database.First(&shard, "id = ?", shard.ID).Error; err != nil {
		t.Fatalf("reload requeued shard: %v", err)
	}
	if shard.State != int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED) || shard.CrackstationID != models.NilUUID() {
		t.Fatalf("unsupported fallback leased requeued shard: %#v", shard)
	}
	if runtimeUnsupported := core.GetCrackstation(unsupported.HostUUID); runtimeUnsupported == nil {
		t.Fatal("unsupported station unexpectedly disconnected")
	}
}

//nolint:gocyclo // The test spans disconnect recovery, version pinning, and rescheduled keyspace completion.
func TestKeyspaceCompletionSurvivesDisconnectAndPinsHashcatVersion(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	versionOne := addQueueTestStationVersion(t, database, "11111111-1111-4111-8111-111111111111", "hashcat-v1", map[int32]uint64{100: 100})
	versionTwo := addQueueTestStationVersion(t, database, "22222222-2222-4222-8222-222222222222", "hashcat-v2", map[int32]uint64{100: 1000})
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
	keyspaceTask := &models.CrackTask{CrackJobID: job.ID, Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE), State: int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)}
	if err := database.Create(keyspaceTask).Error; err != nil {
		t.Fatalf("create keyspace task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: keyspaceTask.ID, HashType: 100, Keyspace: true}).Error; err != nil {
		t.Fatalf("create keyspace command: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule keyspace task: %v", err)
	}
	if err := database.Preload("Command").First(keyspaceTask, "id = ?", keyspaceTask.ID).Error; err != nil {
		t.Fatalf("load keyspace task: %v", err)
	}
	if keyspaceTask.CrackstationID.String() != versionOne.HostUUID {
		t.Fatalf("keyspace station = %s, want %s", keyspaceTask.CrackstationID, versionOne.HostUUID)
	}

	// Simulate the registration stream removing the runtime station after the
	// unary completion RPC has been authorized but before its transaction runs.
	core.RemoveCrackstation(versionOne.HostUUID)
	completion := keyspaceTask.ToProtobuf()
	completion.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	completion.CompletedAt = time.Now().Unix()
	completion.Keyspace = "100"
	if _, err := updateLeasedCrackTask(completion); err != nil {
		t.Fatalf("complete keyspace task after runtime disconnect: %v", err)
	}
	var storedJob models.CrackJob
	if err := database.First(&storedJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load pinned job: %v", err)
	}
	if storedJob.HashcatVersion != "hashcat-v1" {
		t.Fatalf("job hashcat version = %q, want hashcat-v1", storedJob.HashcatVersion)
	}
	var shard models.CrackTask
	if err := database.First(&shard, "crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Error; err != nil {
		t.Fatalf("load queued crack shard: %v", err)
	}
	if shard.CrackstationID.String() != versionOne.HostUUID || shard.State != int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED) {
		t.Fatalf("disconnect-race shard = %#v", shard)
	}
	select {
	case event := <-versionTwo.Events:
		t.Fatalf("cross-version station received event %#v", event)
	default:
	}

	replacement := core.NewCrackstation(&clientpb.Crackstation{HostUUID: versionOne.HostUUID, OperatorName: "queue-test", HashcatVersion: "hashcat-v1"})
	if err := core.AddCrackstation(replacement); err != nil {
		t.Fatalf("reconnect matching-version station: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule pinned shard after reconnect: %v", err)
	}
	if err := database.First(&shard, "id = ?", shard.ID).Error; err != nil {
		t.Fatalf("reload scheduled shard: %v", err)
	}
	if shard.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || shard.CrackstationID.String() != versionOne.HostUUID {
		t.Fatalf("reconnected matching version did not receive shard: %#v", shard)
	}
}

func TestExpiredLeaseRequeuesWithFreshAttemptAndClearsTelemetry(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{0: 1})
	job := createQueueTestJob(t, database, models.CrackCommand{})
	task := &models.CrackTask{
		CrackJobID:       job.ID,
		CrackstationID:   models.ParseUUIDOrNil(station.HostUUID),
		Kind:             int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		State:            int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		Attempt:          1,
		LeaseToken:       "old-token",
		LeaseExpiresAt:   time.Now().Add(-time.Minute),
		StartedAt:        time.Now().Add(-time.Hour),
		Err:              "old error",
		Stdout:           []byte("old output"),
		LatestStatusJSON: []byte(`{"old":true}`),
		RecoveredJSON:    []byte(`[]`),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create expired task: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("requeue expired task: %v", err)
	}
	var got models.CrackTask
	if err := database.First(&got, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load re-leased task: %v", err)
	}
	if got.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || got.Attempt != 2 || got.LeaseToken == "" || got.LeaseToken == "old-token" {
		t.Fatalf("re-leased task = %#v", got)
	}
	if !got.StartedAt.IsZero() || got.Err != "" || len(got.Stdout) != 0 || len(got.LatestStatusJSON) != 0 || len(got.RecoveredJSON) != 0 {
		t.Fatalf("stale telemetry survived requeue: %#v", got)
	}
}

//nolint:gocyclo // Parent preservation, weighted ranges, and child commands are one sharding contract.
func TestKeyspaceCompletionCreatesWeightedRangeShardsAndPreservesParent(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	stationA := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
	addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", map[int32]uint64{100: 3})
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}, Skip: 10, Limit: 40})
	task := &models.CrackTask{
		CrackJobID:     job.ID,
		CrackstationID: models.ParseUUIDOrNil(stationA.HostUUID),
		Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
		State:          int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:        1,
		LeaseToken:     "keyspace-token",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create keyspace task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
		t.Fatalf("create keyspace command: %v", err)
	}
	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	request.CompletedAt = time.Now().Unix()
	request.Keyspace = "100"
	if _, err := updateLeasedCrackTask(request); err != nil {
		t.Fatalf("complete keyspace task: %v", err)
	}
	var shards []models.CrackTask
	if err := database.Where("crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Order("shard_skip asc").Find(&shards).Error; err != nil {
		t.Fatalf("load shards: %v", err)
	}
	if len(shards) != 2 || shards[0].ShardSkip != 10 || shards[0].ShardLimit != 10 || shards[1].ShardSkip != 20 || shards[1].ShardLimit != 30 {
		t.Fatalf("shards = %#v, want [10,20) and [20,50)", shards)
	}
	loaded, err := loadCrackJob(database, job.ID)
	if err != nil {
		t.Fatalf("load crack job: %v", err)
	}
	if loaded.Command.Skip != 10 || loaded.Command.Limit != 40 || loaded.Keyspace != "100" {
		t.Fatalf("parent command/range changed: %#v", loaded.Command)
	}
	for _, shard := range shards {
		var command models.CrackCommand
		if err := database.First(&command, "crack_task_id = ?", shard.ID).Error; err != nil {
			t.Fatalf("load shard command: %v", err)
		}
		if command.CrackJobID != models.NilUUID() || command.Skip != shard.ShardSkip || command.Limit != shard.ShardLimit {
			t.Fatalf("shard command = %#v", command)
		}
	}
}

func TestKeyspaceCompletionAcceptsTypedAndLegacyCanonicalResults(t *testing.T) {
	tests := []struct {
		name     string
		keyspace string
		stdout   []byte
	}{
		{name: "typed field", keyspace: "10"},
		{name: "legacy stdout", stdout: []byte("  10\n")},
		{name: "agreeing field and stdout", keyspace: "10", stdout: []byte("10\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := setupCrackstationRPCTestDB(t)
			station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
			job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
			task := &models.CrackTask{
				CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil(station.HostUUID),
				Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				Attempt: 1, LeaseToken: "compatible-keyspace-token", LeaseExpiresAt: time.Now().Add(time.Minute),
			}
			if err := database.Create(task).Error; err != nil {
				t.Fatalf("create keyspace task: %v", err)
			}
			if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
				t.Fatalf("create keyspace command: %v", err)
			}
			request := task.ToProtobuf()
			request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
			request.CompletedAt = time.Now().Unix()
			request.Keyspace = test.keyspace
			request.Stdout = append([]byte(nil), test.stdout...)
			if _, err := updateLeasedCrackTask(request); err != nil {
				t.Fatalf("complete compatible keyspace task: %v", err)
			}

			var storedTask models.CrackTask
			if err := database.First(&storedTask, "id = ?", task.ID).Error; err != nil {
				t.Fatalf("reload keyspace task: %v", err)
			}
			if storedTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_COMPLETED) || storedTask.Keyspace != "10" || storedTask.Err != "" || storedTask.CompletedAt.IsZero() {
				t.Fatalf("stored compatible keyspace task = %#v", storedTask)
			}
			var shardCount int64
			if err := database.Model(&models.CrackTask{}).Where("crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Count(&shardCount).Error; err != nil {
				t.Fatalf("count crack shards: %v", err)
			}
			if shardCount != 1 {
				t.Fatalf("crack shard count = %d, want 1", shardCount)
			}
			var storedJob models.CrackJob
			if err := database.First(&storedJob, "id = ?", job.ID).Error; err != nil {
				t.Fatalf("reload crack job: %v", err)
			}
			if storedJob.Keyspace != "10" || storedJob.Err != "" || !storedJob.CompletedAt.IsZero() {
				t.Fatalf("stored compatible keyspace job = %#v", storedJob)
			}
		})
	}
}

func TestInvalidTerminalKeyspaceResultFailsTaskAndJobWithoutShards(t *testing.T) {
	diagnostic := append([]byte("\x1b[31mOpenCL\\m00000.cl\r\nmissing\x00 "), bytes.Repeat([]byte("x"), maxCrackTaskDiagnosticBytes+256)...)
	tests := []struct {
		name      string
		mutate    func(*clientpb.CrackTask)
		wantError string
	}{
		{
			name: "nonzero exit preserves safe stderr excerpt",
			mutate: func(request *clientpb.CrackTask) {
				request.Stdout = nil
				request.Stderr = diagnostic
				request.ExitCode = -1
			},
			wantError: "hashcat exited with status -1",
		},
		{name: "stdout truncated", mutate: func(request *clientpb.CrackTask) { request.StdoutTruncated = true }, wantError: "output was truncated"},
		{name: "stderr truncated", mutate: func(request *clientpb.CrackTask) { request.StderrTruncated = true }, wantError: "output was truncated"},
		{name: "empty", mutate: func(request *clientpb.CrackTask) { request.Stdout = nil }, wantError: "invalid keyspace result"},
		{name: "noncanonical stdout", mutate: func(request *clientpb.CrackTask) { request.Stdout = []byte("01\n") }, wantError: "invalid keyspace result"},
		{name: "overflowing stdout", mutate: func(request *clientpb.CrackTask) { request.Stdout = []byte("18446744073709551616\n") }, wantError: "invalid keyspace result"},
		{
			name: "noncanonical typed field",
			mutate: func(request *clientpb.CrackTask) {
				request.Keyspace = "01"
				request.Stdout = nil
			},
			wantError: "invalid keyspace result",
		},
		{
			name: "whitespace-padded typed field",
			mutate: func(request *clientpb.CrackTask) {
				request.Keyspace = " 10 "
				request.Stdout = nil
			},
			wantError: "invalid keyspace result",
		},
		{
			name: "typed field and stdout disagree",
			mutate: func(request *clientpb.CrackTask) {
				request.Keyspace = "10"
				request.Stdout = []byte("11\n")
			},
			wantError: "keyspace field and stdout disagree",
		},
		{name: "invalid UTF-8", mutate: func(request *clientpb.CrackTask) { request.Stdout = []byte{0xff} }, wantError: "not valid UTF-8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := setupCrackstationRPCTestDB(t)
			station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
			job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
			task := &models.CrackTask{
				CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil(station.HostUUID),
				Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				Attempt: 1, LeaseToken: "invalid-keyspace-token", LeaseExpiresAt: time.Now().Add(time.Minute),
			}
			if err := database.Create(task).Error; err != nil {
				t.Fatalf("create keyspace task: %v", err)
			}
			if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
				t.Fatalf("create keyspace command: %v", err)
			}
			request := task.ToProtobuf()
			request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
			request.CompletedAt = time.Now().Unix()
			request.Stdout = []byte("10\n")
			test.mutate(request)
			if _, err := updateLeasedCrackTask(request); err != nil {
				t.Fatalf("invalid terminal keyspace result was not acknowledged: %v", err)
			}

			var storedTask models.CrackTask
			if err := database.First(&storedTask, "id = ?", task.ID).Error; err != nil {
				t.Fatalf("reload failed keyspace task: %v", err)
			}
			if storedTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_FAILED) || storedTask.CompletedAt.IsZero() || !storedTask.LeaseExpiresAt.IsZero() || storedTask.Keyspace != "" || !strings.Contains(storedTask.Err, test.wantError) {
				t.Fatalf("stored failed keyspace task = %#v, want error containing %q", storedTask, test.wantError)
			}
			if len(storedTask.Err) > maxCrackTaskDiagnosticBytes+256 {
				t.Fatalf("stored keyspace error is not bounded: %d bytes", len(storedTask.Err))
			}
			if !bytes.Equal(storedTask.Stdout, request.Stdout) || !bytes.Equal(storedTask.Stderr, request.Stderr) {
				t.Fatal("failed keyspace task did not retain its complete bounded process output")
			}
			if test.name == "nonzero exit preserves safe stderr excerpt" {
				if !strings.Contains(storedTask.Err, "OpenCL\\m00000.cl missing") || strings.ContainsAny(storedTask.Err, "\x1b\r\n\x00") || !strings.HasSuffix(storedTask.Err, "...") {
					t.Fatalf("unsafe or unhelpful keyspace diagnostic %q", storedTask.Err)
				}
			}

			var storedJob models.CrackJob
			if err := database.First(&storedJob, "id = ?", job.ID).Error; err != nil {
				t.Fatalf("reload failed crack job: %v", err)
			}
			if storedJob.CompletedAt.IsZero() || storedJob.Err != storedTask.Err || storedJob.Keyspace != "" {
				t.Fatalf("stored failed keyspace job = %#v", storedJob)
			}
			var shardCount int64
			if err := database.Model(&models.CrackTask{}).Where("crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Count(&shardCount).Error; err != nil {
				t.Fatalf("count crack shards: %v", err)
			}
			if shardCount != 0 {
				t.Fatalf("invalid keyspace result created %d crack shards", shardCount)
			}
		})
	}
}

func TestKeyspaceCompletionHandlesEffectiveEmptyRanges(t *testing.T) {
	tests := []struct {
		name          string
		skip          uint64
		limit         uint64
		wantShardSkip uint64
		wantShards    int64
		wantComplete  bool
	}{
		{name: "skip equals keyspace", skip: 100, wantComplete: true},
		{name: "skip exceeds keyspace", skip: 200, wantComplete: true},
		{name: "limited range remains active", skip: 99, limit: 1, wantShardSkip: 99, wantShards: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := setupCrackstationRPCTestDB(t)
			station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
			job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}, Skip: test.skip, Limit: test.limit})
			task := &models.CrackTask{
				CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil(station.HostUUID),
				Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				Attempt: 1, LeaseToken: "keyspace-boundary-token", LeaseExpiresAt: time.Now().Add(time.Minute),
			}
			if err := database.Create(task).Error; err != nil {
				t.Fatalf("create keyspace task: %v", err)
			}
			if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
				t.Fatalf("create keyspace command: %v", err)
			}
			request := task.ToProtobuf()
			request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
			request.CompletedAt = time.Now().Unix()
			request.Keyspace = "100"
			if _, err := updateLeasedCrackTask(request); err != nil {
				t.Fatalf("complete keyspace task: %v", err)
			}
			var shards []models.CrackTask
			if err := database.Where("crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Find(&shards).Error; err != nil {
				t.Fatalf("load crack shards: %v", err)
			}
			if int64(len(shards)) != test.wantShards {
				t.Fatalf("crack shard count = %d, want %d", len(shards), test.wantShards)
			}
			if test.wantShards == 1 && (shards[0].ShardSkip != test.wantShardSkip || shards[0].ShardLimit != 1) {
				t.Fatalf("limited-range shard = %#v", shards[0])
			}
			var gotJob models.CrackJob
			if err := database.First(&gotJob, "id = ?", job.ID).Error; err != nil {
				t.Fatalf("load crack job: %v", err)
			}
			if gotComplete := !gotJob.CompletedAt.IsZero(); gotComplete != test.wantComplete || gotJob.Keyspace != "100" {
				t.Fatalf("crack job completion/keyspace = %#v, want complete=%v", gotJob, test.wantComplete)
			}
		})
	}
}

func TestKeyspaceShardCreationRollsBackPartialRows(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	stationA := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
	addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", map[int32]uint64{100: 1})
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
	task := &models.CrackTask{
		CrackJobID:     job.ID,
		CrackstationID: models.ParseUUIDOrNil(stationA.HostUUID),
		Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
		State:          int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:        1,
		LeaseToken:     "keyspace-token",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create keyspace task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100, Keyspace: true}).Error; err != nil {
		t.Fatalf("create keyspace command: %v", err)
	}
	callbackName := "test:reject-second-crack-shard-command"
	if err := database.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		command, ok := tx.Statement.Dest.(*models.CrackCommand)
		if ok && command.CrackTaskID != models.NilUUID() && command.Skip > 0 {
			_ = tx.AddError(errors.New("injected second shard command failure"))
		}
	}); err != nil {
		t.Fatalf("register create failure callback: %v", err)
	}
	t.Cleanup(func() { _ = database.Callback().Create().Remove(callbackName) })

	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	request.CompletedAt = time.Now().Unix()
	request.Keyspace = "10"
	if _, err := updateLeasedCrackTask(request); err != nil {
		t.Fatalf("complete keyspace task: %v", err)
	}
	var crackTasks int64
	if err := database.Model(&models.CrackTask{}).Where("crack_job_id = ? AND kind = ?", job.ID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Count(&crackTasks).Error; err != nil {
		t.Fatalf("count partial crack tasks: %v", err)
	}
	if crackTasks != 0 {
		t.Fatalf("partial crack tasks persisted after shard failure: %d", crackTasks)
	}
	var gotJob models.CrackJob
	if err := database.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load failed crack job: %v", err)
	}
	if gotJob.CompletedAt.IsZero() || !strings.Contains(gotJob.Err, "injected second shard command failure") {
		t.Fatalf("failed crack job = %#v", gotJob)
	}
}

func TestSchedulerDoesNotLeaseTasksForCompletedJobs(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 1})
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"hash"}})
	if err := database.Model(&models.CrackJob{}).Where("id = ?", job.ID).Update("completed_at", time.Now()).Error; err != nil {
		t.Fatalf("complete crack job: %v", err)
	}
	task := &models.CrackTask{CrackJobID: job.ID, Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create queued task: %v", err)
	}
	if err := database.Create(&models.CrackCommand{CrackTaskID: task.ID, HashType: 100}).Error; err != nil {
		t.Fatalf("create task command: %v", err)
	}
	if err := scheduleCrackTasks(); err != nil {
		t.Fatalf("schedule crack tasks: %v", err)
	}
	if err := database.First(task, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("reload queued task: %v", err)
	}
	if task.State != int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED) || task.Attempt != 0 || task.CrackstationID != models.NilUUID() {
		t.Fatalf("completed-job task was leased: %#v", task)
	}
	select {
	case event := <-station.Events:
		t.Fatalf("completed-job task dispatched event: %#v", event)
	default:
	}
}

//nolint:gocyclo // Binary plaintext ingestion, deduplication, and result counts share one recovery contract.
func TestRecoveredResultsAreIdempotentAndPreserveBinaryPlaintext(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"ABCDEF-UPPERCASE"}})
	credential := &models.Credential{Hash: "ABCDEF-UPPERCASE", HashType: 100}
	if err := database.Create(credential).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	recoveredPlaintext := []byte{'a', 0, 'b'}
	recovered, err := json.Marshal([]recoveredCrackResult{{Hash: credential.Hash, Plaintext: recoveredPlaintext}})
	if err != nil {
		t.Fatalf("marshal recovered result: %v", err)
	}
	hostID := models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111")
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: hostID,
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 1, LeaseToken: "result-token", LeaseExpiresAt: time.Now().Add(time.Minute), RecoveredJSON: recovered,
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create result task: %v", err)
	}
	var firstEvents, secondEvents []string
	brokerEvents := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(brokerEvents)
	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	request.CompletedAt = time.Now().Unix()
	request.RecoveredJSON = recovered
	firstEvents, err = updateLeasedCrackTask(request)
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	foundCredentialEvent := false
	deadline := time.After(time.Second)
	for !foundCredentialEvent {
		select {
		case event := <-brokerEvents:
			if event.EventType == consts.CredentialCrackedEvent && string(event.Data) == credential.ID.String() {
				foundCredentialEvent = true
			}
		case <-deadline:
			t.Fatal("credential-cracked event was not published")
		}
	}
	if err := database.Transaction(func(tx *gorm.DB) error {
		var ingestErr error
		secondEvents, ingestErr = ingestRecoveredResults(tx, task, time.Now())
		return ingestErr
	}); err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if !slices.Equal(firstEvents, []string{credential.ID.String()}) || len(secondEvents) != 0 {
		t.Fatalf("credential transitions = %#v then %#v", firstEvents, secondEvents)
	}
	var result models.CrackResult
	if err := database.First(&result).Error; err != nil {
		t.Fatalf("load crack result: %v", err)
	}
	if !bytes.Equal(result.Plaintext, recoveredPlaintext) {
		t.Fatalf("result plaintext = %x", result.Plaintext)
	}
	var gotCredential models.Credential
	if err := database.First(&gotCredential, "id = ?", credential.ID).Error; err != nil {
		t.Fatalf("load credential: %v", err)
	}
	if !gotCredential.IsCracked || gotCredential.Plaintext != "$HEX[610062]" {
		t.Fatalf("updated credential = %#v", gotCredential)
	}
	var count int64
	if err := database.Model(&models.CrackResult{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("result count = %d, err=%v", count, err)
	}
	conflicting, err := json.Marshal([]recoveredCrackResult{{Hash: credential.Hash, Plaintext: []byte("different-plaintext")}})
	if err != nil {
		t.Fatal(err)
	}
	task.RecoveredJSON = conflicting
	if err := database.Transaction(func(tx *gorm.DB) error {
		_, ingestErr := ingestRecoveredResults(tx, task, time.Now())
		return ingestErr
	}); err == nil || !strings.Contains(err.Error(), "conflicting plaintext") {
		t.Fatalf("conflicting recovery error = %v", err)
	}
	if err := database.Model(&models.CrackResult{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("conflicting recovery grew result count to %d, err=%v", count, err)
	}
	task.RecoveredJSON = []byte(`[{"hash":"not-submitted","plaintext":"cHdu"}]`)
	if err := database.Transaction(func(tx *gorm.DB) error {
		_, ingestErr := ingestRecoveredResults(tx, task, time.Now())
		return ingestErr
	}); err == nil {
		t.Fatal("unsubmitted recovered hash was accepted")
	}
}

func TestRunningRecoveryBatchHonorsCumulativeJobBudget(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"budget-hash"}})
	if err := database.Model(&models.CrackJob{}).Where("id = ?", job.ID).Update("recovered_bytes", uint64(maxCrackRecoveredTotalBytes-1)).Error; err != nil {
		t.Fatal(err)
	}
	credential := &models.Credential{Hash: "budget-hash", HashType: 100}
	if err := database.Create(credential).Error; err != nil {
		t.Fatal(err)
	}
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111"),
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 1, LeaseToken: "budget-token", LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	batch, err := json.Marshal([]recoveredCrackResult{{Hash: credential.Hash, Plaintext: []byte("plaintext")}})
	if err != nil {
		t.Fatal(err)
	}
	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_RUNNING
	request.StartedAt = time.Now().Unix()
	request.RecoveredJSON = batch
	if _, err := updateLeasedCrackTask(request); err == nil || !strings.Contains(err.Error(), "cumulative size limit") {
		t.Fatalf("cumulative recovery budget error = %v", err)
	}
	var resultCount int64
	if err := database.Model(&models.CrackResult{}).Count(&resultCount).Error; err != nil || resultCount != 0 {
		t.Fatalf("over-budget batch persisted %d results, err=%v", resultCount, err)
	}
	var gotCredential models.Credential
	if err := database.First(&gotCredential, "id = ?", credential.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotCredential.IsCracked {
		t.Fatalf("over-budget batch updated credential: %#v", gotCredential)
	}
}

//nolint:gocyclo // Recovery-batch ordering and the final empty terminal update form one transaction.
func TestRunningTaskRecoveryBatchesAreIngestedBeforeEmptyTerminalUpdate(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hashes := []string{"batched-hash-one", "batched-hash-two"}
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: hashes})
	credentials := []*models.Credential{
		{Hash: hashes[0], HashType: 100},
		{Hash: hashes[1], HashType: 100},
	}
	for _, credential := range credentials {
		if err := database.Create(credential).Error; err != nil {
			t.Fatalf("create credential: %v", err)
		}
	}
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111"),
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 1, LeaseToken: "batched-result-token", LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create batched result task: %v", err)
	}
	batches := [][]recoveredCrackResult{
		{{Hash: hashes[0], Plaintext: []byte("plain-one")}},
		{{Hash: hashes[1], Plaintext: []byte("plain-two")}},
	}
	var lastBatch []byte
	for index, batch := range batches {
		encoded, err := json.Marshal(batch)
		if err != nil {
			t.Fatal(err)
		}
		lastBatch = encoded
		request := task.ToProtobuf()
		request.State = clientpb.CrackTaskState_CRACK_TASK_RUNNING
		request.StartedAt = time.Now().Unix()
		request.RecoveredJSON = encoded
		crackedIDs, err := updateLeasedCrackTask(request)
		if err != nil {
			t.Fatalf("ingest running batch %d: %v", index, err)
		}
		if !slices.Equal(crackedIDs, []string{credentials[index].ID.String()}) {
			t.Fatalf("batch %d cracked IDs = %#v", index, crackedIDs)
		}
	}
	terminal := task.ToProtobuf()
	terminal.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	terminal.StartedAt = time.Now().Unix()
	terminal.CompletedAt = time.Now().Unix()
	terminal.RecoveredJSON = nil
	if crackedIDs, err := updateLeasedCrackTask(terminal); err != nil {
		t.Fatalf("empty terminal update: %v", err)
	} else if len(crackedIDs) != 0 {
		t.Fatalf("empty terminal update re-ingested credentials: %#v", crackedIDs)
	}
	var resultCount int64
	if err := database.Model(&models.CrackResult{}).Where("crack_task_id = ?", task.ID).Count(&resultCount).Error; err != nil || resultCount != 2 {
		t.Fatalf("batched result count = %d, err=%v", resultCount, err)
	}
	for index, credential := range credentials {
		var got models.Credential
		if err := database.First(&got, "id = ?", credential.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !got.IsCracked || got.Plaintext != fmt.Sprintf("plain-%s", []string{"one", "two"}[index]) {
			t.Fatalf("credential %d = %#v", index, got)
		}
	}
	var gotTask models.CrackTask
	if err := database.First(&gotTask, "id = ?", task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_COMPLETED) || !bytes.Equal(gotTask.RecoveredJSON, lastBatch) {
		t.Fatalf("terminal batched task = %#v; want completed with last diagnostic batch", gotTask)
	}
}

//nolint:gocyclo // Recovered-result ingestion and failure-state retention must be asserted atomically.
func TestFailedCrackTaskIngestsRecoveredResultsAndPreservesFailure(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"failed-task-hash"}})
	credential := &models.Credential{Hash: "failed-task-hash", HashType: 100}
	if err := database.Create(credential).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	recovered, err := json.Marshal([]recoveredCrackResult{{Hash: credential.Hash, Plaintext: []byte("recovered-before-failure")}})
	if err != nil {
		t.Fatalf("marshal recovered result: %v", err)
	}
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111"),
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 1, LeaseToken: "failed-result-token", LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create failed result task: %v", err)
	}
	brokerEvents := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(brokerEvents)
	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_FAILED
	request.CompletedAt = time.Now().Unix()
	request.Err = "hashcat failed after reporting a recovery"
	request.RecoveredJSON = recovered
	credentialIDs, err := updateLeasedCrackTask(request)
	if err != nil {
		t.Fatalf("complete failed task with recovery: %v", err)
	}
	if !slices.Equal(credentialIDs, []string{credential.ID.String()}) {
		t.Fatalf("cracked credential IDs = %#v", credentialIDs)
	}
	foundCredentialEvent := false
	deadline := time.After(time.Second)
	for !foundCredentialEvent {
		select {
		case event := <-brokerEvents:
			if event.EventType == consts.CredentialCrackedEvent && string(event.Data) == credential.ID.String() {
				foundCredentialEvent = true
			}
		case <-deadline:
			t.Fatal("credential-cracked event was not published for failed task recovery")
		}
	}
	var gotTask models.CrackTask
	if err := database.First(&gotTask, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load failed task: %v", err)
	}
	if gotTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_FAILED) || gotTask.Err != request.Err || gotTask.CompletedAt.IsZero() {
		t.Fatalf("failed task state was not preserved: %#v", gotTask)
	}
	var gotJob models.CrackJob
	if err := database.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("load failed job: %v", err)
	}
	if gotJob.Err != request.Err || gotJob.CompletedAt.IsZero() {
		t.Fatalf("failed job state was not preserved: %#v", gotJob)
	}
	var result models.CrackResult
	if err := database.First(&result, "crack_task_id = ?", task.ID).Error; err != nil {
		t.Fatalf("load failed-task crack result: %v", err)
	}
	if result.CredentialID != credential.ID || !bytes.Equal(result.Plaintext, []byte("recovered-before-failure")) {
		t.Fatalf("failed-task crack result = %#v", result)
	}
	var gotCredential models.Credential
	if err := database.First(&gotCredential, "id = ?", credential.ID).Error; err != nil {
		t.Fatalf("load cracked credential: %v", err)
	}
	if !gotCredential.IsCracked || gotCredential.Plaintext != "recovered-before-failure" {
		t.Fatalf("failed-task credential update = %#v", gotCredential)
	}
}

//nolint:gocyclo // Malformed recovery must roll back both result ingestion and terminal-state changes.
func TestMalformedFailedTaskRecoveryRollsBackTerminalUpdate(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: 100, Hashes: []string{"rollback-hash"}})
	credential := &models.Credential{Hash: "rollback-hash", HashType: 100}
	if err := database.Create(credential).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111"),
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 1, LeaseToken: "malformed-result-token", LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create malformed result task: %v", err)
	}
	request := task.ToProtobuf()
	request.State = clientpb.CrackTaskState_CRACK_TASK_FAILED
	request.CompletedAt = time.Now().Unix()
	request.Err = "hashcat failed"
	request.RecoveredJSON = []byte(`[{`)
	if _, err := updateLeasedCrackTask(request); err == nil || !strings.Contains(err.Error(), "invalid recovered results") {
		t.Fatalf("malformed failed-task recovery error = %v", err)
	}
	var gotTask models.CrackTask
	if err := database.First(&gotTask, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("reload task after rollback: %v", err)
	}
	if gotTask.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || !gotTask.CompletedAt.IsZero() || gotTask.Err != "" || len(gotTask.RecoveredJSON) != 0 {
		t.Fatalf("malformed recovery persisted terminal task state: %#v", gotTask)
	}
	var gotJob models.CrackJob
	if err := database.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("reload job after rollback: %v", err)
	}
	if !gotJob.CompletedAt.IsZero() || gotJob.Err != "" {
		t.Fatalf("malformed recovery persisted job failure: %#v", gotJob)
	}
	var resultCount int64
	if err := database.Model(&models.CrackResult{}).Count(&resultCount).Error; err != nil {
		t.Fatalf("count rolled-back results: %v", err)
	}
	if resultCount != 0 {
		t.Fatalf("malformed recovery persisted %d crack results", resultCount)
	}
	var gotCredential models.Credential
	if err := database.First(&gotCredential, "id = ?", credential.ID).Error; err != nil {
		t.Fatalf("reload credential after rollback: %v", err)
	}
	if gotCredential.IsCracked || gotCredential.Plaintext != "" {
		t.Fatalf("malformed recovery updated credential: %#v", gotCredential)
	}
}

//nolint:gocyclo // Size limits, disallowed keyspace fields, and rollback share one atomic update boundary.
func TestCrackTaskUpdateRejectsOversizedAndDisallowedKeyspaceAtomically(t *testing.T) {
	legitimateWorkerPayload := &clientpb.CrackTask{
		Stdout:        make([]byte, maxCrackOutputBytes),
		Stderr:        make([]byte, maxCrackOutputBytes),
		RecoveredJSON: make([]byte, 8<<20),
	}
	if size := proto.Size(legitimateWorkerPayload); size > maxCrackTaskUpdateBytes {
		t.Fatalf("legitimate maximum worker payload size = %d, cap = %d", size, maxCrackTaskUpdateBytes)
	}

	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{})
	hostID := models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111")
	tests := []struct {
		name    string
		kind    clientpb.CrackTaskKind
		mutate  func(*clientpb.CrackTask)
		wantErr string
	}{
		{
			name: "total protobuf size", kind: clientpb.CrackTaskKind_CRACK_TASK_CRACK,
			mutate:  func(request *clientpb.CrackTask) { request.Keyspace = strings.Repeat("1", maxCrackTaskUpdateBytes+1) },
			wantErr: "crack task update exceeds size limit",
		},
		{
			name: "error size", kind: clientpb.CrackTaskKind_CRACK_TASK_CRACK,
			mutate: func(request *clientpb.CrackTask) {
				request.CompletedAt = time.Now().Unix()
				request.Err = strings.Repeat("e", maxCrackTaskErrorBytes+1)
			},
			wantErr: "crack task error exceeds size limit",
		},
		{
			name: "crack task keyspace", kind: clientpb.CrackTaskKind_CRACK_TASK_CRACK,
			mutate: func(request *clientpb.CrackTask) {
				request.CompletedAt = time.Now().Unix()
				request.Keyspace = "10"
			},
			wantErr: "only valid on a successfully completed keyspace task",
		},
		{
			name: "running keyspace", kind: clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE,
			mutate: func(request *clientpb.CrackTask) {
				request.StartedAt = time.Now().Unix()
				request.Keyspace = "10"
			},
			wantErr: "only valid on a successfully completed keyspace task",
		},
		{
			name: "failed keyspace", kind: clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE,
			mutate: func(request *clientpb.CrackTask) {
				request.CompletedAt = time.Now().Unix()
				request.Err = "hashcat failed"
				request.Keyspace = "10"
			},
			wantErr: "only valid on a successfully completed keyspace task",
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := &models.CrackTask{
				CrackJobID: job.ID, CrackstationID: hostID, Kind: int32(test.kind),
				State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), Attempt: 1,
				LeaseToken: fmt.Sprintf("validation-token-%d", index), LeaseExpiresAt: time.Now().Add(time.Minute),
			}
			if err := database.Create(task).Error; err != nil {
				t.Fatalf("create task: %v", err)
			}
			request := task.ToProtobuf()
			test.mutate(request)
			if _, err := updateLeasedCrackTask(request); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("update error = %v, want %q", err, test.wantErr)
			}
			var got models.CrackTask
			if err := database.First(&got, "id = ?", task.ID).Error; err != nil {
				t.Fatalf("reload rejected task: %v", err)
			}
			if got.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) || got.Err != "" || got.Keyspace != "" || !got.StartedAt.IsZero() || !got.CompletedAt.IsZero() || len(got.Stdout) != 0 || len(got.Stderr) != 0 {
				t.Fatalf("rejected update changed task: %#v", got)
			}
		})
	}
	var gotJob models.CrackJob
	if err := database.First(&gotJob, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("reload job after rejected updates: %v", err)
	}
	if gotJob.Err != "" || !gotJob.CompletedAt.IsZero() {
		t.Fatalf("rejected update changed job: %#v", gotJob)
	}
}

//nolint:gocyclo // Lease renewal, heartbeat persistence, and stale-token rejection form one lifecycle.
func TestCrackTaskStatusHeartbeatRenewsLeaseAndRejectsStaleToken(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{})
	hostID := models.ParseUUIDOrNil("11111111-1111-4111-8111-111111111111")
	task := &models.CrackTask{
		CrackJobID: job.ID, CrackstationID: hostID,
		Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt: 3, LeaseToken: "status-token", LeaseExpiresAt: time.Now().Add(time.Second),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	before := time.Now()
	event := crackTaskStatusEvent{
		TaskID: task.ID.String(), HostUUID: hostID.String(), Attempt: task.Attempt,
		LeaseToken: task.LeaseToken, ObservedAt: time.Now().Add(-time.Hour).Unix(), Status: json.RawMessage(`{"progress":[1,10],"temperature":65}`),
	}
	brokerEvents := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(brokerEvents)
	if err := updateCrackTaskStatus(event); err != nil {
		t.Fatalf("update status: %v", err)
	}
	select {
	case published := <-brokerEvents:
		if published.EventType != consts.CrackTaskStatus {
			t.Fatalf("published event type = %q", published.EventType)
		}
		var envelope crackTaskStatusEvent
		if err := json.Unmarshal(published.Data, &envelope); err != nil {
			t.Fatalf("decode published status: %v", err)
		}
		if envelope.LeaseToken != "" || envelope.TaskID != task.ID.String() || !bytes.Equal(envelope.Status, event.Status) {
			t.Fatalf("published status envelope = %#v", envelope)
		}
	case <-time.After(time.Second):
		t.Fatal("crack task status event was not published")
	}
	var got models.CrackTask
	if err := database.First(&got, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if got.State != int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING) || !bytes.Equal(got.LatestStatusJSON, event.Status) || got.LeaseExpiresAt.Before(before.Add(crackTaskLeaseDuration-time.Second)) || got.LastHeartbeatAt.Before(before) {
		t.Fatalf("heartbeat task = %#v", got)
	}
	event.LeaseToken = "stale-token"
	if err := updateCrackTaskStatus(event); err != errStaleCrackTaskAttempt {
		t.Fatalf("stale heartbeat error = %v", err)
	}
}

//nolint:gocyclo // Credential filtering, hash modes, and managed file resolution are one selection contract.
func TestCredentialSelectionUsesHashModeAndManagedFileResolution(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hashMode := uint32(100)
	credential := &models.Credential{Hash: "sha1-hash", HashType: 100, Collection: "selected"}
	if err := database.Create(credential).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	command := &models.CrackCommand{HashType: 0, HashMode: &hashMode, CredentialIDs: []string{credential.ID.String()[:8]}, Hashes: []string{"raw"}}
	credentials, err := selectCrackCredentials(database, command)
	if err != nil {
		t.Fatalf("select credentials: %v", err)
	}
	if len(credentials) != 1 || !slices.Equal(command.Hashes, []string{"raw", "sha1-hash"}) {
		t.Fatalf("selection = %#v, hashes=%#v", credentials, command.Hashes)
	}
	md5Credential := &models.Credential{Hash: "md5-hash", HashType: int32(clientpb.HashType_MD5)}
	if err := database.Create(md5Credential).Error; err != nil {
		t.Fatalf("create MD5 credential: %v", err)
	}
	defaultCommand := &models.CrackCommand{
		HashType:      int32(clientpb.HashType_INVALID),
		CredentialIDs: []string{md5Credential.ID.String()},
	}
	defaultCredentials, err := selectCrackCredentials(database, defaultCommand)
	if err != nil {
		t.Fatalf("select default MD5 credential: %v", err)
	}
	if effectiveCrackHashType(defaultCommand) != int32(clientpb.HashType_MD5) || len(defaultCredentials) != 1 || !slices.Equal(defaultCommand.Hashes, []string{"md5-hash"}) {
		t.Fatalf("default MD5 selection = %#v, command=%#v", defaultCredentials, defaultCommand)
	}
	files := []models.CrackFile{
		{Name: "words.txt", Type: int32(clientpb.CrackFileType_WORDLIST), Sha2_256: "AAAA", IsComplete: true},
		{Name: "rules.rule", Type: int32(clientpb.CrackFileType_RULES), Sha2_256: "BBBB", IsComplete: true},
		{Name: "markov.hcstat2", Type: int32(clientpb.CrackFileType_MARKOV_HCSTAT2), Sha2_256: "CCCC", IsComplete: true},
	}
	for index := range files {
		if err := database.Create(&files[index]).Error; err != nil {
			t.Fatalf("create managed file: %v", err)
		}
	}
	fileCommand := &models.CrackCommand{
		PositionalArguments: []string{"?d?d", "words.txt"},
		RulesFile:           []byte("rules.rule"),
		MarkovHcstat2:       []byte("markov.hcstat2"),
	}
	if err := resolveManagedCrackFiles(database, fileCommand); err != nil {
		t.Fatalf("resolve managed files: %v", err)
	}
	if fileCommand.PositionalArguments[0] != "?d?d" || fileCommand.PositionalArguments[1] != "crackfile://wordlist/aaaa" || string(fileCommand.RulesFile) != "crackfile://rules/bbbb" || string(fileCommand.MarkovHcstat2) != "crackfile://hcstat2/cccc" {
		t.Fatalf("resolved command = %#v", fileCommand)
	}
	if err := database.Create(&models.CrackFile{Name: "words.txt", Type: int32(clientpb.CrackFileType_RULES), Sha2_256: "DDDD", IsComplete: true}).Error; err != nil {
		t.Fatalf("create ambiguous file: %v", err)
	}
	typedCommand := &models.CrackCommand{PositionalArguments: []string{"rules.rule", "words.txt"}, RulesFile: []byte("words.txt")}
	if err := resolveManagedCrackFiles(database, typedCommand); err != nil {
		t.Fatalf("resolve typed same-name files: %v", err)
	}
	if typedCommand.PositionalArguments[0] != "rules.rule" || typedCommand.PositionalArguments[1] != "crackfile://wordlist/aaaa" || string(typedCommand.RulesFile) != "crackfile://rules/dddd" {
		t.Fatalf("typed managed resolution = %#v", typedCommand)
	}
	if err := resolveManagedCrackFiles(database, &models.CrackCommand{RulesFile: []byte("crackfile://wordlist/" + strings.Repeat("a", 64))}); err == nil {
		t.Fatal("wrong-type managed URI was accepted")
	}
	if err := resolveManagedCrackFiles(database, &models.CrackCommand{PositionalArguments: []string{"crackfile://wordlist/not-a-sha"}}); err == nil {
		t.Fatal("malformed managed URI was accepted")
	}
}

func TestCrackRejectsEmptyEffectiveHashSelection(t *testing.T) {
	setupCrackstationRPCTestDB(t)
	rpcServer := &Server{}
	tests := []*clientpb.CrackCommand{
		{},
		{CredentialCollection: "missing"},
		{CredentialIDs: []string{""}},
		{Hashes: []string{"   "}},
		{Hashes: []string{"valid", "injected\nhash"}},
		{Hashes: []string{"hash\x00suffix"}},
	}
	for _, request := range tests {
		if _, err := rpcServer.Crack(t.Context(), request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Crack(%#v) error = %v, want InvalidArgument", request, err)
		}
	}
}

func TestDistributedCrackOperandValidation(t *testing.T) {
	wordlist := "crackfile://wordlist/" + strings.Repeat("a", 64)
	valid := []models.CrackCommand{
		{AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT), PositionalArguments: []string{wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT), PositionalArguments: []string{wordlist, wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_COMBINATION), PositionalArguments: []string{wordlist, wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), PositionalArguments: []string{"?d?d"}},
		{AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), PositionalArguments: []string{wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK), PositionalArguments: []string{wordlist, "?d"}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST), PositionalArguments: []string{"?d", wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT), Identify: wordlist},
	}
	for index := range valid {
		if err := validateDistributedCrackOperands(&valid[index]); err != nil {
			t.Fatalf("valid command %d rejected: %v", index, err)
		}
	}
	invalid := []models.CrackCommand{
		{AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT)},
		{AttackMode: int32(clientpb.CrackAttackMode_STRAIGHT), PositionalArguments: []string{"station-local.txt"}},
		{AttackMode: int32(clientpb.CrackAttackMode_COMBINATION), PositionalArguments: []string{wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_COMBINATION), PositionalArguments: []string{wordlist, "station-local.txt"}},
		{AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE)},
		{AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), PositionalArguments: []string{"?d", "?l"}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK), PositionalArguments: []string{wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK), PositionalArguments: []string{wordlist, wordlist, "?d"}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK), PositionalArguments: []string{"station-local.txt", "?d"}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST), PositionalArguments: []string{"?d"}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST), PositionalArguments: []string{"?d", wordlist, wordlist}},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST), PositionalArguments: []string{"?d", "station-local.txt"}},
		{AttackMode: int32(clientpb.CrackAttackMode_PCFG)},
		{AttackMode: int32(clientpb.CrackAttackMode_GENERIC)},
		{AttackMode: int32(clientpb.CrackAttackMode_ASSOCIATION)},
		{AttackMode: int32(clientpb.CrackAttackMode_NO_ATTACK)},
		{AttackMode: int32(clientpb.CrackAttackMode_HYBRID)},
		{AttackMode: 99},
	}
	for index := range invalid {
		if err := validateDistributedCrackOperands(&invalid[index]); err == nil {
			t.Fatalf("invalid command %d accepted: %#v", index, invalid[index])
		}
	}
}

func TestCrackRejectsUnmanagedWordlistsWithoutPersisting(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	rpcServer := &Server{}
	tests := []*clientpb.CrackCommand{
		{AttackMode: clientpb.CrackAttackMode_STRAIGHT, Hashes: []string{"hash"}, PositionalArguments: []string{"station-local.txt"}},
		{AttackMode: clientpb.CrackAttackMode_COMBINATION, Hashes: []string{"hash"}, PositionalArguments: []string{"left.txt", "right.txt"}},
		{AttackMode: clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK, Hashes: []string{"hash"}, PositionalArguments: []string{"words.txt", "?d"}},
		{AttackMode: clientpb.CrackAttackMode_HYBRID_MASK_WORDLIST, Hashes: []string{"hash"}, PositionalArguments: []string{"?d", "words.txt"}},
		{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"}, RulesFile: []byte("inline-rule")},
		{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"}, RulesFilesV7: [][]byte{[]byte("local.rule")}},
		{AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"}, MarkovHcstat2: []byte("local.hcstat2")},
	}
	for _, request := range tests {
		if _, err := rpcServer.Crack(t.Context(), request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Crack(%#v) error = %v, want InvalidArgument", request, err)
		}
	}
	var jobs, tasks int64
	if err := database.Model(&models.CrackJob{}).Count(&jobs).Error; err != nil {
		t.Fatalf("count crack jobs: %v", err)
	}
	if err := database.Model(&models.CrackTask{}).Count(&tasks).Error; err != nil {
		t.Fatalf("count crack tasks: %v", err)
	}
	if jobs != 0 || tasks != 0 {
		t.Fatalf("unmanaged commands persisted jobs=%d tasks=%d", jobs, tasks)
	}
}

//nolint:gocyclo // The test follows managed wordlist identity through failure, requeue, and reassignment.
func TestManagedWordlistPersistsAcrossTaskFailover(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{100: 100})
	file := &models.CrackFile{
		Name:       "words.txt",
		Type:       int32(clientpb.CrackFileType_WORDLIST),
		Sha2_256:   strings.Repeat("a", 64),
		IsComplete: true,
	}
	if err := database.Create(file).Error; err != nil {
		t.Fatalf("create managed wordlist: %v", err)
	}
	rules := &models.CrackFile{Name: "managed.rule", Type: int32(clientpb.CrackFileType_RULES), Sha2_256: strings.Repeat("b", 64), IsComplete: true}
	if err := database.Create(rules).Error; err != nil {
		t.Fatalf("create managed rules: %v", err)
	}
	markov := &models.CrackFile{Name: "managed.hcstat2", Type: int32(clientpb.CrackFileType_MARKOV_HCSTAT2), Sha2_256: strings.Repeat("c", 64), IsComplete: true}
	if err := database.Create(markov).Error; err != nil {
		t.Fatalf("create managed markov file: %v", err)
	}
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	response, err := (&Server{}).Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode:          clientpb.CrackAttackMode_STRAIGHT,
		HashMode:            func() *uint32 { value := uint32(100); return &value }(),
		Hashes:              []string{"hash"},
		PositionalArguments: []string{file.Name},
		RulesFile:           []byte(rules.Name),
		RulesFilesV7:        [][]byte{[]byte(rules.ID.String())},
		MarkovHcstat2:       []byte(markov.Sha2_256),
	})
	if err != nil {
		t.Fatalf("queue managed-wordlist job: %v", err)
	}
	jobID := models.ParseUUIDOrNil(response.Job.ID)
	uri := managedCrackFileURI(file)
	rulesURI := managedCrackFileURI(rules)
	markovURI := managedCrackFileURI(markov)
	var parent models.CrackCommand
	if err := database.First(&parent, "crack_job_id = ?", jobID).Error; err != nil {
		t.Fatalf("load parent command: %v", err)
	}
	if !slices.Equal(parent.PositionalArguments, []string{uri}) || len(parent.RulesFile) != 0 || len(parent.RulesFilesV7) != 1 || string(parent.RulesFilesV7[0]) != rulesURI || string(parent.MarkovHcstat2) != markovURI {
		t.Fatalf("parent managed files = %#v", parent)
	}
	var keyspaceTask models.CrackTask
	if err := database.Preload("Command").First(&keyspaceTask, "crack_job_id = ? AND kind = ?", jobID, int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE)).Error; err != nil {
		t.Fatalf("load keyspace task: %v", err)
	}
	if !slices.Equal(keyspaceTask.Command.PositionalArguments, []string{uri}) || len(keyspaceTask.Command.RulesFile) != 0 || len(keyspaceTask.Command.RulesFilesV7) != 1 || string(keyspaceTask.Command.RulesFilesV7[0]) != rulesURI || string(keyspaceTask.Command.MarkovHcstat2) != markovURI {
		t.Fatalf("keyspace managed files = %#v", keyspaceTask.Command)
	}
	completion := keyspaceTask.ToProtobuf()
	completion.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	completion.CompletedAt = time.Now().Unix()
	completion.Keyspace = "10"
	if _, err := updateLeasedCrackTask(completion); err != nil {
		t.Fatalf("complete keyspace task: %v", err)
	}
	var shard models.CrackTask
	if err := database.Preload("Command").First(&shard, "crack_job_id = ? AND kind = ?", jobID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Error; err != nil {
		t.Fatalf("load crack shard: %v", err)
	}
	if !slices.Equal(shard.Command.PositionalArguments, []string{uri}) || len(shard.Command.RulesFile) != 0 || len(shard.Command.RulesFilesV7) != 1 || string(shard.Command.RulesFilesV7[0]) != rulesURI || string(shard.Command.MarkovHcstat2) != markovURI {
		t.Fatalf("shard managed files = %#v", shard.Command)
	}
	core.RemoveCrackstation(station.HostUUID)
	if err := requeueCrackstationTasks(station.HostUUID); err != nil {
		t.Fatalf("requeue station tasks: %v", err)
	}
	if err := database.Preload("Command").First(&shard, "id = ?", shard.ID).Error; err != nil {
		t.Fatalf("reload requeued shard: %v", err)
	}
	if shard.State != int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED) || shard.CrackstationID != models.NilUUID() || !slices.Equal(shard.Command.PositionalArguments, []string{uri}) {
		t.Fatalf("requeued managed shard = %#v", shard)
	}
}

func TestManagedWordlistAliasDoesNotRewriteMaskOperand(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	for index := 0; index < 2; index++ {
		file := &models.CrackFile{
			Name: "?d?d", Type: int32(clientpb.CrackFileType_WORDLIST),
			Sha2_256: strings.Repeat(string(rune('a'+index)), 64), IsComplete: true,
		}
		if err := database.Create(file).Error; err != nil {
			t.Fatalf("create colliding managed wordlist %d: %v", index, err)
		}
	}
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	rpcServer := &Server{}
	response, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d?d"},
	})
	if err != nil {
		t.Fatalf("queue mask colliding with managed aliases: %v", err)
	}
	var command models.CrackCommand
	if err := database.First(&command, "crack_job_id = ?", models.ParseUUIDOrNil(response.Job.ID)).Error; err != nil {
		t.Fatalf("load parent command: %v", err)
	}
	if !slices.Equal(command.PositionalArguments, []string{"?d?d"}) {
		t.Fatalf("mask alias was rewritten: %#v", command.PositionalArguments)
	}

	managedMask := &models.CrackFile{
		Name: "mask.hcmask", Type: int32(clientpb.CrackFileType_WORDLIST),
		Sha2_256: strings.Repeat("c", 64), IsComplete: true,
	}
	if err := database.Create(managedMask).Error; err != nil {
		t.Fatalf("create managed hcmask: %v", err)
	}
	uri := managedCrackFileURI(managedMask)
	response, err = rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash-two"}, PositionalArguments: []string{uri},
	})
	if err != nil {
		t.Fatalf("queue canonical managed hcmask: %v", err)
	}
	command = models.CrackCommand{}
	if err := database.First(&command, "crack_job_id = ?", models.ParseUUIDOrNil(response.Job.ID)).Error; err != nil {
		t.Fatalf("load managed hcmask parent command: %v", err)
	}
	if !slices.Equal(command.PositionalArguments, []string{uri}) {
		t.Fatalf("managed hcmask reference changed: %#v", command.PositionalArguments)
	}
}

func TestDistributedCommandNormalizesHashcatFieldPrecedence(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	rules := &models.CrackFile{Name: "preferred.rule", Type: int32(clientpb.CrackFileType_RULES), Sha2_256: strings.Repeat("d", 64), IsComplete: true}
	if err := database.Create(rules).Error; err != nil {
		t.Fatalf("create managed rules file: %v", err)
	}
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	seed := uint32(7)
	disabledOutfileCheck := uint32(0)
	response, err := (&Server{}).Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"},
		Identify:     "crackfile://not-a-valid-ignored-reference", //nolint:staticcheck // Verify canonical inputs override the deprecated field.
		RulesFile:    []byte("ignored-local.rule"),
		RulesFilesV7: [][]byte{[]byte(managedCrackFileURI(rules))}, GenerateRules: 1, GenerateRulesSeed: -1, GenerateRulesSeedV7: &seed,
		OutfileCheckTimer: 5, OutfileCheckTimerV7: &disabledOutfileCheck,
	})
	if err != nil {
		t.Fatalf("queue command with ignored legacy fields: %v", err)
	}
	var command models.CrackCommand
	if err := database.First(&command, "crack_job_id = ?", models.ParseUUIDOrNil(response.Job.ID)).Error; err != nil {
		t.Fatalf("load normalized parent command: %v", err)
	}
	if command.Identify != "" || len(command.RulesFile) != 0 || command.GenerateRulesSeed != 0 || command.GenerateRulesSeedV7 == nil || *command.GenerateRulesSeedV7 != seed ||
		command.OutfileCheckTimer != 0 || command.OutfileCheckTimerV7 == nil || *command.OutfileCheckTimerV7 != 0 {
		t.Fatalf("precedence fields were not normalized: %#v", command)
	}
	if len(command.RulesFilesV7) != 1 || string(command.RulesFilesV7[0]) != managedCrackFileURI(rules) {
		t.Fatalf("effective v7 rules were not preserved: %#v", command.RulesFilesV7)
	}
}

//nolint:gocyclo // The table covers all terminating-mode rejections and internal normalization invariants.
func TestCrackRejectsTerminatingModesAndNormalizesInternalCommands(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	rpcServer := &Server{}
	tests := []struct {
		name string
		set  func(*clientpb.CrackCommand)
	}{
		{name: "hash-mode-overflow", set: func(command *clientpb.CrackCommand) { value := uint32(math.MaxInt32) + 1; command.HashMode = &value }},
		{name: "negative-hash-type", set: func(command *clientpb.CrackCommand) { command.HashType = clientpb.HashType(-2) }},
		{name: "separator-multiple-bytes", set: func(command *clientpb.CrackCommand) { command.Separator = "::" }},
		{name: "separator-nul", set: func(command *clientpb.CrackCommand) { command.Separator = "\x00" }},
		{name: "separator-carriage-return", set: func(command *clientpb.CrackCommand) { command.Separator = "\r" }},
		{name: "separator-line-feed", set: func(command *clientpb.CrackCommand) { command.Separator = "\n" }},
		{name: "rule-left-nul", set: func(command *clientpb.CrackCommand) { command.RuleLeft = "x\x00y" }},
		{name: "rule-right-nul", set: func(command *clientpb.CrackCommand) { command.RuleRight = "x\x00y" }},
		{name: "generate-rules-func-sel-nul", set: func(command *clientpb.CrackCommand) { command.GenerateRulesFuncSel = "x\x00y" }},
		{name: "encoding-from-nul", set: func(command *clientpb.CrackCommand) { command.EncodingFromName = "x\x00y" }},
		{name: "encoding-to-nul", set: func(command *clientpb.CrackCommand) { command.EncodingToName = "x\x00y" }},
		{name: "username", set: func(command *clientpb.CrackCommand) { command.Username = true }},
		{name: "keep-guessing", set: func(command *clientpb.CrackCommand) { command.KeepGuessing = true }},
		{name: "runtime", set: func(command *clientpb.CrackCommand) { command.Runtime = 1 }},
		{name: "session", set: func(command *clientpb.CrackCommand) { command.Session = "station-local-session" }},
		{name: "remove", set: func(command *clientpb.CrackCommand) { command.Remove = true }},
		{name: "remove-timer", set: func(command *clientpb.CrackCommand) { command.RemoveTimer = 1 }},
		{name: "segment-size", set: func(command *clientpb.CrackCommand) { command.SegmentSize = 32 }}, //nolint:staticcheck // Verify rejection of the deprecated compatibility field.
		{name: "negative-generate-rules-seed", set: func(command *clientpb.CrackCommand) { command.GenerateRulesSeed = -1 }},
		{name: "loopback", set: func(command *clientpb.CrackCommand) { command.Loopback = true }},
		{name: "induction-directory", set: func(command *clientpb.CrackCommand) { command.InductionDir = "induct" }},
		{name: "brain-feed", set: func(command *clientpb.CrackCommand) { command.BrainFeed = true }},
		{name: "brain-client", set: func(command *clientpb.CrackCommand) { command.BrainClient = true }},
		{name: "brain-server-timer", set: func(command *clientpb.CrackCommand) { command.BrainServerTimer = 1 }},
		{name: "brain-client-features", set: func(command *clientpb.CrackCommand) { command.BrainClientFeatures = "1" }},
		{name: "brain-host", set: func(command *clientpb.CrackCommand) { command.BrainHost = "127.0.0.1" }},
		{name: "brain-port", set: func(command *clientpb.CrackCommand) { command.BrainPort = 1 }},
		{name: "brain-password", set: func(command *clientpb.CrackCommand) { command.BrainPassword = "secret" }},
		{name: "brain-session", set: func(command *clientpb.CrackCommand) { command.BrainSession = "1" }},
		{name: "brain-session-whitelist", set: func(command *clientpb.CrackCommand) { command.BrainSessionWhitelist = "1" }},
		{name: "brain-client-features-v7", set: func(command *clientpb.CrackCommand) { command.BrainClientFeaturesV7 = 1 }},
		{name: "brain-session-v7", set: func(command *clientpb.CrackCommand) { value := uint32(0); command.BrainSessionV7 = &value }},
		{name: "brain-session-whitelist-v7", set: func(command *clientpb.CrackCommand) { command.BrainSessionWhitelistV7 = []uint32{1} }},
		{name: "brain-server-timer-v7", set: func(command *clientpb.CrackCommand) { value := uint32(0); command.BrainServerTimerV7 = &value }},
		{name: "brain-password-v7", set: func(command *clientpb.CrackCommand) { value := ""; command.BrainPasswordV7 = &value }},
		{name: "encrypt-with-pubkey", set: func(command *clientpb.CrackCommand) { command.EncryptWithPubkey = "public-key" }},
		{name: "outfile-check-directory", set: func(command *clientpb.CrackCommand) { command.OutfileCheckDir = "outfiles" }},
		{name: "outfile-check-timer", set: func(command *clientpb.CrackCommand) { command.OutfileCheckTimer = 1 }},
		{name: "outfile-check-timer-v7", set: func(command *clientpb.CrackCommand) { value := uint32(1); command.OutfileCheckTimerV7 = &value }},
		{name: "machine-readable", set: func(command *clientpb.CrackCommand) { command.MachineReadable = true }},
		{name: "debug-mode", set: func(command *clientpb.CrackCommand) { command.DebugMode = 2 }},
		{name: "debug-file", set: func(command *clientpb.CrackCommand) { command.DebugFile = "station-local.log" }},
		{name: "seek-db-path", set: func(command *clientpb.CrackCommand) { command.SeekDBPath = "station-local-directory" }},
		{name: "bridge-parameter-1", set: func(command *clientpb.CrackCommand) { command.BridgeParameter1 = "station-local-bridge" }},
		{name: "bridge-parameter-2", set: func(command *clientpb.CrackCommand) { command.BridgeParameter2 = "station-local-bridge" }},
		{name: "bridge-parameter-3", set: func(command *clientpb.CrackCommand) { command.BridgeParameter3 = "station-local-bridge" }},
		{name: "bridge-parameter-4", set: func(command *clientpb.CrackCommand) { command.BridgeParameter4 = "station-local-bridge" }},
		{name: "hwmon-disable", set: func(command *clientpb.CrackCommand) { command.HwmonDisable = true }},
		{name: "truecrypt-keyfiles", set: func(command *clientpb.CrackCommand) { command.TruecryptKeyfiles = "keyfile" }},
		{name: "veracrypt-keyfiles", set: func(command *clientpb.CrackCommand) { command.VeracryptKeyfiles = "keyfile" }},
		{name: "benchmark", set: func(command *clientpb.CrackCommand) { command.Benchmark = true }},
		{name: "benchmark-all", set: func(command *clientpb.CrackCommand) { command.BenchmarkAll = true }},
		{name: "benchmark-min", set: func(command *clientpb.CrackCommand) { command.BenchmarkMin = 1 }},
		{name: "benchmark-max", set: func(command *clientpb.CrackCommand) { value := uint32(0); command.BenchmarkMax = &value }},
		{name: "speed-only", set: func(command *clientpb.CrackCommand) { command.SpeedOnly = true }},
		{name: "progress-only", set: func(command *clientpb.CrackCommand) { command.ProgressOnly = true }},
		{name: "stdout", set: func(command *clientpb.CrackCommand) { command.Stdout = true }},
		{name: "show", set: func(command *clientpb.CrackCommand) { command.Show = true }},
		{name: "left", set: func(command *clientpb.CrackCommand) { command.Left = true }},
		{name: "hash-info", set: func(command *clientpb.CrackCommand) { command.HashInfo = true }},
		{name: "hash-info-level", set: func(command *clientpb.CrackCommand) { command.HashInfoLevel = 1 }},
		{name: "backend-info", set: func(command *clientpb.CrackCommand) { command.BackendInfo = true }},
		{name: "backend-info-level", set: func(command *clientpb.CrackCommand) { command.BackendInfoLevel = 1 }},
		{name: "keyspace", set: func(command *clientpb.CrackCommand) { command.Keyspace = true }},
		{name: "total-candidates", set: func(command *clientpb.CrackCommand) { command.TotalCandidates = true }},
		{name: "lookup", set: func(command *clientpb.CrackCommand) { command.Lookup = "candidate" }},
		{name: "identify-mode", set: func(command *clientpb.CrackCommand) { command.IdentifyMode = true }},
		{name: "brain-server", set: func(command *clientpb.CrackCommand) { command.BrainServer = true }},
		{name: "restore", set: func(command *clientpb.CrackCommand) { command.Restore = true }}, //nolint:staticcheck // Verify rejection of the deprecated compatibility field.
		{name: "restore-position", set: func(command *clientpb.CrackCommand) { command.RestorePosition = true }},
		{name: "restore-show", set: func(command *clientpb.CrackCommand) { command.RestoreShowCommand = true }},
		{name: "restore-file", set: func(command *clientpb.CrackCommand) { command.RestoreFile = []byte("restore") }},
		{name: "stdin", set: func(command *clientpb.CrackCommand) { command.Stdin = []byte("candidate") }},
		{name: "local-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"/tmp/local.hcmask"} }},
		{name: "traversing-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"masks/../local.hcmask"} }},
		{name: "device-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"NUL"} }},
		{name: "nested-device-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"foo/NUL/bar"} }},
		{name: "console-device-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"foo/CONIN$/bar"} }},
		{name: "device-stream-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"NUL:stream"} }},
		{name: "dot-only-mask-path", set: func(command *clientpb.CrackCommand) { command.PositionalArguments = []string{"foo/.../bar"} }},
		{name: "custom-charset-1-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset1 = "/tmp/charset" }},
		{name: "custom-charset-2-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset2 = "/tmp/charset" }},
		{name: "custom-charset-3-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset3 = "/tmp/charset" }},
		{name: "custom-charset-4-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset4 = "/tmp/charset" }},
		{name: "custom-charset-5-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset5 = "/tmp/charset" }},
		{name: "custom-charset-6-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset6 = "/tmp/charset" }},
		{name: "custom-charset-7-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset7 = "/tmp/charset" }},
		{name: "custom-charset-8-path", set: func(command *clientpb.CrackCommand) { command.CustomCharset8 = "/tmp/charset" }},
	}
	for _, accepted := range []models.CrackCommand{
		{HashType: int32(clientpb.HashType_MD5)},
		{HashType: int32(clientpb.HashType_INVALID)},
	} {
		if err := validateDistributedCrackCommand(&accepted); err != nil {
			t.Fatalf("valid default hash type %d rejected: %v", accepted.HashType, err)
		}
		if effectiveCrackHashType(&accepted) != int32(clientpb.HashType_MD5) {
			t.Fatalf("effective hash type for %d = %d, want MD5", accepted.HashType, effectiveCrackHashType(&accepted))
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &clientpb.CrackCommand{
				AttackMode: clientpb.CrackAttackMode_BRUTEFORCE, Hashes: []string{"hash"}, PositionalArguments: []string{"?d"},
			}
			test.set(request)
			if _, err := rpcServer.Crack(t.Context(), request); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("Crack error = %v, want InvalidArgument", err)
			}
			var jobs, tasks int64
			if err := database.Model(&models.CrackJob{}).Count(&jobs).Error; err != nil {
				t.Fatalf("count crack jobs: %v", err)
			}
			if err := database.Model(&models.CrackTask{}).Count(&tasks).Error; err != nil {
				t.Fatalf("count crack tasks: %v", err)
			}
			if jobs != 0 || tasks != 0 {
				t.Fatalf("invalid command persisted jobs=%d tasks=%d", jobs, tasks)
			}
		})
	}

	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	response, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode:          clientpb.CrackAttackMode_BRUTEFORCE,
		Hashes:              []string{"hash"},
		PositionalArguments: []string{"?l/?d"},
		BackendDevices:      []uint32{1, 3},
		Separator:           ":",
		CustomCharset1:      "?l/?d",
		CustomCharset8:      "abc/DEF",
	})
	if err != nil {
		t.Fatalf("queue backend-device command: %v", err)
	}
	jobID := models.ParseUUIDOrNil(response.Job.ID)
	var parent models.CrackCommand
	if err := database.First(&parent, "crack_job_id = ?", jobID).Error; err != nil {
		t.Fatalf("load parent command: %v", err)
	}
	if !parent.RestoreDisable || !parent.LogfileDisable || !parent.HashCopy || parent.OutfileCheckTimer != 0 || parent.OutfileCheckTimerV7 == nil || *parent.OutfileCheckTimerV7 != 0 || parent.Separator != ":" || !slices.Equal(parent.BackendDevices, []uint32{1, 3}) || parent.CustomCharset1 != "?l/?d" || parent.CustomCharset8 != "abc/DEF" {
		t.Fatalf("persisted parent command = %#v", parent)
	}
	var task models.CrackTask
	if err := database.Preload("Command").First(&task, "crack_job_id = ?", jobID).Error; err != nil {
		t.Fatalf("load keyspace task: %v", err)
	}
	if !task.Command.Keyspace || !task.Command.RestoreDisable || !task.Command.LogfileDisable || !task.Command.HashCopy || task.Command.OutfileCheckTimer != 0 || task.Command.OutfileCheckTimerV7 == nil || *task.Command.OutfileCheckTimerV7 != 0 || task.Command.Separator != ":" || !slices.Equal(task.Command.BackendDevices, []uint32{1, 3}) {
		t.Fatalf("persisted internal command = %#v", task.Command)
	}
	internalCommand := task.Command
	internalCommand.Keyspace = false // Keyspace is the one server-owned terminating mode.
	if err := validateDistributedCrackCommand(&internalCommand); err != nil {
		t.Fatalf("internal command retained a user-controlled terminating mode: %v", err)
	}
}

//nolint:gocyclo // The test compares generated-rule seeds across assignments and persisted commands.
func TestGeneratedRulesSeedIsStableAcrossDistributedTasks(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", map[int32]uint64{int32(clientpb.HashType_MD5): 100})
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	rpcServer := &Server{}
	response, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode:          clientpb.CrackAttackMode_BRUTEFORCE,
		Hashes:              []string{"hash"},
		PositionalArguments: []string{"?d"},
		GenerateRules:       10,
	})
	if err != nil {
		t.Fatalf("queue generated-rules job: %v", err)
	}
	jobID := models.ParseUUIDOrNil(response.Job.ID)
	var parent models.CrackCommand
	if err := database.First(&parent, "crack_job_id = ?", jobID).Error; err != nil {
		t.Fatalf("load generated-rules parent command: %v", err)
	}
	if parent.GenerateRulesSeedV7 == nil || !parent.HashCopy || parent.OutfileCheckTimerV7 == nil || *parent.OutfileCheckTimerV7 != 0 {
		t.Fatalf("server did not persist distributed invariants: %#v", parent)
	}
	var keyspaceTask models.CrackTask
	if err := database.Preload("Command").First(&keyspaceTask, "crack_job_id = ? AND kind = ?", jobID, int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE)).Error; err != nil {
		t.Fatalf("load generated-rules keyspace task: %v", err)
	}
	if keyspaceTask.Command.GenerateRulesSeedV7 == nil || *keyspaceTask.Command.GenerateRulesSeedV7 != *parent.GenerateRulesSeedV7 || !keyspaceTask.Command.HashCopy || keyspaceTask.Command.OutfileCheckTimerV7 == nil || *keyspaceTask.Command.OutfileCheckTimerV7 != 0 {
		t.Fatalf("keyspace seed = %#v, parent seed = %#v", keyspaceTask.Command.GenerateRulesSeedV7, parent.GenerateRulesSeedV7)
	}
	update := keyspaceTask.ToProtobuf()
	update.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	update.CompletedAt = time.Now().Unix()
	update.Keyspace = "100"
	if _, err := updateLeasedCrackTask(update); err != nil {
		t.Fatalf("complete generated-rules keyspace task: %v", err)
	}
	var shard models.CrackTask
	if err := database.Preload("Command").First(&shard, "crack_job_id = ? AND kind = ?", jobID, int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK)).Error; err != nil {
		t.Fatalf("load generated-rules shard: %v", err)
	}
	if shard.Command.GenerateRulesSeedV7 == nil || *shard.Command.GenerateRulesSeedV7 != *parent.GenerateRulesSeedV7 || !shard.Command.HashCopy || shard.Command.OutfileCheckTimerV7 == nil || *shard.Command.OutfileCheckTimerV7 != 0 {
		t.Fatalf("shard seed = %#v, parent seed = %#v", shard.Command.GenerateRulesSeedV7, parent.GenerateRulesSeedV7)
	}

	explicitZero := uint32(0)
	zeroResponse, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
		AttackMode:          clientpb.CrackAttackMode_BRUTEFORCE,
		Hashes:              []string{"second-hash"},
		PositionalArguments: []string{"?d"},
		GenerateRules:       10,
		GenerateRulesSeedV7: &explicitZero,
	})
	if err != nil {
		t.Fatalf("queue explicit-zero generated-rules job: %v", err)
	}
	var zeroParent models.CrackCommand
	if err := database.First(&zeroParent, "crack_job_id = ?", models.ParseUUIDOrNil(zeroResponse.Job.ID)).Error; err != nil {
		t.Fatalf("load explicit-zero parent command: %v", err)
	}
	if zeroParent.GenerateRulesSeedV7 == nil || *zeroParent.GenerateRulesSeedV7 != 0 {
		t.Fatalf("explicit zero seed changed to %#v", zeroParent.GenerateRulesSeedV7)
	}

	select {
	case <-station.Events:
	default:
	}
}

func TestDistributedNormalizationDisablesInstallationLocalMarkovData(t *testing.T) {
	withoutManagedHcstat := &models.CrackCommand{}
	normalizeDistributedCrackCommand(withoutManagedHcstat, models.NewUUID())
	if !withoutManagedHcstat.MarkovDisable {
		t.Fatal("Markov ordering remained enabled without a managed hcstat2 file")
	}

	withManagedHcstat := &models.CrackCommand{MarkovHcstat2: []byte("crackfile://hcstat2/" + strings.Repeat("a", 64))}
	normalizeDistributedCrackCommand(withManagedHcstat, models.NewUUID())
	if withManagedHcstat.MarkovDisable {
		t.Fatal("Markov ordering was disabled despite an explicit managed hcstat2 file")
	}
}

//nolint:gocyclo // List and detail responses must redact task secrets while retaining public job metadata.
func TestCrackJobRPCsRedactLeaseTokens(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	job := createQueueTestJob(t, database, models.CrackCommand{HashType: int32(clientpb.HashType_MD5), Hashes: []string{"hash"}})
	task := &models.CrackTask{
		CrackJobID:       job.ID,
		Kind:             int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		State:            int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:          1,
		LeaseToken:       "operator-must-not-see-this",
		LatestStatusJSON: []byte(`{"progress":[1,2]}`),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	results := []models.CrackResult{
		{CrackJobID: job.ID, CrackTaskID: task.ID, Hash: "hash", Plaintext: []byte("one"), Fingerprint: strings.Repeat("1", 64)},
		{CrackJobID: job.ID, CrackTaskID: task.ID, Hash: "hash", Plaintext: []byte("two"), Fingerprint: strings.Repeat("2", 64)},
	}
	if err := database.Create(&results).Error; err != nil {
		t.Fatalf("create results: %v", err)
	}
	rpcServer := &Server{}
	jobs, err := rpcServer.CrackJobs(t.Context(), nil)
	if err != nil {
		t.Fatalf("list crack jobs: %v", err)
	}
	if len(jobs.Jobs) != 1 || len(jobs.Jobs[0].Tasks) != 1 {
		t.Fatalf("listed jobs = %#v", jobs.Jobs)
	}
	if jobs.Jobs[0].Command != nil || jobs.Jobs[0].Tasks[0].LeaseToken != "" || len(jobs.Jobs[0].Tasks[0].LatestStatusJSON) != 0 {
		t.Fatalf("job summary exposed private task data: %#v", jobs.Jobs[0].Tasks[0])
	}
	if jobs.Jobs[0].ResultCount != 2 || len(jobs.Jobs[0].Results) != 0 {
		t.Fatalf("job summary results = count %d, payload %#v", jobs.Jobs[0].ResultCount, jobs.Jobs[0].Results)
	}
	detail, err := rpcServer.CrackJobByID(t.Context(), &clientpb.CrackJob{ID: job.ID.String()})
	if err != nil {
		t.Fatalf("get crack job: %v", err)
	}
	if len(detail.Tasks) != 1 || detail.Tasks[0].LeaseToken != "" {
		t.Fatalf("job detail exposed lease token: %#v", detail.Tasks)
	}
	if detail.Command == nil || !slices.Equal(detail.Command.Hashes, []string{"hash"}) {
		t.Fatalf("job detail lost parent command: %#v", detail.Command)
	}
	if !bytes.Equal(detail.Tasks[0].LatestStatusJSON, task.LatestStatusJSON) {
		t.Fatalf("job detail status = %s, want %s", detail.Tasks[0].LatestStatusJSON, task.LatestStatusJSON)
	}
	if detail.ResultCount != 2 || len(detail.Results) != 2 {
		t.Fatalf("job detail results = count %d, payload %#v", detail.ResultCount, detail.Results)
	}
}

func TestCrackJobStatusDerivesFailureFromTaskState(t *testing.T) {
	job := &models.CrackJob{CompletedAt: time.Now(), Tasks: []models.CrackTask{{State: int32(clientpb.CrackTaskState_CRACK_TASK_FAILED)}}}
	if got := job.Status(); got != clientpb.CrackJobStatus_FAILED {
		t.Fatalf("job status = %s, want FAILED", got)
	}
	job.CompletedAt = time.Time{}
	job.Err = "one shard failed"
	job.Tasks = append(job.Tasks, models.CrackTask{State: int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING)})
	if got := job.Status(); got != clientpb.CrackJobStatus_IN_PROGRESS {
		t.Fatalf("mixed running/failed job status = %s, want IN_PROGRESS", got)
	}
}

//nolint:gocyclo // Multi-chunk verification, publication, and listing visibility form one upload lifecycle.
func TestCrackFileCompletionVerifiesMultichunkStreamAndListsOnlyComplete(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	payload := bytes.Repeat([]byte("hashcat-wordlist-entry\n"), 4096)
	digest := sha256.Sum256(payload)
	digestText := hex.EncodeToString(digest[:])
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("create zstd encoder: %v", err)
	}
	if _, err := encoder.Write(payload); err != nil {
		t.Fatalf("compress payload: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("close zstd encoder: %v", err)
	}
	crackFile := &models.CrackFile{
		Name: "words.txt", Type: int32(clientpb.CrackFileType_WORDLIST),
		UncompressedSize: int64(len(payload)), IsCompressed: true,
	}
	if err := database.Create(crackFile).Error; err != nil {
		t.Fatalf("create crack file: %v", err)
	}
	chunkData := compressed.Bytes()
	cutOne, cutTwo := len(chunkData)/3, 2*len(chunkData)/3
	parts := [][]byte{chunkData[:cutOne], chunkData[cutOne:cutTwo], chunkData[cutTwo:]}
	chunkDir := assets.GetChunkDataDir()
	for index, part := range parts {
		chunk := &models.CrackFileChunk{CrackFileID: crackFile.ID, N: uint32(index)}
		if err := database.Create(chunk).Error; err != nil {
			t.Fatalf("create chunk %d: %v", index, err)
		}
		if err := os.WriteFile(filepath.Join(chunkDir, chunk.ID.String()), part, 0600); err != nil {
			t.Fatalf("write chunk %d: %v", index, err)
		}
		crackFile.Chunks = append(crackFile.Chunks, *chunk)
	}
	if err := verifyCrackFileUpload(crackFile, digestText); err != nil {
		t.Fatalf("verify multichunk upload: %v", err)
	}
	wrongSize := *crackFile
	wrongSize.UncompressedSize++
	if err := verifyCrackFileUpload(&wrongSize, digestText); err == nil {
		t.Fatal("wrong uncompressed size was accepted")
	}
	if err := verifyCrackFileUpload(crackFile, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong digest was accepted")
	}
	noncontiguous := *crackFile
	noncontiguous.Chunks = []models.CrackFileChunk{crackFile.Chunks[0], crackFile.Chunks[2]}
	if err := verifyCrackFileUpload(&noncontiguous, digestText); err == nil {
		t.Fatal("noncontiguous chunks were accepted")
	}
	duplicate := &models.CrackFileChunk{CrackFileID: crackFile.ID, N: 0}
	if err := database.Create(duplicate).Error; err == nil {
		t.Fatal("duplicate crack file chunk index was accepted")
	}

	incomplete := &models.CrackFile{Name: "incomplete.txt", Type: int32(clientpb.CrackFileType_WORDLIST), UncompressedSize: 1}
	if err := database.Create(incomplete).Error; err != nil {
		t.Fatalf("create incomplete file: %v", err)
	}
	if _, err := (&Server{}).CrackFileChunkDownload(t.Context(), &clientpb.CrackFileChunk{CrackFileID: incomplete.ID.String()}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("incomplete download error = %v, want FailedPrecondition", err)
	}
	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	if _, err := (&Server{}).CrackFileComplete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String(), Sha2_256: digestText}); err != nil {
		t.Fatalf("complete crack file: %v", err)
	}
	select {
	case event := <-events:
		if event.EventType != consts.CrackFileUpdated || string(event.Data) != crackFile.ID.String() {
			t.Fatalf("completion event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("crack-file-updated event was not published")
	}
	files, err := db.AllCrackFiles()
	if err != nil {
		t.Fatalf("list crack files: %v", err)
	}
	if len(files) != 1 || files[0].ID != crackFile.ID {
		t.Fatalf("completed files = %#v", files)
	}
	rpcServer := &Server{}
	if _, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "invalid", Type: clientpb.CrackFileType_INVALID_TYPE, UncompressedSize: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid crack file type error = %v", err)
	}
	if _, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: crackFile.Name, Type: clientpb.CrackFileType_WORDLIST, UncompressedSize: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("duplicate crack file name error = %v", err)
	}
	if _, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: crackFile.Name, Type: clientpb.CrackFileType_RULES, UncompressedSize: 1}); err != nil {
		t.Fatalf("same name for different crack file type: %v", err)
	}
	roundTrip, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{
		Name: "roundtrip.txt", Type: clientpb.CrackFileType_WORDLIST,
		UncompressedSize: int64(len(payload)), IsCompressed: true,
	})
	if err != nil {
		t.Fatalf("create round-trip crack file: %v", err)
	}
	roundTripParts := [][]byte{chunkData[:len(chunkData)/2], chunkData[len(chunkData)/2:]}
	for index, part := range roundTripParts {
		if _, err := rpcServer.CrackFileChunkUpload(t.Context(), &clientpb.CrackFileChunk{CrackFileID: roundTrip.ID, N: uint32(index), Data: part}); err != nil {
			t.Fatalf("upload round-trip chunk %d: %v", index, err)
		}
	}
	if _, err := rpcServer.CrackFileComplete(t.Context(), &clientpb.CrackFile{ID: roundTrip.ID, Sha2_256: digestText}); err != nil {
		t.Fatalf("complete round-trip crack file: %v", err)
	}
	storedRoundTrip, err := db.GetByCrackFileByID(roundTrip.ID)
	if err != nil {
		t.Fatalf("load round-trip crack file: %v", err)
	}
	sort.Slice(storedRoundTrip.Chunks, func(i, j int) bool { return storedRoundTrip.Chunks[i].N < storedRoundTrip.Chunks[j].N })
	var downloaded bytes.Buffer
	for _, chunk := range storedRoundTrip.Chunks {
		response, err := rpcServer.CrackFileChunkDownload(t.Context(), &clientpb.CrackFileChunk{ID: chunk.ID.String(), CrackFileID: roundTrip.ID})
		if err != nil {
			t.Fatalf("download round-trip chunk %d: %v", chunk.N, err)
		}
		downloaded.Write(response.Data)
	}
	if !bytes.Equal(downloaded.Bytes(), chunkData) {
		t.Fatal("round-trip compressed bytes changed")
	}
	if _, err := rpcServer.CrackFileComplete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String(), Sha2_256: strings.Repeat("f", 64)}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("mismatched idempotent completion error = %v", err)
	}
}

//nolint:gocyclo // The table exercises persisted-size and accumulation overflow boundaries together.
func TestCrackFileSizeArithmeticRejectsOverflow(t *testing.T) {
	if value, ok := checkedCrackFileSizeAdd(math.MaxInt64-1, 1); !ok || value != math.MaxInt64 {
		t.Fatalf("checked max-boundary addition = %d, %v", value, ok)
	}
	if _, ok := checkedCrackFileSizeAdd(math.MaxInt64, 1); ok {
		t.Fatal("overflowing crack file size addition succeeded")
	}
	if _, ok := checkedCrackFileSizeAdd(-1, 1); ok {
		t.Fatal("negative crack file size addition succeeded")
	}

	t.Run("quota and disk sum", func(t *testing.T) {
		database := setupCrackstationRPCTestDB(t)
		t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
		if err := configs.SaveCrackConfig(&clientpb.CrackConfig{MaxFileSize: math.MaxInt64, MaxDiskUsage: math.MaxInt64, ChunkSize: 1024}); err != nil {
			t.Fatalf("save near-int64 crack config: %v", err)
		}
		existing := &models.CrackFile{Name: "near-limit", Type: int32(clientpb.CrackFileType_WORDLIST), UncompressedSize: math.MaxInt64 - 5}
		if err := database.Create(existing).Error; err != nil {
			t.Fatalf("create near-limit crack file: %v", err)
		}
		rpcServer := &Server{}
		if _, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "exact-limit", Type: clientpb.CrackFileType_RULES, UncompressedSize: 5}); err != nil {
			t.Fatalf("create exact-limit crack file: %v", err)
		}
		if _, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "overflow", Type: clientpb.CrackFileType_MARKOV_HCSTAT2, UncompressedSize: 1}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("overflowing create error = %v, want InvalidArgument", err)
		}
		if err := database.Create(&models.CrackFile{Name: "forced-overflow", Type: int32(clientpb.CrackFileType_MARKOV_HCSTAT2), UncompressedSize: 1}).Error; err != nil {
			t.Fatalf("create forced overflow row: %v", err)
		}
		if usage, err := db.CrackFilesDiskUsage(); err == nil || usage != -1 {
			t.Fatalf("overflowing disk usage = %d, err=%v", usage, err)
		}

		overflowUpload := &models.CrackFile{Name: "overflow-upload", Type: int32(clientpb.CrackFileType_WORDLIST), UncompressedSize: math.MaxInt64, CompressedSize: math.MaxInt64 - 1}
		if err := database.Create(overflowUpload).Error; err != nil {
			t.Fatalf("create overflowing upload: %v", err)
		}
		if _, err := rpcServer.CrackFileChunkUpload(t.Context(), &clientpb.CrackFileChunk{CrackFileID: overflowUpload.ID.String(), N: 0, Data: []byte("xx")}); status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("overflowing chunk upload error = %v, want ResourceExhausted", err)
		}
	})

	t.Run("negative persisted size", func(t *testing.T) {
		database := setupCrackstationRPCTestDB(t)
		if err := database.Create(&models.CrackFile{Name: "negative", Type: int32(clientpb.CrackFileType_WORDLIST), UncompressedSize: -1}).Error; err != nil {
			t.Fatalf("create negative-size row: %v", err)
		}
		if usage, err := db.CrackFilesDiskUsage(); err == nil || usage != -1 {
			t.Fatalf("negative disk usage = %d, err=%v", usage, err)
		}
	})
}

func TestActiveCrackJobsProtectManagedFileReferences(t *testing.T) {
	referenceFields := []struct {
		name     string
		fileType clientpb.CrackFileType
		set      func(*models.CrackCommand, string)
	}{
		{name: "positional", fileType: clientpb.CrackFileType_WORDLIST, set: func(command *models.CrackCommand, uri string) { command.PositionalArguments = []string{uri} }},
		{name: "identify", fileType: clientpb.CrackFileType_WORDLIST, set: func(command *models.CrackCommand, uri string) { command.Identify = uri }},
		{name: "rules-file", fileType: clientpb.CrackFileType_RULES, set: func(command *models.CrackCommand, uri string) { command.RulesFile = []byte(uri) }},
		{name: "rules-files-v7", fileType: clientpb.CrackFileType_RULES, set: func(command *models.CrackCommand, uri string) { command.RulesFilesV7 = [][]byte{[]byte(uri)} }},
		{name: "markov", fileType: clientpb.CrackFileType_MARKOV_HCSTAT2, set: func(command *models.CrackCommand, uri string) { command.MarkovHcstat2 = []byte(uri) }},
	}
	for _, test := range referenceFields {
		t.Run(test.name, func(t *testing.T) {
			database := setupCrackstationRPCTestDB(t)
			t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
			crackFile := &models.CrackFile{
				Name:             test.name,
				Type:             int32(test.fileType),
				UncompressedSize: 1,
				CompressedSize:   1,
				Sha2_256:         strings.Repeat("a", 64),
				IsComplete:       true,
			}
			if err := database.Create(crackFile).Error; err != nil {
				t.Fatalf("create managed file: %v", err)
			}
			chunk := &models.CrackFileChunk{CrackFileID: crackFile.ID, N: 0}
			if err := database.Create(chunk).Error; err != nil {
				t.Fatalf("create managed file chunk: %v", err)
			}
			chunkPath := filepath.Join(assets.GetChunkDataDir(), chunk.ID.String())
			if err := os.WriteFile(chunkPath, []byte("x"), 0600); err != nil {
				t.Fatalf("write managed file chunk: %v", err)
			}
			uri := managedCrackFileURI(crackFile)
			command := models.CrackCommand{HashType: int32(clientpb.HashType_MD5), Hashes: []string{"hash"}}
			test.set(&command, uri)
			job := createQueueTestJob(t, database, command)

			referenced, err := activeCrackJobReferencesManagedFile(database, crackFile)
			if err != nil {
				t.Fatalf("check active reference: %v", err)
			}
			if !referenced {
				t.Fatalf("serialized %s reference was not found", test.name)
			}
			rpcServer := &Server{}
			if _, err := rpcServer.CrackFileDelete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String()}); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("active reference delete error = %v, want FailedPrecondition", err)
			}
			if _, err := db.GetByCrackFileByID(crackFile.ID.String()); err != nil {
				t.Fatalf("active referenced file was deleted: %v", err)
			}
			if _, err := os.Stat(chunkPath); err != nil {
				t.Fatalf("active referenced chunk was deleted: %v", err)
			}

			if err := database.Model(&models.CrackJob{}).Where("id = ?", job.ID).Update("completed_at", time.Now()).Error; err != nil {
				t.Fatalf("complete crack job: %v", err)
			}
			if _, err := rpcServer.CrackFileDelete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String()}); err != nil {
				t.Fatalf("delete terminal job's managed file: %v", err)
			}
			if _, err := db.GetByCrackFileByID(crackFile.ID.String()); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("deleted file lookup error = %v, want record not found", err)
			}
			if _, err := os.Stat(chunkPath); !os.IsNotExist(err) {
				t.Fatalf("deleted chunk stat error = %v, want not exist", err)
			}
		})
	}
}

func createCrackFileDeleteFixture(t *testing.T, database *gorm.DB) (*models.CrackFile, *models.CrackFileChunk, string, []byte) {
	t.Helper()
	data := []byte("durable-delete-fixture")
	crackFile := &models.CrackFile{
		Name:             "delete-fixture",
		Type:             int32(clientpb.CrackFileType_WORDLIST),
		UncompressedSize: int64(len(data)),
		CompressedSize:   int64(len(data)),
		Sha2_256:         strings.Repeat("d", 64),
		IsComplete:       true,
	}
	if err := database.Create(crackFile).Error; err != nil {
		t.Fatalf("create delete fixture: %v", err)
	}
	chunk := &models.CrackFileChunk{CrackFileID: crackFile.ID, N: 0}
	if err := database.Create(chunk).Error; err != nil {
		t.Fatalf("create delete fixture chunk: %v", err)
	}
	chunkPath := filepath.Join(assets.GetChunkDataDir(), chunk.ID.String())
	if err := os.WriteFile(chunkPath, data, 0600); err != nil {
		t.Fatalf("write delete fixture chunk: %v", err)
	}
	return crackFile, chunk, chunkPath, data
}

func TestCrackFileDeleteRollsBackDatabaseBeforeRemovingChunks(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	crackFile, chunk, chunkPath, data := createCrackFileDeleteFixture(t, database)
	injectedErr := errors.New("injected crack file delete failure")
	callbackName := "test:reject-crack-file-parent-delete"
	if err := database.Callback().Delete().Before("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.CrackFile); ok {
			_ = tx.AddError(injectedErr)
		}
	}); err != nil {
		t.Fatalf("register delete failure callback: %v", err)
	}
	t.Cleanup(func() { _ = database.Callback().Delete().Remove(callbackName) })
	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	if _, err := (&Server{}).CrackFileDelete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String()}); status.Code(err) != codes.Internal {
		t.Fatalf("injected database delete error = %v, want Internal", err)
	}
	stored, err := db.GetByCrackFileByID(crackFile.ID.String())
	if err != nil {
		t.Fatalf("completed manifest was not rolled back: %v", err)
	}
	if !stored.IsComplete || len(stored.Chunks) != 1 || stored.Chunks[0].ID != chunk.ID {
		t.Fatalf("rolled-back manifest/chunks = %#v", stored)
	}
	storedData, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatalf("backing chunk was removed before database commit: %v", err)
	}
	if !bytes.Equal(storedData, data) {
		t.Fatalf("backing chunk changed: %q", storedData)
	}
	select {
	case event := <-events:
		if event.EventType == consts.CrackFileUpdated && string(event.Data) == crackFile.ID.String() {
			t.Fatalf("failed deletion published completion event: %#v", event)
		}
	default:
	}
}

func TestCrackFileDeleteCommitsBeforeBestEffortChunkCleanup(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	crackFile, chunk, chunkPath, data := createCrackFileDeleteFixture(t, database)
	originalRemove := removeCrackFileChunk
	cleanupErr := errors.New("injected chunk cleanup failure")
	removeCrackFileChunk = func(path string) error {
		if path == chunkPath {
			return cleanupErr
		}
		return originalRemove(path)
	}
	t.Cleanup(func() { removeCrackFileChunk = originalRemove })
	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	if _, err := (&Server{}).CrackFileDelete(t.Context(), &clientpb.CrackFile{ID: crackFile.ID.String()}); err != nil {
		t.Fatalf("best-effort cleanup delete: %v", err)
	}
	if _, err := db.GetByCrackFileByID(crackFile.ID.String()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted manifest lookup error = %v, want record not found", err)
	}
	var chunkCount int64
	if err := database.Model(&models.CrackFileChunk{}).Where("id = ?", chunk.ID).Count(&chunkCount).Error; err != nil || chunkCount != 0 {
		t.Fatalf("deleted chunk record count = %d, err=%v", chunkCount, err)
	}
	storedData, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatalf("injected cleanup orphan was not preserved: %v", err)
	}
	if !bytes.Equal(storedData, data) {
		t.Fatalf("cleanup orphan changed: %q", storedData)
	}
	select {
	case event := <-events:
		if event.EventType != consts.CrackFileUpdated || string(event.Data) != crackFile.ID.String() {
			t.Fatalf("successful logical delete event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("logical delete did not publish crack-file-updated")
	}
	if err := originalRemove(chunkPath); err != nil {
		t.Fatalf("clean up injected orphan: %v", err)
	}
}

//nolint:gocyclo // Stale reservation reaping, orphan cleanup, and live-upload preservation share one scenario.
func TestCrackFileCreateReapsOnlyStaleIncompleteReservationsAndOrphans(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	if err := configs.SaveCrackConfig(&clientpb.CrackConfig{MaxFileSize: 1 << 20, MaxDiskUsage: 1 << 20, ChunkSize: 1024}); err != nil {
		t.Fatalf("save crack config: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	originalNow := crackFileNow
	crackFileNow = func() time.Time { return now }
	t.Cleanup(func() { crackFileNow = originalNow })

	createReservation := func(name string, complete bool, activity time.Time) (*models.CrackFile, string) {
		t.Helper()
		crackFile := &models.CrackFile{Name: name, Type: int32(clientpb.CrackFileType_WORDLIST), UncompressedSize: 1, CompressedSize: 1, IsComplete: complete, LastModified: activity}
		if err := database.Create(crackFile).Error; err != nil {
			t.Fatalf("create %s reservation: %v", name, err)
		}
		if err := database.Exec("UPDATE crack_files SET created_at = ?, last_modified = ? WHERE id = ?", activity, activity, crackFile.ID).Error; err != nil {
			t.Fatalf("age %s reservation: %v", name, err)
		}
		chunk := &models.CrackFileChunk{CrackFileID: crackFile.ID, N: 0}
		if err := database.Create(chunk).Error; err != nil {
			t.Fatalf("create %s chunk: %v", name, err)
		}
		chunkPath := filepath.Join(assets.GetChunkDataDir(), chunk.ID.String())
		if err := os.WriteFile(chunkPath, []byte(name), 0600); err != nil {
			t.Fatalf("write %s chunk: %v", name, err)
		}
		return crackFile, chunkPath
	}
	stale, staleChunkPath := createReservation("retry.txt", false, now.Add(-crackFileUploadStaleAfter-time.Hour))
	active, activeChunkPath := createReservation("active.txt", false, now.Add(-crackFileUploadStaleAfter+time.Hour))
	complete, completeChunkPath := createReservation("complete.txt", true, now.Add(-2*crackFileUploadStaleAfter))
	chunkDir := assets.GetChunkDataDir()
	staleTempPath := filepath.Join(chunkDir, ".crack-upload-stale")
	activeTempPath := filepath.Join(chunkDir, ".crack-upload-active")
	unrelatedPath := filepath.Join(chunkDir, "unrelated-stale-file")
	preservedDirectory := filepath.Join(chunkDir, ".crack-upload-directory")
	staleOrphanPath := filepath.Join(chunkDir, models.NewUUID().String())
	activeOrphanPath := filepath.Join(chunkDir, models.NewUUID().String())
	for _, path := range []string{staleTempPath, activeTempPath, unrelatedPath, staleOrphanPath, activeOrphanPath} {
		if err := os.WriteFile(path, []byte("temporary"), 0600); err != nil {
			t.Fatalf("write staging fixture %s: %v", path, err)
		}
	}
	if err := os.Mkdir(preservedDirectory, 0700); err != nil {
		t.Fatalf("create staging directory fixture: %v", err)
	}
	staleTime := now.Add(-crackFileUploadStaleAfter - time.Hour)
	for _, path := range []string{staleTempPath, unrelatedPath, preservedDirectory, staleOrphanPath, completeChunkPath} {
		if err := os.Chtimes(path, staleTime, staleTime); err != nil {
			t.Fatalf("age fixture %s: %v", path, err)
		}
	}
	activeTime := now.Add(-crackFileUploadStaleAfter + time.Hour)
	for _, path := range []string{activeTempPath, activeOrphanPath} {
		if err := os.Chtimes(path, activeTime, activeTime); err != nil {
			t.Fatalf("age active fixture %s: %v", path, err)
		}
	}

	created, err := (&Server{}).CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "retry.txt", Type: clientpb.CrackFileType_WORDLIST, UncompressedSize: 1})
	if err != nil {
		t.Fatalf("replace stale upload reservation: %v", err)
	}
	if created.ID == stale.ID.String() || created.LastModified != now.Unix() {
		t.Fatalf("replacement reservation = %#v", created)
	}
	if _, err := db.GetByCrackFileByID(stale.ID.String()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("stale reservation lookup error = %v, want record not found", err)
	}
	for _, removed := range []string{staleChunkPath, staleTempPath, staleOrphanPath} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("stale entry %s stat error = %v, want not exist", removed, err)
		}
	}
	for _, path := range []string{activeTempPath, activeOrphanPath, unrelatedPath, preservedDirectory} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("preserved staging entry %s missing: %v", path, err)
		}
	}
	for _, preserved := range []struct {
		file *models.CrackFile
		path string
	}{{active, activeChunkPath}, {complete, completeChunkPath}} {
		if _, err := db.GetByCrackFileByID(preserved.file.ID.String()); err != nil {
			t.Fatalf("preserved reservation %s missing: %v", preserved.file.Name, err)
		}
		if _, err := os.Stat(preserved.path); err != nil {
			t.Fatalf("preserved chunk %s missing: %v", preserved.path, err)
		}
	}
	if _, err := (&Server{}).CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "active.txt", Type: clientpb.CrackFileType_WORDLIST, UncompressedSize: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("active duplicate create error = %v, want InvalidArgument", err)
	}
}

func TestCrackFileChunkUploadSyncsDirectoryBeforeDatabaseCommit(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	if err := configs.SaveCrackConfig(&clientpb.CrackConfig{MaxFileSize: 1 << 20, MaxDiskUsage: 1 << 20, ChunkSize: 1024}); err != nil {
		t.Fatalf("save crack config: %v", err)
	}
	rpcServer := &Server{}
	crackFile, err := rpcServer.CrackFileCreate(t.Context(), &clientpb.CrackFile{Name: "sync-failure.txt", Type: clientpb.CrackFileType_WORDLIST, UncompressedSize: 4})
	if err != nil {
		t.Fatalf("create crack file: %v", err)
	}
	originalSync := syncCrackFileChunkDirectory
	injectedErr := errors.New("injected directory sync failure")
	syncCalls := 0
	syncCrackFileChunkDirectory = func(path string) error {
		syncCalls++
		var chunkCount int64
		if err := database.Model(&models.CrackFileChunk{}).Where("crack_file_id = ?", models.ParseUUIDOrNil(crackFile.ID)).Count(&chunkCount).Error; err != nil {
			t.Fatalf("count chunks during directory sync: %v", err)
		}
		if chunkCount != 0 {
			t.Fatalf("chunk record committed before directory sync: %d", chunkCount)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			t.Fatalf("read chunk directory during sync: %v", err)
		}
		if len(entries) != 1 || strings.HasPrefix(entries[0].Name(), ".crack-upload-") {
			t.Fatalf("published chunk directory entries during sync = %#v", entries)
		}
		return injectedErr
	}
	t.Cleanup(func() { syncCrackFileChunkDirectory = originalSync })
	if _, err := rpcServer.CrackFileChunkUpload(t.Context(), &clientpb.CrackFileChunk{CrackFileID: crackFile.ID, N: 0, Data: []byte("data")}); status.Code(err) != codes.Internal {
		t.Fatalf("directory sync failure error = %v, want Internal", err)
	}
	if syncCalls != 1 {
		t.Fatalf("directory sync calls = %d, want 1", syncCalls)
	}
	stored, err := db.GetByCrackFileByID(crackFile.ID)
	if err != nil {
		t.Fatalf("load upload after sync failure: %v", err)
	}
	if stored.CompressedSize != 0 || len(stored.Chunks) != 0 {
		t.Fatalf("sync failure committed manifest state: %#v", stored)
	}
	entries, err := os.ReadDir(assets.GetChunkDataDir())
	if err != nil {
		t.Fatalf("read chunk directory after sync failure: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("sync failure left chunk files: %#v", entries)
	}
}

func TestCrackFileRPCsRejectNilRequests(t *testing.T) {
	setupCrackstationRPCTestDB(t)
	rpcServer := &Server{}
	assertInvalid := func(name string, err error) {
		t.Helper()
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("%s error = %v, want InvalidArgument", name, err)
		}
	}
	_, err := rpcServer.CrackFilesList(t.Context(), nil)
	assertInvalid("list", err)
	_, err = rpcServer.CrackFileCreate(t.Context(), nil)
	assertInvalid("create", err)
	_, err = rpcServer.CrackFileChunkUpload(t.Context(), nil)
	assertInvalid("upload", err)
	_, err = rpcServer.CrackFileComplete(t.Context(), nil)
	assertInvalid("complete", err)
	_, err = rpcServer.CrackFileChunkDownload(t.Context(), nil)
	assertInvalid("download", err)
	_, err = rpcServer.CrackFileDelete(t.Context(), nil)
	assertInvalid("delete", err)
}
