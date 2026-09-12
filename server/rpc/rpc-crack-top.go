package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	crackTopHistoryLimit       = 100
	crackTopCountBatch         = 500
	crackTopMaxDevicesPerTask  = 64
	crackTopMaxNumericTextSize = 128
)

// CrackTop returns the complete active cracking set together with a bounded
// history and the currently connected crackstations. The response is built
// from dedicated summary messages so it cannot expose cracking inputs,
// recovered values, raw status targets, task output, commands, or lease tokens.
func (rpc *Server) CrackTop(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
	dbSession := db.Session().WithContext(ctx)
	active, err := loadCrackTopJobs(dbSession, "completed_at = ?", time.Time{}, true, 0)
	if err != nil {
		crackCommandRPCLog.Errorf("Failed to load active crack jobs for crack top: %s", err)
		return nil, status.Error(codes.Internal, "failed to load crack top jobs")
	}
	history, err := loadCrackTopJobs(dbSession, "completed_at <> ?", time.Time{}, false, crackTopHistoryLimit)
	if err != nil {
		crackCommandRPCLog.Errorf("Failed to load crack job history for crack top: %s", err)
		return nil, status.Error(codes.Internal, "failed to load crack top jobs")
	}

	jobs := make([]models.CrackJob, 0, len(active)+len(history))
	seen := make(map[models.UUID]struct{}, len(active)+len(history))
	for _, group := range [][]models.CrackJob{active, history} {
		for index := range group {
			if _, duplicate := seen[group[index].ID]; duplicate {
				continue
			}
			seen[group[index].ID] = struct{}{}
			jobs = append(jobs, group[index])
		}
	}
	if err := populateCrackTopResultCounts(dbSession, jobs); err != nil {
		crackCommandRPCLog.Errorf("Failed to count crack top results: %s", err)
		return nil, status.Error(codes.Internal, "failed to count crack top results")
	}
	failedTaskJobs, err := crackTopFailedTaskJobs(dbSession, jobs)
	if err != nil {
		crackCommandRPCLog.Errorf("Failed to load crack top task outcomes: %s", err)
		return nil, status.Error(codes.Internal, "failed to load crack top jobs")
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID.String() > jobs[j].ID.String()
		}
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	stations := make([]*clientpb.CrackTopStation, 0)
	for _, station := range core.AllCrackstations() {
		if station != nil {
			stations = append(stations, crackTopStationToProtobuf(station))
		}
	}
	sort.Slice(stations, func(i, j int) bool {
		if stations[i].GetHostUUID() == stations[j].GetHostUUID() {
			return stations[i].GetName() < stations[j].GetName()
		}
		return stations[i].GetHostUUID() < stations[j].GetHostUUID()
	})

	snapshot := &clientpb.CrackTopSnapshot{
		Jobs:          make([]*clientpb.CrackTopJob, 0, len(jobs)),
		Crackstations: stations,
	}
	for index := range jobs {
		_, hasFailedTask := failedTaskJobs[jobs[index].ID]
		snapshot.Jobs = append(snapshot.Jobs, crackTopJobToProtobuf(&jobs[index], hasFailedTask))
	}
	snapshot.ObservedAt = time.Now().UTC().Unix()
	return snapshot, nil
}

func loadCrackTopJobs(dbSession *gorm.DB, predicate string, completedAt time.Time, includeTasks bool, limit int) ([]models.CrackJob, error) {
	query := dbSession.
		Select("id", "created_at", "updated_at", "completed_at", "err", "hashcat_version").
		Where(predicate, completedAt).
		Preload("Command", func(commandQuery *gorm.DB) *gorm.DB {
			return commandQuery.Select("id", "crack_job_id", "attack_mode", "hash_type", "hash_mode")
		})
	if includeTasks {
		query = query.Order("created_at DESC")
		query = query.Preload("Tasks", func(taskQuery *gorm.DB) *gorm.DB {
			return taskQuery.
				Select(
					"id", "crack_job_id", "crackstation_id", "created_at", "updated_at",
					"started_at", "completed_at", "lease_expires_at", "last_heartbeat_at",
					"kind", "state", "attempt", "latest_status_json", "shard_skip", "shard_limit",
				).
				Order("created_at ASC").
				Order("id ASC")
		})
	} else {
		query = query.Order("completed_at DESC")
	}
	query = query.Order("id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var jobs []models.CrackJob
	return jobs, query.Find(&jobs).Error
}

func populateCrackTopResultCounts(dbSession *gorm.DB, jobs []models.CrackJob) error {
	for offset := 0; offset < len(jobs); offset += crackTopCountBatch {
		end := min(offset+crackTopCountBatch, len(jobs))
		jobIDs := make([]models.UUID, 0, end-offset)
		for index := offset; index < end; index++ {
			jobIDs = append(jobIDs, jobs[index].ID)
		}
		var counts []struct {
			CrackJobID  models.UUID
			ResultCount uint64
		}
		if err := dbSession.Model(&models.CrackResult{}).
			Select("crack_job_id, COUNT(*) AS result_count").
			Where("crack_job_id IN ?", jobIDs).
			Group("crack_job_id").
			Scan(&counts).Error; err != nil {
			return err
		}
		countsByJobID := make(map[models.UUID]uint64, len(counts))
		for _, count := range counts {
			countsByJobID[count.CrackJobID] = count.ResultCount
		}
		for index := offset; index < end; index++ {
			jobs[index].ResultCount = countsByJobID[jobs[index].ID]
		}
	}
	return nil
}

func crackTopFailedTaskJobs(dbSession *gorm.DB, jobs []models.CrackJob) (map[models.UUID]struct{}, error) {
	failed := make(map[models.UUID]struct{})
	for offset := 0; offset < len(jobs); offset += crackTopCountBatch {
		end := min(offset+crackTopCountBatch, len(jobs))
		jobIDs := make([]models.UUID, 0, end-offset)
		for index := offset; index < end; index++ {
			jobIDs = append(jobIDs, jobs[index].ID)
		}
		var failedIDs []models.UUID
		if err := dbSession.Model(&models.CrackTask{}).
			Distinct("crack_job_id").
			Where("crack_job_id IN ? AND state = ?", jobIDs, int32(clientpb.CrackTaskState_CRACK_TASK_FAILED)).
			Pluck("crack_job_id", &failedIDs).Error; err != nil {
			return nil, err
		}
		for _, jobID := range failedIDs {
			failed[jobID] = struct{}{}
		}
	}
	return failed, nil
}

func crackTopJobToProtobuf(job *models.CrackJob, hasFailedTask bool) *clientpb.CrackTopJob {
	jobStatus := job.Status()
	if !job.CompletedAt.IsZero() && job.Err == "" && hasFailedTask {
		jobStatus = clientpb.CrackJobStatus_FAILED
	}
	result := &clientpb.CrackTopJob{
		ID:          job.ID.String(),
		CreatedAt:   job.CreatedAt.UTC().Format(time.RFC3339),
		Status:      jobStatus,
		ResultCount: job.ResultCount,
		AttackMode:  clientpb.CrackAttackMode(job.Command.AttackMode),
		HashType:    clientpb.HashType(job.Command.HashType),
		Tasks:       make([]*clientpb.CrackTopTask, 0, len(job.Tasks)),
	}
	if !job.CompletedAt.IsZero() {
		result.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	if !job.UpdatedAt.IsZero() {
		result.UpdatedAt = job.UpdatedAt.Unix()
	}
	if job.Command.HashMode != nil {
		hashMode := *job.Command.HashMode
		result.HashMode = &hashMode
	}
	for index := range job.Tasks {
		result.Tasks = append(result.Tasks, crackTopTaskToProtobuf(&job.Tasks[index], job.HashcatVersion))
	}
	return result
}

func crackTopTaskToProtobuf(task *models.CrackTask, hashcatVersion string) *clientpb.CrackTopTask {
	result := &clientpb.CrackTopTask{
		ID:         task.ID.String(),
		Kind:       clientpb.CrackTaskKind(task.Kind),
		State:      clientpb.CrackTaskState(task.State),
		Attempt:    task.Attempt,
		ShardSkip:  task.ShardSkip,
		ShardLimit: task.ShardLimit,
	}
	if task.CrackstationID != models.NilUUID() {
		result.HostUUID = task.CrackstationID.String()
	}
	if !task.CreatedAt.IsZero() {
		result.CreatedAt = task.CreatedAt.Unix()
	}
	if !task.StartedAt.IsZero() {
		result.StartedAt = task.StartedAt.Unix()
	}
	if !task.CompletedAt.IsZero() {
		result.CompletedAt = task.CompletedAt.Unix()
	}
	if !task.LeaseExpiresAt.IsZero() {
		result.LeaseExpiresAt = task.LeaseExpiresAt.Unix()
	}
	if !task.UpdatedAt.IsZero() {
		result.UpdatedAt = task.UpdatedAt.Unix()
	}
	if !task.LastHeartbeatAt.IsZero() {
		result.LastHeartbeatAt = task.LastHeartbeatAt.Unix()
	}
	parseCrackTopStatus(task.LatestStatusJSON, result)
	normalizeCrackTopProgress(task, hashcatVersion, result)
	return result
}

// Hashcat releases through v7.1.2 report progress for --skip/--limit as an
// absolute amplified candidate range. Development builds after v7.1.2 changed
// to shard-relative counters. Only strict official release strings are ordered;
// custom/git builds deliberately remain unknown instead of guessing.
func crackTopAbsoluteProgressVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) || semver.Prerelease(version) != "" || semver.Build(version) != "" || semver.Canonical(version) != version {
		return false
	}
	return semver.Compare(version, "v7.1.2") <= 0
}

func normalizeCrackTopProgress(task *models.CrackTask, hashcatVersion string, result *clientpb.CrackTopTask) {
	if task == nil || result == nil || result.GetProgressCurrent() == "" || result.GetProgressTotal() == "" {
		return
	}
	if task.ShardSkip == 0 {
		result.ProgressExact = true
		return
	}
	if task.ShardLimit == 0 || !crackTopAbsoluteProgressVersion(hashcatVersion) {
		return
	}

	current, currentOK := new(big.Int).SetString(result.GetProgressCurrent(), 10)
	total, totalOK := new(big.Int).SetString(result.GetProgressTotal(), 10)
	if !currentOK || !totalOK || total.Sign() <= 0 {
		return
	}
	skip := new(big.Int).SetUint64(task.ShardSkip)
	limit := new(big.Int).SetUint64(task.ShardLimit)
	baseTotal := new(big.Int).Add(new(big.Int).Set(skip), limit)

	// (current / total - skip / (skip + limit)) divided by
	// (limit / (skip + limit)), expressed as an exact integer ratio. This
	// remains correct when Hashcat amplifies each base restore point into many
	// candidates.
	localCurrent := new(big.Int).Sub(
		new(big.Int).Mul(current, baseTotal),
		new(big.Int).Mul(total, skip),
	)
	if localCurrent.Sign() < 0 {
		localCurrent.SetInt64(0)
	}
	localTotal := new(big.Int).Mul(total, limit)
	if localTotal.Sign() <= 0 {
		return
	}
	fraction := new(big.Rat).SetFrac(localCurrent, localTotal)
	result.ProgressCurrent = fraction.Num().String()
	result.ProgressTotal = fraction.Denom().String()
	result.ProgressExact = true
}

func crackTopStationToProtobuf(station *clientpb.Crackstation) *clientpb.CrackTopStation {
	result := &clientpb.CrackTopStation{
		ID:       station.GetID(),
		Name:     station.GetName(),
		HostUUID: station.GetHostUUID(),
	}
	if station.GetStatus() == nil {
		return result
	}
	stationStatus := station.GetStatus()
	result.StatusAvailable = true
	result.State = stationStatus.GetState()
	result.IsSyncing = stationStatus.GetIsSyncing()
	if result.Name == "" {
		result.Name = stationStatus.GetName()
	}
	if result.HostUUID == "" {
		result.HostUUID = stationStatus.GetHostUUID()
	}
	jobID := models.ParseUUIDOrNil(strings.TrimSpace(stationStatus.GetCurrentCrackJobID()))
	if jobID != models.NilUUID() {
		result.CurrentCrackJobID = jobID.String()
	}
	return result
}

func parseCrackTopStatus(raw []byte, result *clientpb.CrackTopTask) {
	if result == nil || len(bytes.TrimSpace(raw)) == 0 {
		return
	}
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		result.StatusParseError = true
		return
	}
	result.StatusAvailable = true

	if progressRaw, ok := root["progress"]; ok {
		var progress []json.RawMessage
		if err := json.Unmarshal(progressRaw, &progress); err != nil || len(progress) < 2 {
			result.StatusParseError = true
		} else {
			var currentOK, totalOK bool
			result.ProgressCurrent, currentOK = crackTopJSONUnsignedInteger(progress[0])
			result.ProgressTotal, totalOK = crackTopJSONUnsignedInteger(progress[1])
			if !currentOK || !totalOK {
				result.ProgressCurrent = ""
				result.ProgressTotal = ""
				result.StatusParseError = true
			}
		}
	}

	devicesRaw, ok := root["devices"]
	if !ok {
		return
	}
	var devices []json.RawMessage
	if err := json.Unmarshal(devicesRaw, &devices); err != nil {
		result.StatusParseError = true
		return
	}
	if len(devices) > crackTopMaxDevicesPerTask {
		result.StatusParseError = true
		devices = devices[:crackTopMaxDevicesPerTask]
	}
	result.Devices = make([]*clientpb.CrackTopDevice, 0, len(devices))
	for _, deviceRaw := range devices {
		device, parseError := parseCrackTopDevice(deviceRaw)
		if parseError {
			result.StatusParseError = true
		}
		if device != nil {
			result.Devices = append(result.Devices, device)
		}
	}
}

func parseCrackTopDevice(raw json.RawMessage) (*clientpb.CrackTopDevice, bool) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, true
	}
	device := &clientpb.CrackTopDevice{}
	parseError := false
	if value, ok := crackTopFirstJSONField(fields, "device_id", "id"); ok {
		if id, valid := crackTopJSONUint64(value); valid {
			device.ID = strconv.FormatUint(id, 10)
		}
	}
	if value, ok := fields["speed"]; ok {
		device.Speed, device.SpeedAvailable = crackTopJSONUint64(value)
		parseError = parseError || !device.SpeedAvailable
	} else {
		parseError = true
	}
	if value, ok := crackTopFirstJSONField(fields, "temp", "temperature"); ok {
		device.Temperature, device.TemperatureAvailable = crackTopJSONNonnegativeFloat(value)
	}
	if value, ok := crackTopFirstJSONField(fields, "util", "utilization"); ok {
		device.Utilization, device.UtilizationAvailable = crackTopJSONNonnegativeFloat(value)
	}
	return device, parseError
}

func crackTopFirstJSONField(fields map[string]json.RawMessage, names ...string) (json.RawMessage, bool) {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			return value, true
		}
	}
	return nil, false
}

func crackTopJSONNumericText(raw json.RawMessage) (string, bool) {
	trimmed := bytes.TrimSpace(raw)
	var text string
	if len(trimmed) >= 2 && trimmed[0] == '"' {
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return "", false
		}
		text = strings.TrimSpace(text)
	} else {
		var number json.Number
		if err := json.Unmarshal(trimmed, &number); err != nil {
			return "", false
		}
		text = number.String()
	}
	if text == "" || len(text) > crackTopMaxNumericTextSize {
		return "", false
	}
	return text, true
}

func crackTopJSONUnsignedInteger(raw json.RawMessage) (string, bool) {
	text, ok := crackTopJSONNumericText(raw)
	if !ok {
		return "", false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return "", false
		}
	}
	return text, true
}

func crackTopJSONUint64(raw json.RawMessage) (uint64, bool) {
	text, ok := crackTopJSONUnsignedInteger(raw)
	if !ok {
		return 0, false
	}
	value, err := strconv.ParseUint(text, 10, 64)
	return value, err == nil
}

func crackTopJSONNonnegativeFloat(raw json.RawMessage) (float64, bool) {
	text, ok := crackTopJSONNumericText(raw)
	if !ok {
		return 0, false
	}
	dot := false
	digits := 0
	for _, character := range text {
		switch {
		case character >= '0' && character <= '9':
			digits++
		case character == '.' && !dot && digits > 0:
			dot = true
		default:
			return 0, false
		}
	}
	if digits == 0 || strings.HasSuffix(text, ".") {
		return 0, false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}
