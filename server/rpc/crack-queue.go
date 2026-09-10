package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	crackTaskLeaseDuration      = 45 * time.Second
	crackKeyspaceLeaseDuration  = 10 * time.Minute
	crackQueueReapInterval      = 5 * time.Second
	crackFileUploadReapInterval = time.Hour
	maxCrackStatusBytes         = 1 << 20
	maxCrackRecoveredBytes      = 16 << 20
	maxCrackRecoveredTotalBytes = 64 << 20
	maxCrackOutputBytes         = 8 << 20
	maxCrackTaskUpdateBytes     = 32 << 20
	maxCrackTaskErrorBytes      = 64 << 10
	maxCrackStatusEventBytes    = maxCrackStatusBytes + (4 << 10)
)

var (
	crackQueueMu              sync.Mutex
	crackQueueReapOnce        sync.Once
	crackQueueReaperStarter   = startCrackQueueReaper
	schedulableCrackTaskKinds = []int32{
		int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		int32(clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE),
	}
)

var errStaleCrackTaskAttempt = errors.New("stale crack task attempt")

type weightedCrackstation struct {
	HostUUID string
	Rate     uint64
}

type crackShard struct {
	HostUUID string
	Skip     uint64
	Limit    uint64
}

type crackTaskStatusEvent struct {
	TaskID     string          `json:"task_id"`
	HostUUID   string          `json:"host_uuid"`
	Attempt    uint32          `json:"attempt"`
	LeaseToken string          `json:"lease_token"`
	ObservedAt int64           `json:"observed_at"`
	Status     json.RawMessage `json:"status"`
}

type crackTaskAssignment struct {
	TaskID     string `json:"task_id"`
	HostUUID   string `json:"host_uuid"`
	Attempt    uint32 `json:"attempt"`
	LeaseToken string `json:"lease_token"`
}

type recoveredCrackResult struct {
	Hash      string `json:"hash"`
	Plaintext []byte `json:"plaintext"`
}

// splitCrackKeyspace uses largest-remainder apportionment. It is deterministic,
// covers the keyspace exactly, and gives zero-rate stations a fallback weight so
// a missing benchmark does not permanently starve a connected station.
func splitCrackKeyspace(keyspace uint64, stations []weightedCrackstation) []crackShard {
	if keyspace == 0 || len(stations) == 0 {
		return nil
	}
	byHost := map[string]uint64{}
	for _, station := range stations {
		if station.HostUUID == "" {
			continue
		}
		if previous, ok := byHost[station.HostUUID]; !ok || station.Rate > previous {
			byHost[station.HostUUID] = station.Rate
		}
	}
	stations = stations[:0]
	for hostUUID, rate := range byHost {
		if rate == 0 {
			rate = 1
		}
		stations = append(stations, weightedCrackstation{HostUUID: hostUUID, Rate: rate})
	}
	sort.Slice(stations, func(i, j int) bool { return stations[i].HostUUID < stations[j].HostUUID })
	if len(stations) == 0 {
		return nil
	}

	totalWeight := new(big.Int)
	for _, station := range stations {
		totalWeight.Add(totalWeight, new(big.Int).SetUint64(station.Rate))
	}
	type allocation struct {
		station   weightedCrackstation
		limit     uint64
		remainder *big.Int
	}
	allocations := make([]allocation, 0, len(stations))
	assigned := uint64(0)
	keyspaceInt := new(big.Int).SetUint64(keyspace)
	for _, station := range stations {
		numerator := new(big.Int).Mul(keyspaceInt, new(big.Int).SetUint64(station.Rate))
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, totalWeight, remainder)
		limit := quotient.Uint64()
		assigned += limit
		allocations = append(allocations, allocation{station: station, limit: limit, remainder: remainder})
	}
	sort.SliceStable(allocations, func(i, j int) bool {
		comparison := allocations[i].remainder.Cmp(allocations[j].remainder)
		if comparison == 0 {
			return allocations[i].station.HostUUID < allocations[j].station.HostUUID
		}
		return comparison > 0
	})
	for remaining, index := keyspace-assigned, uint64(0); remaining > 0; remaining, index = remaining-1, index+1 {
		allocations[index%uint64(len(allocations))].limit++
	}
	sort.Slice(allocations, func(i, j int) bool { return allocations[i].station.HostUUID < allocations[j].station.HostUUID })

	shards := make([]crackShard, 0, len(allocations))
	skip := uint64(0)
	for _, allocation := range allocations {
		if allocation.limit == 0 {
			continue
		}
		shards = append(shards, crackShard{
			HostUUID: allocation.station.HostUUID,
			Skip:     skip,
			Limit:    allocation.limit,
		})
		skip += allocation.limit
	}
	return shards
}

func effectiveCrackRange(keyspace uint64, skip uint64, limit uint64) (uint64, uint64) {
	start := skip
	if start > keyspace {
		start = keyspace
	}
	end := keyspace
	if limit != 0 && limit < keyspace-start {
		end = start + limit
	}
	return start, end
}

func startCrackQueueReaper() {
	crackQueueReapOnce.Do(func() {
		go func() {
			queueTicker := time.NewTicker(crackQueueReapInterval)
			defer queueTicker.Stop()
			uploadTicker := time.NewTicker(crackFileUploadReapInterval)
			defer uploadTicker.Stop()
			for {
				select {
				case <-queueTicker.C:
					if err := scheduleCrackTasks(); err != nil {
						crackCommandRpcLog.Warnf("Failed to reap/schedule crack tasks: %s", err)
					}
				case now := <-uploadTicker.C:
					if err := reapStaleIncompleteCrackFiles(now); err != nil {
						crackCommandRpcLog.Warnf("Failed to reap stale crack file uploads: %s", err)
					}
				}
			}
		}()
	})
}

func crackTaskEventType(kind int32) string {
	switch clientpb.CrackTaskKind(kind) {
	case clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE:
		return consts.CrackKeyspace
	default:
		return consts.Crack
	}
}

func crackLeaseDuration(kind int32) time.Duration {
	switch clientpb.CrackTaskKind(kind) {
	case clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE:
		return crackKeyspaceLeaseDuration
	default:
		return crackTaskLeaseDuration
	}
}

func scheduleCrackTasks() error {
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	return scheduleCrackTasksLocked(time.Now())
}

func scheduleCrackTasksLocked(now time.Time) error {
	dbSession := db.Session()
	if err := dbSession.Model(&models.CrackTask{}).
		Where("kind IN ? AND state IN ? AND lease_expires_at < ?", schedulableCrackTaskKinds, []int32{
			int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
			int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		}, now).
		Updates(map[string]interface{}{
			"state":              int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
			"crackstation_id":    models.NilUUID(),
			"lease_token":        "",
			"lease_expires_at":   time.Time{},
			"last_heartbeat_at":  time.Time{},
			"started_at":         time.Time{},
			"completed_at":       time.Time{},
			"err":                "",
			"stdout":             []byte(nil),
			"stderr":             []byte(nil),
			"exit_code":          int32(0),
			"latest_status_json": []byte(nil),
			"recovered_json":     []byte(nil),
			"updated_at":         now,
		}).Error; err != nil {
		return err
	}

	online := core.AllCrackstations()
	sort.Slice(online, func(i, j int) bool { return online[i].HostUUID < online[j].HostUUID })
	if len(online) == 0 {
		return nil
	}
	available := map[string]*core.Crackstation{}
	for _, station := range online {
		if station.HostUUID == "" {
			continue
		}
		stationID := models.ParseUUIDOrNil(station.HostUUID)
		if stationID == models.NilUUID() {
			continue
		}
		dbCrackstation := &models.Crackstation{}
		if err := dbSession.Preload("Benchmarks").First(dbCrackstation, "id = ?", stationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if !crackstationBenchmarksFresh(dbCrackstation, station.HashcatVersion) {
			continue
		}
		if runtimeStation := core.GetCrackstation(station.HostUUID); runtimeStation != nil {
			available[station.HostUUID] = runtimeStation
		}
	}
	var active []models.CrackTask
	if err := dbSession.Where("kind IN ? AND state IN ?", schedulableCrackTaskKinds, []int32{
		int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
	}).Find(&active).Error; err != nil {
		return err
	}
	for _, task := range active {
		delete(available, task.CrackstationID.String())
	}

	var queued []models.CrackTask
	if err := dbSession.Model(&models.CrackTask{}).Select("crack_tasks.*").Preload("Command").
		Joins("JOIN crack_jobs ON crack_jobs.id = crack_tasks.crack_job_id").
		Where("crack_tasks.state = ? AND crack_tasks.kind IN ?", int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED), schedulableCrackTaskKinds).
		Where("crack_jobs.completed_at = ?", time.Time{}).
		Order("crack_tasks.created_at asc").Order("crack_tasks.id asc").Find(&queued).Error; err != nil {
		return err
	}
	jobVersions := make(map[models.UUID]string)
	if len(queued) != 0 {
		jobIDs := make([]models.UUID, 0, len(queued))
		seenJobIDs := make(map[models.UUID]struct{}, len(queued))
		for index := range queued {
			if _, seen := seenJobIDs[queued[index].CrackJobID]; seen {
				continue
			}
			seenJobIDs[queued[index].CrackJobID] = struct{}{}
			jobIDs = append(jobIDs, queued[index].CrackJobID)
		}
		var jobs []models.CrackJob
		if err := dbSession.Select("id", "hashcat_version").Where("id IN ?", jobIDs).Find(&jobs).Error; err != nil {
			return err
		}
		for index := range jobs {
			jobVersions[jobs[index].ID] = jobs[index].HashcatVersion
		}
	}
	for index := range queued {
		if len(available) == 0 {
			break
		}
		task := &queued[index]
		hashType := effectiveCrackHashType(&task.Command)
		requiredHashcatVersion := jobVersions[task.CrackJobID]
		hostUUID := task.CrackstationID.String()
		station := available[hostUUID]
		if station != nil {
			supported, err := crackstationSupportsHashType(dbSession, hostUUID, hashType, requiredHashcatVersion)
			if err != nil {
				return err
			}
			if !supported {
				station = nil
				hostUUID = ""
			}
		}
		if station == nil {
			hostUUID = ""
			for candidate := range available {
				supported, err := crackstationSupportsHashType(dbSession, candidate, hashType, requiredHashcatVersion)
				if err != nil {
					return err
				}
				if supported && (hostUUID == "" || candidate < hostUUID) {
					hostUUID = candidate
				}
			}
			station = available[hostUUID]
		}
		if station == nil {
			continue
		}

		leaseToken := models.NewUUID().String()
		leaseExpiresAt := now.Add(crackLeaseDuration(task.Kind))
		result := dbSession.Model(&models.CrackTask{}).
			Where("id = ? AND state = ?", task.ID, int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED)).
			Updates(map[string]interface{}{
				"crackstation_id":  hostUUID,
				"state":            int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				"attempt":          gorm.Expr("attempt + ?", 1),
				"lease_token":      leaseToken,
				"lease_expires_at": leaseExpiresAt,
				"updated_at":       now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			continue
		}
		assignmentData, err := json.Marshal(crackTaskAssignment{
			TaskID: task.ID.String(), HostUUID: hostUUID, Attempt: task.Attempt + 1, LeaseToken: leaseToken,
		})
		if err != nil {
			return err
		}
		event := &clientpb.Event{EventType: crackTaskEventType(task.Kind), Data: assignmentData}
		select {
		case station.Events <- event:
			delete(available, hostUUID)
		default:
			if err := dbSession.Model(&models.CrackTask{}).
				Where("id = ? AND lease_token = ?", task.ID, leaseToken).
				Updates(map[string]interface{}{
					"state":            int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
					"crackstation_id":  models.NilUUID(),
					"lease_token":      "",
					"lease_expires_at": time.Time{},
					"updated_at":       now,
				}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func requeueCrackstationTasks(hostUUID string) error {
	hostID := models.ParseUUIDOrNil(hostUUID)
	if hostID == models.NilUUID() {
		return nil
	}
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	now := time.Now()
	if err := db.Session().Model(&models.CrackTask{}).
		Where("crackstation_id = ? AND kind IN ? AND state IN ?", hostID, schedulableCrackTaskKinds, []int32{
			int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
			int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		}).Updates(map[string]interface{}{
		"state":              int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
		"crackstation_id":    models.NilUUID(),
		"lease_token":        "",
		"lease_expires_at":   time.Time{},
		"last_heartbeat_at":  time.Time{},
		"started_at":         time.Time{},
		"completed_at":       time.Time{},
		"err":                "",
		"stdout":             []byte(nil),
		"stderr":             []byte(nil),
		"exit_code":          int32(0),
		"latest_status_json": []byte(nil),
		"recovered_json":     []byte(nil),
		"updated_at":         now,
	}).Error; err != nil {
		return err
	}
	return scheduleCrackTasksLocked(now)
}

func crackstationSupportsHashType(tx *gorm.DB, hostUUID string, hashType int32, requiredHashcatVersion string) (bool, error) {
	id := models.ParseUUIDOrNil(hostUUID)
	if id == models.NilUUID() {
		return false, nil
	}
	runtimeCrackstation := core.GetCrackstation(hostUUID)
	if runtimeCrackstation == nil {
		return false, nil
	}
	actualHashcatVersion := normalizeHashcatVersion(runtimeCrackstation.Snapshot().HashcatVersion)
	if requiredHashcatVersion != "" && actualHashcatVersion != requiredHashcatVersion {
		return false, nil
	}
	dbCrackstation := &models.Crackstation{}
	if err := tx.Preload("Benchmarks").First(dbCrackstation, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if !crackstationBenchmarksFresh(dbCrackstation, actualHashcatVersion) {
		return false, nil
	}
	benchmark := &models.Benchmark{}
	err := tx.Where("crackstation_id = ? AND hash_type = ?", id, hashType).
		Order("created_at desc").First(benchmark).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return benchmark.PerSecondRate > 0, nil
}

func stationWeights(tx *gorm.DB, hashType int32, hashcatVersion string) ([]weightedCrackstation, error) {
	online := core.AllCrackstations()
	stations := make([]weightedCrackstation, 0, len(online))
	for _, station := range online {
		id := models.ParseUUIDOrNil(station.HostUUID)
		if id == models.NilUUID() {
			continue
		}
		actualHashcatVersion := normalizeHashcatVersion(station.HashcatVersion)
		if actualHashcatVersion != hashcatVersion {
			continue
		}
		dbCrackstation := &models.Crackstation{}
		if err := tx.Preload("Benchmarks").First(dbCrackstation, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		if !crackstationBenchmarksFresh(dbCrackstation, actualHashcatVersion) {
			continue
		}
		benchmark := &models.Benchmark{}
		err := tx.Where("crackstation_id = ? AND hash_type = ?", id, hashType).
			Order("created_at desc").First(benchmark).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if benchmark.PerSecondRate == 0 {
			continue
		}
		stations = append(stations, weightedCrackstation{HostUUID: station.HostUUID, Rate: benchmark.PerSecondRate})
	}
	return stations, nil
}

func createCrackShards(tx *gorm.DB, task *models.CrackTask, now time.Time) error {
	keyspaceText := strings.TrimSpace(task.Keyspace)
	if keyspaceText == "" {
		keyspaceText = strings.TrimSpace(string(task.Stdout))
	}
	keyspace, err := strconv.ParseUint(keyspaceText, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid keyspace %q: %w", keyspaceText, err)
	}
	job := &models.CrackJob{}
	if err := tx.Preload("Command").First(job, "id = ?", task.CrackJobID).Error; err != nil {
		return err
	}
	hashType := effectiveCrackHashType(&job.Command)
	if task.CrackstationID == models.NilUUID() {
		return errors.New("keyspace task has no crackstation")
	}
	sourceCrackstation := &models.Crackstation{}
	if err := tx.Preload("Benchmarks").First(sourceCrackstation, "id = ?", task.CrackstationID).Error; err != nil {
		return err
	}
	hashcatVersion := normalizeHashcatVersion(sourceCrackstation.HashcatVersion)
	if job.HashcatVersion != "" && job.HashcatVersion != hashcatVersion {
		return fmt.Errorf("keyspace hashcat version %q does not match pinned job version %q", hashcatVersion, job.HashcatVersion)
	}
	weights, err := stationWeights(tx, hashType, hashcatVersion)
	if err != nil {
		return err
	}
	sourceIncluded := false
	for _, weight := range weights {
		if weight.HostUUID == task.CrackstationID.String() {
			sourceIncluded = true
			break
		}
	}
	if !sourceIncluded {
		if sourceCrackstation.BenchmarkSchemaVersion != crackBenchmarkSchemaVersion || sourceCrackstation.BenchmarkHashcatVersion != hashcatVersion {
			return errors.New("keyspace crackstation benchmark is stale")
		}
		for _, benchmark := range sourceCrackstation.Benchmarks {
			if benchmark.HashType == hashType && benchmark.PerSecondRate > 0 {
				weights = append(weights, weightedCrackstation{HostUUID: task.CrackstationID.String(), Rate: benchmark.PerSecondRate})
				sourceIncluded = true
				break
			}
		}
		if !sourceIncluded {
			return fmt.Errorf("keyspace crackstation has no benchmark for hash mode %d", hashType)
		}
	}
	rangeStart, rangeEnd := effectiveCrackRange(keyspace, job.Command.Skip, job.Command.Limit)
	shards := splitCrackKeyspace(rangeEnd-rangeStart, weights)
	for index := range shards {
		shards[index].Skip += rangeStart
	}
	if rangeEnd != rangeStart && len(shards) == 0 {
		return errors.New("no crackstations are connected")
	}
	for _, shard := range shards {
		shardTask := &models.CrackTask{
			CrackJobID:     job.ID,
			CrackstationID: models.ParseUUIDOrNil(shard.HostUUID),
			Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
			State:          int32(clientpb.CrackTaskState_CRACK_TASK_QUEUED),
			ShardSkip:      shard.Skip,
			ShardLimit:     shard.Limit,
			UpdatedAt:      now,
		}
		if err := tx.Create(shardTask).Error; err != nil {
			return err
		}
		command := job.Command
		command.ID = models.NilUUID()
		command.CreatedAt = time.Time{}
		command.CrackTaskID = shardTask.ID
		command.CrackJobID = models.NilUUID()
		command.Skip = shard.Skip
		command.Limit = shard.Limit
		command.Keyspace = false
		if err := tx.Create(&command).Error; err != nil {
			return err
		}
	}
	updates := map[string]interface{}{"keyspace": keyspaceText, "hashcat_version": hashcatVersion, "updated_at": now}
	if rangeStart == rangeEnd {
		updates["completed_at"] = now
	}
	return tx.Model(job).Updates(updates).Error
}

func crackResultFingerprint(jobID models.UUID, credentialID models.UUID, hash string, plaintext []byte) string {
	digest := sha256.New()
	digest.Write(jobID[:])
	digest.Write(credentialID[:])
	digest.Write([]byte(hash))
	digest.Write([]byte{0})
	digest.Write(plaintext)
	return hex.EncodeToString(digest.Sum(nil))
}

func ingestRecoveredResults(tx *gorm.DB, task *models.CrackTask, now time.Time) ([]string, error) {
	if len(task.RecoveredJSON) == 0 {
		return nil, nil
	}
	var recovered []recoveredCrackResult
	if err := json.Unmarshal(task.RecoveredJSON, &recovered); err != nil {
		return nil, fmt.Errorf("invalid recovered results: %w", err)
	}
	job := &models.CrackJob{}
	if err := tx.Preload("Command").First(job, "id = ?", task.CrackJobID).Error; err != nil {
		return nil, err
	}
	allowedHashes := make(map[string]struct{}, len(job.Command.Hashes))
	for _, hash := range job.Command.Hashes {
		allowedHashes[hash] = struct{}{}
	}
	hashType := effectiveCrackHashType(&job.Command)
	crackedCredentialIDs := []string{}
	for _, recovery := range recovered {
		if recovery.Hash == "" {
			continue
		}
		if _, allowed := allowedHashes[recovery.Hash]; !allowed {
			return nil, fmt.Errorf("recovered hash was not submitted with crack job")
		}
		existingResult := &models.CrackResult{}
		err := tx.Where("crack_job_id = ? AND hash = ?", job.ID, recovery.Hash).First(existingResult).Error
		if err == nil {
			if !bytes.Equal(existingResult.Plaintext, recovery.Plaintext) {
				return nil, fmt.Errorf("recovered hash has conflicting plaintext")
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		encodedRecovery, err := json.Marshal(recovery)
		if err != nil {
			return nil, err
		}
		recoveryBytes := uint64(len(encodedRecovery))
		if recoveryBytes > maxCrackRecoveredTotalBytes || job.RecoveredBytes > maxCrackRecoveredTotalBytes-recoveryBytes {
			return nil, errors.New("crack job recovered results exceed cumulative size limit")
		}
		job.RecoveredBytes += recoveryBytes
		var credentials []models.Credential
		if err := tx.Where("hash = ? AND hash_type = ?", recovery.Hash, hashType).Find(&credentials).Error; err != nil {
			return nil, err
		}
		if len(credentials) == 0 {
			credentials = append(credentials, models.Credential{ID: models.NilUUID()})
		}
		for _, credential := range credentials {
			result := &models.CrackResult{
				CrackJobID:   job.ID,
				CrackTaskID:  task.ID,
				CredentialID: credential.ID,
				Hash:         recovery.Hash,
				Plaintext:    append([]byte(nil), recovery.Plaintext...),
				Fingerprint:  crackResultFingerprint(job.ID, credential.ID, recovery.Hash, recovery.Plaintext),
				CreatedAt:    now,
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "fingerprint"}}, DoNothing: true}).Create(result).Error; err != nil {
				return nil, err
			}
			if credential.ID == models.NilUUID() {
				continue
			}
			plaintext := string(recovery.Plaintext)
			if !utf8.Valid(recovery.Plaintext) || bytes.IndexByte(recovery.Plaintext, 0) >= 0 {
				plaintext = "$HEX[" + hex.EncodeToString(recovery.Plaintext) + "]"
			}
			updated := tx.Model(&models.Credential{}).
				Where("id = ? AND is_cracked = ?", credential.ID, false).
				Updates(map[string]interface{}{
					"plaintext":  plaintext,
					"is_cracked": true,
				})
			if updated.Error != nil {
				return nil, updated.Error
			}
			if updated.RowsAffected == 1 {
				crackedCredentialIDs = append(crackedCredentialIDs, credential.ID.String())
			}
		}
	}
	if err := tx.Model(&models.CrackJob{}).Where("id = ?", job.ID).Update("recovered_bytes", job.RecoveredBytes).Error; err != nil {
		return nil, err
	}
	return crackedCredentialIDs, nil
}

func updateCrackJobState(tx *gorm.DB, jobID models.UUID, now time.Time) error {
	if jobID == models.NilUUID() {
		return nil
	}
	var tasks []models.CrackTask
	if err := tx.Where("crack_job_id = ?", jobID).Find(&tasks).Error; err != nil {
		return err
	}
	hasCrackTask := false
	allCrackTasksComplete := true
	allTasksTerminal := true
	jobError := ""
	for _, task := range tasks {
		if clientpb.CrackTaskState(task.State) == clientpb.CrackTaskState_CRACK_TASK_FAILED {
			if jobError == "" {
				jobError = task.Err
			}
		}
		state := clientpb.CrackTaskState(task.State)
		if state != clientpb.CrackTaskState_CRACK_TASK_COMPLETED && state != clientpb.CrackTaskState_CRACK_TASK_FAILED && state != clientpb.CrackTaskState_CRACK_TASK_CANCELLED {
			allTasksTerminal = false
		}
		if clientpb.CrackTaskKind(task.Kind) != clientpb.CrackTaskKind_CRACK_TASK_CRACK {
			continue
		}
		hasCrackTask = true
		if clientpb.CrackTaskState(task.State) != clientpb.CrackTaskState_CRACK_TASK_COMPLETED {
			allCrackTasksComplete = false
		}
	}
	updates := map[string]interface{}{"updated_at": now}
	if jobError != "" {
		updates["err"] = jobError
	}
	if (hasCrackTask && allCrackTasksComplete) || (jobError != "" && allTasksTerminal) {
		updates["completed_at"] = now
	}
	return tx.Model(&models.CrackJob{}).Where("id = ?", jobID).Updates(updates).Error
}

func updateLeasedCrackTask(req *clientpb.CrackTask) ([]string, error) {
	if req == nil {
		return nil, errors.New("missing crack task update")
	}
	if proto.Size(req) > maxCrackTaskUpdateBytes {
		return nil, errors.New("crack task update exceeds size limit")
	}
	if len(req.Err) > maxCrackTaskErrorBytes {
		return nil, errors.New("crack task error exceeds size limit")
	}
	if len(req.Stdout) > maxCrackOutputBytes || len(req.Stderr) > maxCrackOutputBytes {
		return nil, errors.New("crack task output exceeds size limit")
	}
	if len(req.LatestStatusJSON) > maxCrackStatusBytes {
		return nil, errors.New("crack task status exceeds size limit")
	}
	if len(req.LatestStatusJSON) != 0 && !json.Valid(req.LatestStatusJSON) {
		return nil, errors.New("invalid crack task status")
	}
	if len(req.RecoveredJSON) > maxCrackRecoveredBytes {
		return nil, errors.New("crack task recovered results exceed size limit")
	}
	if req.Keyspace != "" {
		keyspace, err := strconv.ParseUint(req.Keyspace, 10, 64)
		if err != nil || strconv.FormatUint(keyspace, 10) != req.Keyspace {
			return nil, errors.New("invalid crack task keyspace")
		}
	}
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()

	now := time.Now()
	taskID := models.ParseUUIDOrNil(req.ID)
	var crackedCredentialIDs []string
	var jobID models.UUID
	err := db.Session().Transaction(func(tx *gorm.DB) error {
		task := &models.CrackTask{}
		if err := tx.Preload("Command").First(task, "id = ?", taskID).Error; err != nil {
			return err
		}
		jobID = task.CrackJobID
		if task.CrackstationID.String() != req.HostUUID || task.Attempt != req.Attempt || task.LeaseToken == "" || task.LeaseToken != req.LeaseToken {
			return errStaleCrackTaskAttempt
		}
		if task.LeaseExpiresAt.IsZero() || !task.LeaseExpiresAt.After(now) {
			return errStaleCrackTaskAttempt
		}
		state := req.State
		if req.CompletedAt != 0 {
			if req.Err != "" {
				state = clientpb.CrackTaskState_CRACK_TASK_FAILED
			} else {
				state = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
			}
		} else if req.StartedAt != 0 {
			state = clientpb.CrackTaskState_CRACK_TASK_RUNNING
		}
		if state == clientpb.CrackTaskState_CRACK_TASK_FAILED && req.Err == "" {
			req.Err = "crack task failed"
		}
		persistedState := clientpb.CrackTaskState(task.State)
		if persistedState != clientpb.CrackTaskState_CRACK_TASK_LEASED && persistedState != clientpb.CrackTaskState_CRACK_TASK_RUNNING {
			return errStaleCrackTaskAttempt
		}
		if state != clientpb.CrackTaskState_CRACK_TASK_LEASED && state != clientpb.CrackTaskState_CRACK_TASK_RUNNING && state != clientpb.CrackTaskState_CRACK_TASK_COMPLETED && state != clientpb.CrackTaskState_CRACK_TASK_FAILED {
			return fmt.Errorf("invalid crack task transition to %s", state.String())
		}
		if req.Keyspace != "" && (clientpb.CrackTaskKind(task.Kind) != clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE || state != clientpb.CrackTaskState_CRACK_TASK_COMPLETED || req.Err != "") {
			return errors.New("crack task keyspace is only valid on a successfully completed keyspace task")
		}
		updates := map[string]interface{}{
			"state":              int32(state),
			"err":                req.Err,
			"stdout":             append([]byte(nil), req.Stdout...),
			"stderr":             append([]byte(nil), req.Stderr...),
			"exit_code":          req.ExitCode,
			"stdout_truncated":   req.StdoutTruncated,
			"stderr_truncated":   req.StderrTruncated,
			"stdout_total_bytes": req.StdoutTotalBytes,
			"stderr_total_bytes": req.StderrTotalBytes,
			"keyspace":           req.Keyspace,
			"latest_status_json": append([]byte(nil), req.LatestStatusJSON...),
			"updated_at":         now,
		}
		if len(req.RecoveredJSON) != 0 {
			// Recovered results are a bounded, idempotent stream. Preserve the
			// latest batch for diagnostics without clearing it on the terminal
			// lifecycle update.
			updates["recovered_json"] = append([]byte(nil), req.RecoveredJSON...)
		}
		if state == clientpb.CrackTaskState_CRACK_TASK_RUNNING && task.StartedAt.IsZero() {
			updates["started_at"] = now
		}
		terminal := state == clientpb.CrackTaskState_CRACK_TASK_COMPLETED || state == clientpb.CrackTaskState_CRACK_TASK_FAILED
		if terminal {
			updates["completed_at"] = now
			updates["lease_expires_at"] = time.Time{}
		} else {
			updates["lease_expires_at"] = now.Add(crackLeaseDuration(task.Kind))
			updates["last_heartbeat_at"] = now
		}
		result := tx.Model(&models.CrackTask{}).
			Where("id = ? AND crackstation_id = ? AND attempt = ? AND lease_token = ? AND lease_expires_at > ? AND state IN ?", task.ID, task.CrackstationID, task.Attempt, task.LeaseToken, now, []int32{
				int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
			}).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errStaleCrackTaskAttempt
		}
		task.State = int32(state)
		task.Err = req.Err
		task.Stdout = append([]byte(nil), req.Stdout...)
		task.Keyspace = req.Keyspace
		if len(req.RecoveredJSON) != 0 {
			task.RecoveredJSON = append([]byte(nil), req.RecoveredJSON...)
		}
		if clientpb.CrackTaskKind(task.Kind) == clientpb.CrackTaskKind_CRACK_TASK_CRACK && len(req.RecoveredJSON) != 0 {
			// Ingest each running-task batch immediately. Fingerprints make an
			// ambiguous retry safe, and prevent a large successful multi-hash
			// shard from being trapped in an oversized terminal RPC forever.
			recoveryTask := *task
			recoveryTask.RecoveredJSON = append([]byte(nil), req.RecoveredJSON...)
			var err error
			crackedCredentialIDs, err = ingestRecoveredResults(tx, &recoveryTask, now)
			if err != nil {
				return err
			}
		}
		if terminal {
			switch clientpb.CrackTaskKind(task.Kind) {
			case clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE:
				if state != clientpb.CrackTaskState_CRACK_TASK_COMPLETED {
					break
				}
				shardErr := tx.Transaction(func(shardTx *gorm.DB) error {
					return createCrackShards(shardTx, task, now)
				})
				if shardErr != nil {
					if updateErr := tx.Model(&models.CrackJob{}).Where("id = ?", task.CrackJobID).Updates(map[string]interface{}{"err": shardErr.Error(), "completed_at": now, "updated_at": now}).Error; updateErr != nil {
						return updateErr
					}
				}
			}
		}
		return updateCrackJobState(tx, task.CrackJobID, now)
	})
	if err != nil {
		return nil, err
	}
	if jobID != models.NilUUID() {
		core.EventBroker.Publish(core.Event{EventType: consts.CrackJobUpdated, Data: []byte(jobID.String())})
	}
	for _, credentialID := range crackedCredentialIDs {
		core.EventBroker.Publish(core.Event{EventType: consts.CredentialCrackedEvent, Data: []byte(credentialID)})
	}
	if err := scheduleCrackTasksLocked(now); err != nil {
		crackCommandRpcLog.Warnf("Accepted crack task update but could not schedule follow-up work: %s", err)
	}
	return crackedCredentialIDs, nil
}

func updateCrackTaskStatus(event crackTaskStatusEvent) error {
	if event.TaskID == "" || event.HostUUID == "" || event.Attempt == 0 || event.LeaseToken == "" || len(event.Status) == 0 || len(event.Status) > maxCrackStatusBytes || !json.Valid(event.Status) {
		return errors.New("invalid crack task status event")
	}
	crackQueueMu.Lock()
	defer crackQueueMu.Unlock()
	now := time.Now()
	observedAt := event.ObservedAt
	if observedAt == 0 {
		observedAt = now.Unix()
	}
	taskID := models.ParseUUIDOrNil(event.TaskID)
	hostID := models.ParseUUIDOrNil(event.HostUUID)
	if taskID == models.NilUUID() || hostID == models.NilUUID() {
		return errors.New("invalid crack task status identity")
	}
	task := &models.CrackTask{}
	if err := db.Session().Where("id = ? AND crackstation_id = ? AND attempt = ? AND lease_token = ? AND lease_expires_at > ? AND state IN ?", taskID, hostID, event.Attempt, event.LeaseToken, now, []int32{
		int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
	}).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errStaleCrackTaskAttempt
		}
		return err
	}
	result := db.Session().Model(&models.CrackTask{}).
		Where("id = ? AND crackstation_id = ? AND attempt = ? AND lease_token = ? AND lease_expires_at > ? AND state IN ?", taskID, hostID, event.Attempt, event.LeaseToken, now, []int32{
			int32(clientpb.CrackTaskState_CRACK_TASK_LEASED), int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		}).Updates(map[string]interface{}{
		"state":              int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		"latest_status_json": []byte(event.Status),
		"last_heartbeat_at":  now,
		"lease_expires_at":   now.Add(crackLeaseDuration(task.Kind)),
		"updated_at":         now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errStaleCrackTaskAttempt
	}
	event.ObservedAt = observedAt
	event.LeaseToken = ""
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	core.EventBroker.Publish(core.Event{EventType: consts.CrackTaskStatus, Data: data})
	return nil
}
