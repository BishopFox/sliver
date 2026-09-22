package rpc

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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	crackRPCLog                  = log.NamedLogger("rpc", "crackstations")
	crackFileLifecycleMu         sync.Mutex
	crackFileNow                 = time.Now
	removeCrackFileChunk         = os.Remove
	syncCrackFileChunkDirectory  = syncCrackFileDirectory
	errCrackstationOwnerMismatch = errors.New("crackstation host is owned by another client identity")
)

const (
	crackBenchmarkSchemaVersion = uint32(1)
	crackFileUploadStaleAfter   = 24 * time.Hour
	unknownHashcatVersion       = "unknown"
)

func checkedCrackFileSizeAdd(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || left > math.MaxInt64-right {
		return 0, false
	}
	return left + right, true
}

func crackFileUploadActivity(crackFile *models.CrackFile) time.Time {
	if crackFile == nil {
		return time.Time{}
	}
	if crackFile.LastModified.After(crackFile.CreatedAt) {
		return crackFile.LastModified
	}
	return crackFile.CreatedAt
}

func reapStaleCrackUploadTempFiles(now time.Time) error {
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		return errors.New("crack file chunk directory is unavailable")
	}
	entries, err := os.ReadDir(chunkDataDir)
	if err != nil {
		return err
	}
	cutoff := now.Add(-crackFileUploadStaleAfter)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".crack-upload-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		path := filepath.Join(chunkDataDir, entry.Name())
		if err := removeCrackFileChunk(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		crackRPCLog.Infof("Reaped stale crack file upload staging file %s", path)
	}
	return nil
}

func reapStaleOrphanCrackChunkFiles(now time.Time) error {
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		return errors.New("crack file chunk directory is unavailable")
	}
	var chunks []models.CrackFileChunk
	if err := db.Session().Select("id").Find(&chunks).Error; err != nil {
		return err
	}
	referenced := make(map[string]struct{}, len(chunks))
	for index := range chunks {
		referenced[chunks[index].ID.String()] = struct{}{}
	}
	entries, err := os.ReadDir(chunkDataDir)
	if err != nil {
		return err
	}
	cutoff := now.Add(-crackFileUploadStaleAfter)
	for _, entry := range entries {
		chunkID := models.ParseUUIDOrNil(entry.Name())
		if chunkID == models.NilUUID() || chunkID.String() != entry.Name() {
			continue
		}
		if _, ok := referenced[entry.Name()]; ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		path := filepath.Join(chunkDataDir, entry.Name())
		if err := removeCrackFileChunk(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		crackRPCLog.Infof("Reaped stale orphan crack file chunk %s", path)
	}
	return nil
}

// reapStaleIncompleteCrackFilesLocked releases abandoned upload reservations.
// Callers must hold crackFileLifecycleMu. Each candidate is locked and checked
// again in its deletion transaction so a concurrent uploader's activity wins.
//
//nolint:gocyclo // Filesystem staging, database reservations, and chunks are revalidated under one lifecycle lock.
func reapStaleIncompleteCrackFilesLocked(ctx context.Context, now time.Time) error {
	if err := reapStaleCrackUploadTempFiles(now); err != nil {
		return err
	}
	cutoff := now.Add(-crackFileUploadStaleAfter)
	var candidates []models.CrackFile
	if err := db.Session().WithContext(ctx).Where("is_complete = ?", false).Find(&candidates).Error; err != nil {
		return err
	}
	chunkDataDir := ""
	for index := range candidates {
		if !crackFileUploadActivity(&candidates[index]).Before(cutoff) {
			continue
		}
		var chunkPaths []string
		deleted := false
		err := db.Session().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			current := &models.CrackFile{}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Chunks").
				Where("id = ? AND is_complete = ?", candidates[index].ID, false).First(current).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if !crackFileUploadActivity(current).Before(cutoff) {
				return nil
			}
			chunks := tx.Where("crack_file_id = ?", current.ID).Delete(&models.CrackFileChunk{})
			if chunks.Error != nil {
				return chunks.Error
			}
			if chunks.RowsAffected != int64(len(current.Chunks)) {
				return fmt.Errorf("deleted %d of %d stale crack file chunk records", chunks.RowsAffected, len(current.Chunks))
			}
			file := tx.Where("id = ? AND is_complete = ?", current.ID, false).Delete(&models.CrackFile{})
			if file.Error != nil {
				return file.Error
			}
			if file.RowsAffected != 1 {
				return errors.New("stale crack file upload changed while being reaped")
			}
			if len(current.Chunks) != 0 {
				if chunkDataDir == "" {
					chunkDataDir = assets.GetChunkDataDir()
					if chunkDataDir == "" {
						return errors.New("crack file chunk directory is unavailable")
					}
				}
				for _, chunk := range current.Chunks {
					chunkPaths = append(chunkPaths, filepath.Join(chunkDataDir, chunk.ID.String()))
				}
			}
			deleted = true
			return nil
		})
		if err != nil {
			return err
		}
		if !deleted {
			continue
		}
		crackRPCLog.Infof("Reaped stale incomplete crack file upload %s", candidates[index].ID)
		for _, chunkPath := range chunkPaths {
			if err := removeCrackFileChunk(chunkPath); err != nil && !os.IsNotExist(err) {
				crackRPCLog.Warnf("Failed to clean up stale crack file chunk %s: %s", chunkPath, err)
			}
		}
	}
	return reapStaleOrphanCrackChunkFiles(now)
}

func reapStaleIncompleteCrackFiles(now time.Time) error {
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	return reapStaleIncompleteCrackFilesLocked(context.Background(), now)
}

func normalizeHashcatVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return unknownHashcatVersion
	}
	return version
}

func crackstationBenchmarksFresh(crackstation *models.Crackstation, hashcatVersion string) bool {
	if crackstation == nil || len(crackstation.Benchmarks) == 0 {
		return false
	}
	hashcatVersion = normalizeHashcatVersion(hashcatVersion)
	return crackstation.BenchmarkSchemaVersion == crackBenchmarkSchemaVersion &&
		crackstation.HashcatVersion == hashcatVersion &&
		crackstation.BenchmarkHashcatVersion == hashcatVersion
}

func registerCrackstationRecord(hostID models.UUID, operatorName string, hashcatVersion string) (*models.Crackstation, error) {
	if hostID == models.NilUUID() || operatorName == "" {
		return nil, errors.New("invalid crackstation identity")
	}
	hashcatVersion = normalizeHashcatVersion(hashcatVersion)
	err := db.Session().Transaction(func(tx *gorm.DB) error {
		candidate := &models.Crackstation{ID: hostID, OperatorName: operatorName, HashcatVersion: hashcatVersion}
		if err := tx.Omit("Tasks", "Benchmarks").Clauses(clause.OnConflict{DoNothing: true}).Create(candidate).Error; err != nil {
			return err
		}
		persisted := &models.Crackstation{}
		if err := tx.First(persisted, "id = ?", hostID).Error; err != nil {
			return err
		}
		if persisted.OperatorName == "" {
			result := tx.Model(&models.Crackstation{}).
				Where("id = ? AND (operator_name = ? OR operator_name IS NULL)", hostID, "").
				Update("operator_name", operatorName)
			if result.Error != nil {
				return result.Error
			}
			if err := tx.First(persisted, "id = ?", hostID).Error; err != nil {
				return err
			}
		}
		if persisted.OperatorName != operatorName {
			return errCrackstationOwnerMismatch
		}
		if persisted.HashcatVersion != hashcatVersion {
			result := tx.Model(&models.Crackstation{}).Where("id = ?", hostID).Update("hashcat_version", hashcatVersion)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("failed to update crackstation hashcat version")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return db.CrackstationByHostUUID(hostID.String())
}

func (rpc *Server) Crackstations(ctx context.Context, req *commonpb.Empty) (*clientpb.Crackstations, error) {
	crackstations := core.AllCrackstations()
	for _, crackstation := range crackstations {
		crackstation.Benchmarks = map[int32]uint64{}
		dbCrackstation, err := db.CrackstationByHostUUID(crackstation.HostUUID)
		if err != nil {
			crackRPCLog.Errorf("Failed to get crackstation by host UUID: %s", err)
			return nil, status.Errorf(codes.NotFound, "Failed to find crackstation by host UUID")
		}
		for _, benchmark := range dbCrackstation.Benchmarks {
			crackstation.Benchmarks[benchmark.HashType] = benchmark.PerSecondRate
		}
	}
	return &clientpb.Crackstations{Crackstations: crackstations}, nil
}

// CrackstationBenchmarks returns every benchmark snapshot cached by the
// server, including snapshots for crackstations that are currently offline.
func (rpc *Server) CrackstationBenchmarks(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackBenchmarkSnapshots, error) {
	var crackstations []models.Crackstation
	if err := db.Session().WithContext(ctx).Preload("Benchmarks").Find(&crackstations).Error; err != nil {
		crackRPCLog.Errorf("Failed to query cached crackstation benchmarks: %s", err)
		return nil, status.Error(codes.Internal, "failed to query cached crackstation benchmarks")
	}
	sort.Slice(crackstations, func(i, j int) bool {
		return crackstations[i].ID.String() < crackstations[j].ID.String()
	})

	benchmarks := &clientpb.CrackBenchmarkSnapshots{
		Snapshots: make([]*clientpb.CrackBenchmarkSnapshot, 0, len(crackstations)),
	}
	for index := range crackstations {
		crackstation := &crackstations[index]
		if len(crackstation.Benchmarks) == 0 {
			continue
		}

		hostUUID := crackstation.ID.String()
		currentHashcatVersion := crackstation.HashcatVersion
		latestBenchmark := time.Time{}
		cached := &clientpb.CrackBenchmarkSnapshot{
			HostUUID:                hostUUID,
			OperatorName:            crackstation.OperatorName,
			CurrentHashcatVersion:   currentHashcatVersion,
			BenchmarkHashcatVersion: crackstation.BenchmarkHashcatVersion,
			BenchmarkSchemaVersion:  crackstation.BenchmarkSchemaVersion,
			Benchmarks:              make(map[int32]uint64, len(crackstation.Benchmarks)),
		}
		for _, benchmark := range crackstation.Benchmarks {
			cached.Benchmarks[benchmark.HashType] = benchmark.PerSecondRate
			if benchmark.CreatedAt.After(latestBenchmark) {
				latestBenchmark = benchmark.CreatedAt
			}
		}
		if online := core.GetCrackstation(hostUUID); online != nil {
			snapshot := online.Snapshot()
			cached.Name = snapshot.Name
			cached.Online = true
			currentHashcatVersion = snapshot.HashcatVersion
			cached.CurrentHashcatVersion = currentHashcatVersion
		}
		if !latestBenchmark.IsZero() {
			cached.BenchmarkedAt = latestBenchmark.Unix()
		}
		cached.Fresh = crackstationBenchmarksFresh(crackstation, currentHashcatVersion)
		benchmarks.Snapshots = append(benchmarks.Snapshots, cached)
	}
	return benchmarks, nil
}

func (rpc *Server) authorizeCrackstation(ctx context.Context, hostUUID string) error {
	caller := rpc.getClientCommonName(ctx)
	if caller == "" {
		return status.Error(codes.Unauthenticated, "missing crackstation client identity")
	}
	station := core.GetCrackstation(hostUUID)
	if station == nil {
		return status.Error(codes.FailedPrecondition, "crackstation is not registered")
	}
	if station.Snapshot().OperatorName != caller {
		return status.Error(codes.PermissionDenied, "crackstation identity does not own host")
	}
	return nil
}

func (rpc *Server) CrackstationTrigger(ctx context.Context, req *clientpb.Event) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crackstation event")
	}
	if len(req.Data) > maxCrackStatusEventBytes {
		return nil, status.Error(codes.InvalidArgument, "crackstation event exceeds size limit")
	}
	switch req.EventType {

	case consts.CrackStatusEvent:
		if len(req.Data) > maxCrackStatusBytes {
			return nil, status.Error(codes.InvalidArgument, "crackstation status exceeds size limit")
		}
		statusUpdate := &clientpb.CrackstationStatus{}
		err := proto.Unmarshal(req.Data, statusUpdate)
		if err != nil {
			crackRPCLog.Errorf("Failed to unmarshal crackstation status update: %s", err)
			return nil, status.Errorf(codes.InvalidArgument, "Failed to unmarshal status update")
		}
		if err := rpc.authorizeCrackstation(ctx, statusUpdate.HostUUID); err != nil {
			return nil, err
		}
		crackStation := core.GetCrackstation(statusUpdate.HostUUID)
		if crackStation == nil {
			crackRPCLog.Errorf("Received status update for unknown crackstation: %s", statusUpdate.Name)
			return nil, status.Errorf(codes.InvalidArgument, "Unknown crackstation")
		}
		crackQueueMu.Lock()
		if core.GetCrackstation(statusUpdate.HostUUID) != crackStation {
			crackQueueMu.Unlock()
			return nil, status.Error(codes.Aborted, "crackstation connection changed during status update")
		}
		wasIdle := standaloneCrackstationIsIdle(crackStation.Snapshot())
		crackStation.UpdateStatus(statusUpdate)
		isIdle := standaloneCrackstationIsIdle(crackStation.Snapshot())
		// Registration, benchmark completion, and task completion already
		// attempt scheduling. Rescan on a transition into IDLE because those
		// attempts occur before the worker's deferred status update, and on a
		// completed lifecycle drain handshake. Repeated IDLE heartbeats do not
		// scan the durable queue.
		released := observeCrackstationDrainStatusLocked(crackStation)
		if shouldScheduleAfterCrackstationStatus(wasIdle, isIdle, released) {
			scheduleErr := scheduleCrackTasksLocked(time.Now())
			if scheduleErr != nil {
				crackRPCLog.Warnf("Failed to schedule work after crackstation %s status update: %s", statusUpdate.HostUUID, scheduleErr)
			}
		}
		crackQueueMu.Unlock()
	case consts.CrackTaskCancelAck:
		acknowledgement := crackTaskCancelRequest{}
		if err := json.Unmarshal(req.Data, &acknowledgement); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid crack task cancellation acknowledgement")
		}
		if err := validateCrackTaskCancelRequest(acknowledgement); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if err := rpc.authorizeCrackstation(ctx, acknowledgement.HostUUID); err != nil {
			return nil, err
		}
		crackStation := core.GetCrackstation(acknowledgement.HostUUID)
		crackQueueMu.Lock()
		released := acknowledgeCrackstationDrainLocked(crackStation, acknowledgement)
		var scheduleErr error
		if released {
			scheduleErr = scheduleCrackTasksLocked(time.Now())
		}
		crackQueueMu.Unlock()
		if scheduleErr != nil {
			crackRPCLog.Warnf("Failed to schedule work after crackstation %s cancellation acknowledgement: %s", acknowledgement.HostUUID, scheduleErr)
		}
	case consts.CrackTaskStatus:
		statusUpdate := crackTaskStatusEvent{}
		if err := json.Unmarshal(req.Data, &statusUpdate); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid crack task status update")
		}
		if err := validateCrackTaskStatusEvent(statusUpdate); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if err := rpc.authorizeCrackstation(ctx, statusUpdate.HostUUID); err != nil {
			return nil, err
		}
		crackQueueMu.Lock()
		handled, err := updateStandaloneCrackTaskStatusLocked(statusUpdate, time.Now())
		crackQueueMu.Unlock()
		if !handled {
			err = updateCrackTaskStatus(statusUpdate)
		}
		if err != nil {
			if errors.Is(err, errStaleCrackTaskAttempt) {
				return nil, status.Error(codes.Aborted, err.Error())
			}
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown crackstation event type")
	}
	return &commonpb.Empty{}, nil
}

func shouldScheduleAfterCrackstationStatus(wasIdle, isIdle, releasedDrain bool) bool {
	return releasedDrain || (isIdle && !wasIdle)
}

func (rpc *Server) CrackTaskByID(ctx context.Context, req *clientpb.CrackTask) (*clientpb.CrackTask, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack task")
	}
	// Lifecycle revocation is tracked independently of the durable task row so
	// an exact old attempt still receives Aborted if the operator immediately
	// deletes its now-terminal job. Authenticate before recording that the
	// worker observed revocation.
	if handled, err := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
		return nil, err
	}
	crackTaskAfterRevocationPrecheck()
	if task, handled, err := rpc.standaloneCrackTaskByID(ctx, req); handled {
		return task, err
	}
	task, err := db.GetCrackTaskByID(req.ID)
	if err != nil {
		if handled, revokedErr := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
			return nil, revokedErr
		}
		crackRPCLog.Errorf("Failed to get crack task by ID: %s", err)
		return nil, status.Errorf(codes.NotFound, "Failed to get crack task by ID")
	}
	if req.HostUUID == "" || req.HostUUID != task.CrackstationID.String() || req.Attempt != task.Attempt || req.LeaseToken == "" || req.LeaseToken != task.LeaseToken {
		if handled, revokedErr := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
			return nil, revokedErr
		}
		return nil, status.Error(codes.PermissionDenied, "crack task assignment does not match lease")
	}
	if task.CrackstationID == models.NilUUID() {
		return nil, status.Error(codes.FailedPrecondition, "crack task is not leased")
	}
	state := clientpb.CrackTaskState(task.State)
	if (state != clientpb.CrackTaskState_CRACK_TASK_LEASED && state != clientpb.CrackTaskState_CRACK_TASK_RUNNING) || task.LeaseExpiresAt.IsZero() || !task.LeaseExpiresAt.After(time.Now()) {
		if handled, revokedErr := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
			return nil, revokedErr
		}
		return nil, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	if err := rpc.authorizeCrackstation(ctx, task.CrackstationID.String()); err != nil {
		return nil, err
	}
	if observeRevokedCrackTaskAttempt(req) {
		return nil, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	return task.ToProtobuf(), nil
}

var crackTaskAfterRevocationPrecheck = func() {}
var crackTaskBeforeLeasedUpdate = func() {}

func (rpc *Server) rejectRevokedCrackTaskAttempt(ctx context.Context, req *clientpb.CrackTask) (bool, error) {
	if !revokedCrackTaskAttempt(req) {
		return false, nil
	}
	if err := rpc.authorizeCrackstation(ctx, req.HostUUID); err != nil {
		return true, err
	}
	if observeRevokedCrackTaskAttempt(req) {
		return true, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
	}
	return false, nil
}

func (rpc *Server) CrackTaskUpdate(ctx context.Context, req *clientpb.CrackTask) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack task")
	}
	taskID := models.ParseUUIDOrNil(req.ID)
	if taskID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid crack task id")
	}
	if handled, err := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
		return nil, err
	}
	if handled, err := rpc.standaloneCrackTaskUpdate(ctx, req); handled {
		if err != nil {
			if status.Code(err) != codes.Unknown {
				return nil, err
			}
			if errors.Is(err, errStaleCrackTaskAttempt) {
				return nil, status.Error(codes.Aborted, err.Error())
			}
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return &commonpb.Empty{}, nil
	}

	persisted, err := db.GetCrackTaskByID(req.ID)
	if err != nil {
		if handled, revokedErr := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
			return nil, revokedErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "crack task not found")
		}
		crackRPCLog.Errorf("Failed to query crack task: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack task")
	}
	if persisted.CrackstationID == models.NilUUID() || persisted.CrackstationID.String() != req.HostUUID {
		if handled, revokedErr := rpc.rejectRevokedCrackTaskAttempt(ctx, req); handled {
			return nil, revokedErr
		}
		return nil, status.Error(codes.PermissionDenied, "crack task is not assigned to host")
	}
	if err := rpc.authorizeCrackstation(ctx, req.HostUUID); err != nil {
		return nil, err
	}
	crackTaskBeforeLeasedUpdate()
	if _, err := updateLeasedCrackTask(req); err != nil {
		if observeRevokedCrackTaskAttempt(req) {
			return nil, status.Error(codes.Aborted, errStaleCrackTaskAttempt.Error())
		}
		if errors.Is(err, errStaleCrackTaskAttempt) {
			return nil, status.Error(codes.Aborted, err.Error())
		}
		crackRPCLog.Errorf("Failed to update crack task: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackstationBenchmark(ctx context.Context, req *clientpb.CrackBenchmark) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing benchmark")
	}
	hostUUID := models.ParseUUIDOrNil(req.HostUUID)
	if hostUUID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid host uuid")
	}
	if err := rpc.authorizeCrackstation(ctx, req.HostUUID); err != nil {
		return nil, err
	}
	runtimeCrackstation := core.GetCrackstation(req.HostUUID)
	if runtimeCrackstation == nil {
		return nil, status.Error(codes.FailedPrecondition, "crackstation is not registered")
	}
	hashcatVersion := runtimeCrackstation.Snapshot().HashcatVersion
	if req.SchemaVersion != crackBenchmarkSchemaVersion || req.HashcatVersion != hashcatVersion {
		return nil, status.Errorf(codes.FailedPrecondition,
			"benchmark schema/hashcat version %d/%q does not match registered version %d/%q",
			req.SchemaVersion, req.HashcatVersion, crackBenchmarkSchemaVersion, hashcatVersion)
	}
	if len(req.Benchmarks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "benchmark results cannot be empty")
	}
	for hashType, speed := range req.Benchmarks {
		if hashType < 0 || hashType == int32(clientpb.HashType_INVALID) || speed == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid benchmark result for hash mode %d", hashType)
		}
	}
	crackstation, err := db.CrackstationByHostUUID(req.HostUUID)
	if err != nil {
		crackRPCLog.Errorf("Failed to get crackstation by host UUID: %s", err)
		return nil, status.Errorf(codes.NotFound, "Failed to find crackstation by host UUID")
	}
	benchmarks := make([]models.Benchmark, 0, len(req.Benchmarks))
	for hashType, speed := range req.Benchmarks {
		benchmarks = append(benchmarks, models.Benchmark{
			CrackstationID: crackstation.ID,
			HashType:       hashType,
			PerSecondRate:  speed,
		})
	}

	dbSession := db.Session()
	err = dbSession.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("crackstation_id = ?", crackstation.ID).Delete(&models.Benchmark{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&benchmarks).Error; err != nil {
			return err
		}
		result := tx.Model(&models.Crackstation{}).Where("id = ?", crackstation.ID).Updates(map[string]interface{}{
			"hashcat_version":           hashcatVersion,
			"benchmark_hashcat_version": hashcatVersion,
			"benchmark_schema_version":  crackBenchmarkSchemaVersion,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("failed to update crackstation benchmark version")
		}
		return nil
	})
	if err != nil {
		crackRPCLog.Errorf("Failed to save crackstation benchmarks: %s", err)
		return nil, status.Errorf(codes.Internal, "Failed to save crackstation benchmarks")
	}

	runtimeCrackstation.UpdateBenchmarks(req.Benchmarks)
	_ = scheduleCrackTasks()
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackstationRegister(req *clientpb.Crackstation, stream rpcpb.SliverRPC_CrackstationRegisterServer) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "missing crackstation registration")
	}
	req = proto.Clone(req).(*clientpb.Crackstation)
	req.OperatorName = rpc.getClientCommonName(stream.Context())
	if req.OperatorName == "" {
		return status.Error(codes.Unauthenticated, "missing crackstation client identity")
	}
	hostUUID := models.ParseUUIDOrNil(req.HostUUID)
	if hostUUID == models.NilUUID() {
		return status.Error(codes.InvalidArgument, "invalid host uuid")
	}
	req.HashcatVersion = normalizeHashcatVersion(req.HashcatVersion)
	dbCrackstation, err := registerCrackstationRecord(hostUUID, req.OperatorName, req.HashcatVersion)
	if errors.Is(err, errCrackstationOwnerMismatch) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if err != nil {
		crackRPCLog.Errorf("Failed to query crackstation record: %s", err)
		return status.Error(codes.Internal, "failed to register crackstation")
	}
	crackStation := core.NewCrackstation(req)
	err = addAndRecoverCrackstation(crackStation)
	if err == core.ErrDuplicateHosts {
		return status.Error(codes.AlreadyExists, "crackstation already running on host")
	}
	if err != nil {
		crackRPCLog.Errorf("Failed to add and recover crackstation %s: %s", req.HostUUID, err)
		return status.Error(codes.Internal, "failed to recover crackstation tasks")
	}

	crackRPCLog.Infof("Crackstation %s (%s) connected", req.Name, req.OperatorName)
	events := core.EventBroker.Subscribe()
	defer func() {
		crackRPCLog.Infof("Crackstation %s disconnected", req.Name)
		core.EventBroker.Unsubscribe(events)
		core.RemoveCrackstation(req.HostUUID)
		if err := requeueCrackstationTasksForConnection(req.HostUUID, crackStation); err != nil {
			crackRPCLog.Warnf("Failed to requeue tasks for disconnected crackstation %s: %s", req.HostUUID, err)
		}
	}()
	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return rpcError(err)
	}

	crackQueueReaperStarter()
	select {
	case crackStation.Events <- &clientpb.Event{EventType: consts.CrackFileUpdated}:
	default:
	}
	if !crackstationBenchmarksFresh(dbCrackstation, req.HashcatVersion) {
		crackRPCLog.Infof("Benchmark information for '%s' is missing or stale, requesting benchmark...", req.Name)
		benchmarkRequest, err := proto.Marshal(&clientpb.CrackCommand{
			// Stored-but-stale server results require a fresh local run. When the
			// server has no results, the crackstation may reuse its local cache.
			IgnoreLocalCache: len(dbCrackstation.Benchmarks) > 0,
		})
		if err != nil {
			return status.Error(codes.Internal, "failed to encode benchmark request")
		}
		select {
		case crackStation.Events <- &clientpb.Event{EventType: consts.CrackBenchmark, Data: benchmarkRequest}:
		default:
			return status.Error(codes.ResourceExhausted, "failed to enqueue benchmark request")
		}
	}
	// Only forward these event types
	crackingEvents := []string{
		consts.CrackFileUpdated,
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case msg := <-crackStation.Events: // This event stream is specific to this crackstation
			err := stream.Send(msg)
			if err != nil {
				crackRPCLog.Warnf("Crackstation stream send failed: %s", err)
				return rpcError(err)
			}
		case event := <-events: // All server-side events
			if !slices.Contains(crackingEvents, event.EventType) {
				continue
			}

			pbEvent := &clientpb.Event{
				EventType: event.EventType,
				Data:      event.Data,
			}
			if event.Job != nil {
				pbEvent.Job = event.Job.ToProtobuf()
			}
			if event.Client != nil {
				pbEvent.Client = event.Client.ToProtobuf()
			}
			if event.Session != nil {
				pbEvent.Session = event.Session.ToProtobuf()
			}
			if event.Err != nil {
				pbEvent.Err = event.Err.Error()
			}

			err := stream.Send(pbEvent)
			if err != nil {
				crackRPCLog.Warnf("Crackstation event send failed: %s", err)
				return rpcError(err)
			}
		}
	}
}

// ----------------------------------------------------------------------------------
// CrackFile APIs - Synchronize wordlists, rules, etc. with all the crackstation(s)
// ----------------------------------------------------------------------------------
func (rpc *Server) CrackFilesList(ctx context.Context, req *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file filter")
	}
	var crackFiles []*models.CrackFile
	var err error
	if req.Type != clientpb.CrackFileType_INVALID_TYPE {
		rpcLog.Infof("Listing crack files of type %s", req.Type.String())
		crackFiles, err = db.CrackFilesByType(req.Type)
		rpcLog.Infof("Found %d of given type", len(crackFiles))
	} else {
		crackFiles, err = db.AllCrackFiles()
	}
	if err != nil {
		crackRPCLog.Errorf("Failed to query crack files: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack files")
	}

	crackCfg, _ := configs.LoadCrackConfig()
	currentUsage, err := db.CrackFilesDiskUsage()
	if err != nil {
		crackRPCLog.Errorf("Failed to query crack file usage: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack file usage")
	}
	pbCrackFiles := &clientpb.CrackFiles{
		Files:            []*clientpb.CrackFile{},
		CurrentDiskUsage: currentUsage,
		MaxDiskUsage:     crackCfg.MaxDiskUsage,
	}
	for _, crackFile := range crackFiles {
		pbCrackFiles.Files = append(pbCrackFiles.Files, crackFile.ToProtobuf())
	}
	return pbCrackFiles, nil
}

func (rpc *Server) CrackFileCreate(ctx context.Context, req *clientpb.CrackFile) (*clientpb.CrackFile, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file")
	}
	if len(req.Name) < 1 || 64 < len(req.Name) {
		return nil, status.Error(codes.InvalidArgument, "invalid name length")
	}
	if req.Type != clientpb.CrackFileType_WORDLIST && req.Type != clientpb.CrackFileType_RULES && req.Type != clientpb.CrackFileType_MARKOV_HCSTAT2 {
		return nil, status.Error(codes.InvalidArgument, "invalid crack file type")
	}
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	now := crackFileNow()
	if err := reapStaleIncompleteCrackFilesLocked(ctx, now); err != nil {
		crackRPCLog.Errorf("Failed to reap stale crack file uploads: %s", err)
		return nil, status.Error(codes.Internal, "failed to reap stale crack file uploads")
	}
	duplicateCrackFile := &models.CrackFile{}
	err := db.Session().Where("name = ? AND type = ?", req.Name, int32(req.Type)).First(duplicateCrackFile).Error
	if err == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("duplicate name '%s'", req.Name))
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.Internal, "failed to check crack file name")
	}
	usage, err := db.CrackFilesDiskUsage()
	if err != nil {
		crackRPCLog.Errorf("Failed to query crack files' disk quota: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack files' disk quota")
	}

	// Slight TOCTOU here, but disk limit is a soft limit
	crackCfg, _ := configs.LoadCrackConfig()
	if req.UncompressedSize < 1 || crackCfg.MaxFileSize < req.UncompressedSize {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid file size %d", req.UncompressedSize))
	}
	projectedUsage, ok := checkedCrackFileSizeAdd(usage, req.UncompressedSize)
	if !ok || crackCfg.MaxDiskUsage < projectedUsage {
		return nil, status.Errorf(codes.InvalidArgument, "disk quota exceeded: current=%d requested=%d max=%d", usage, req.UncompressedSize, crackCfg.MaxDiskUsage)
	}

	newCrackFile := &models.CrackFile{
		Name:             req.Name,
		Type:             int32(req.Type),
		UncompressedSize: req.UncompressedSize,
		IsCompressed:     req.IsCompressed,
		IsComplete:       false,
		LastModified:     now,
	}
	err = db.Session().Create(newCrackFile).Error
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create crack file")
	}
	pbCrackFile := newCrackFile.ToProtobuf()
	pbCrackFile.MaxFileSize = crackCfg.MaxFileSize
	pbCrackFile.ChunkSize = crackCfg.ChunkSize
	return pbCrackFile, nil
}

func (rpc *Server) CrackFileChunkUpload(ctx context.Context, req *clientpb.CrackFileChunk) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file chunk")
	}
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	crackCfg, _ := configs.LoadCrackConfig()
	if len(req.Data) < 1 || crackCfg.ChunkSize < int64(len(req.Data)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid data size %d", len(req.Data)))
	}
	crackFile, err := db.GetByCrackFileByID(req.CrackFileID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("crack file not found '%s'", req.ID))
	}
	if crackFile.IsComplete {
		return nil, status.Error(codes.FailedPrecondition, "crack file upload is already complete")
	}
	if crackFile.UncompressedSize < 1 || crackFile.CompressedSize < 0 {
		return nil, status.Error(codes.FailedPrecondition, "crack file has invalid persisted sizes")
	}
	if req.N != uint32(len(crackFile.Chunks)) {
		return nil, status.Errorf(codes.InvalidArgument, "chunk %d is not the next contiguous chunk %d", req.N, len(crackFile.Chunks))
	}
	overheadAllowance := crackFile.UncompressedSize / 100
	if overheadAllowance < 1<<20 {
		overheadAllowance = 1 << 20
	}
	maxCompressedSize, ok := checkedCrackFileSizeAdd(crackFile.UncompressedSize, overheadAllowance)
	if !ok {
		maxCompressedSize = math.MaxInt64
	}
	projectedCompressedSize, ok := checkedCrackFileSizeAdd(crackFile.CompressedSize, int64(len(req.Data)))
	if !ok {
		return nil, status.Error(codes.ResourceExhausted, "compressed crack file size exceeds int64 capacity")
	}
	if projectedCompressedSize > maxCompressedSize {
		return nil, status.Error(codes.ResourceExhausted, "compressed crack file exceeds declared size allowance")
	}
	usage, err := db.CrackFilesDiskUsage()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to query crack file disk usage")
	}
	currentCharge := crackFile.UncompressedSize
	if crackFile.CompressedSize > currentCharge {
		currentCharge = crackFile.CompressedSize
	}
	projectedCharge := crackFile.UncompressedSize
	if projectedCompressedSize > projectedCharge {
		projectedCharge = projectedCompressedSize
	}
	if usage < currentCharge {
		return nil, status.Error(codes.Internal, "crack file disk usage is inconsistent")
	}
	otherUsage := usage - currentCharge
	projectedUsage, ok := checkedCrackFileSizeAdd(otherUsage, projectedCharge)
	if !ok || projectedUsage > crackCfg.MaxDiskUsage {
		return nil, status.Error(codes.ResourceExhausted, "crack file disk quota exceeded")
	}
	uploadStartedAt := crackFileNow()
	activity := db.Session().Model(&models.CrackFile{}).
		Where("id = ? AND is_complete = ? AND compressed_size = ?", crackFile.ID, false, crackFile.CompressedSize).
		Update("last_modified", uploadStartedAt)
	if activity.Error != nil {
		return nil, status.Error(codes.Internal, "failed to reserve crack file upload")
	}
	if activity.RowsAffected != 1 {
		return nil, status.Error(codes.Aborted, "crack file upload changed concurrently")
	}
	fileChunk := &models.CrackFileChunk{
		ID:          models.NewUUID(),
		CrackFileID: models.ParseUUIDOrNil(req.CrackFileID),
		N:           req.N,
	}
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		rpcLog.Errorf("Failed to get chunk data directory")
		return nil, status.Error(codes.Internal, "failed to create crack file chunk (fs)")
	}
	chunkDataPath := filepath.Join(chunkDataDir, fileChunk.ID.String())
	temporaryFile, err := os.CreateTemp(chunkDataDir, ".crack-upload-*")
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create temporary crack file chunk")
	}
	temporaryPath := temporaryFile.Name()
	removeTemporary := true
	defer func() {
		_ = temporaryFile.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporaryFile.Chmod(0600); err != nil {
		return nil, status.Error(codes.Internal, "failed to secure temporary crack file chunk")
	}
	if _, err := temporaryFile.Write(req.Data); err != nil {
		return nil, status.Error(codes.Internal, "failed to write temporary crack file chunk")
	}
	if err := temporaryFile.Sync(); err != nil {
		return nil, status.Error(codes.Internal, "failed to sync temporary crack file chunk")
	}
	if err := temporaryFile.Close(); err != nil {
		return nil, status.Error(codes.Internal, "failed to close temporary crack file chunk")
	}
	if err := publishCrackFileChunk(temporaryPath, chunkDataPath); err != nil {
		return nil, status.Error(codes.Internal, "failed to publish crack file chunk")
	}
	removeTemporary = false
	if err := syncCrackFileChunkDirectory(chunkDataDir); err != nil {
		_ = os.Remove(chunkDataPath)
		return nil, status.Error(codes.Internal, "failed to sync published crack file chunk")
	}
	uploadCompletedAt := crackFileNow()
	err = db.Session().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(fileChunk).Error; err != nil {
			return err
		}
		result := tx.Model(&models.CrackFile{}).Where("id = ? AND is_complete = ?", crackFile.ID, false).
			Updates(map[string]interface{}{"compressed_size": projectedCompressedSize, "last_modified": uploadCompletedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("crack file upload completed concurrently")
		}
		return nil
	})
	if err != nil {
		_ = os.Remove(chunkDataPath)
		rpcLog.Errorf("Failed to persist crack file chunk: %s", err)
		return nil, status.Error(codes.Internal, "failed to create crack file chunk (fs)")
	}
	return &commonpb.Empty{}, nil
}

type sequentialCrackChunkReader struct {
	paths   []string
	current *os.File
	index   int
}

func (reader *sequentialCrackChunkReader) Read(buffer []byte) (int, error) {
	for {
		if reader.current == nil {
			if reader.index >= len(reader.paths) {
				return 0, io.EOF
			}
			file, err := os.Open(reader.paths[reader.index])
			if err != nil {
				return 0, err
			}
			reader.current = file
			reader.index++
		}
		count, err := reader.current.Read(buffer)
		if err == io.EOF {
			_ = reader.current.Close()
			reader.current = nil
			if count != 0 {
				return count, nil
			}
			continue
		}
		return count, err
	}
}

func (reader *sequentialCrackChunkReader) Close() error {
	if reader.current == nil {
		return nil
	}
	err := reader.current.Close()
	reader.current = nil
	return err
}

//nolint:gocyclo // Stream ordering, compression, size, and digest checks share one sequential-reader lifecycle.
func verifyCrackFileUpload(crackFile *models.CrackFile, expectedSHA256 string) error {
	if crackFile.UncompressedSize < 1 || crackFile.CompressedSize < 0 {
		return errors.New("crack file has invalid persisted sizes")
	}
	if len(crackFile.Chunks) == 0 {
		return errors.New("crack file has no chunks")
	}
	sort.Slice(crackFile.Chunks, func(i, j int) bool { return crackFile.Chunks[i].N < crackFile.Chunks[j].N })
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		return errors.New("crack file chunk directory is unavailable")
	}
	paths := make([]string, 0, len(crackFile.Chunks))
	compressedSize := int64(0)
	for index, chunk := range crackFile.Chunks {
		if chunk.N != uint32(index) {
			return fmt.Errorf("crack file chunks are not contiguous at %d", index)
		}
		path := filepath.Join(chunkDataDir, chunk.ID.String())
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		nextCompressedSize, ok := checkedCrackFileSizeAdd(compressedSize, fileInfo.Size())
		if !ok {
			return errors.New("compressed crack file size exceeds int64 capacity")
		}
		compressedSize = nextCompressedSize
		paths = append(paths, path)
	}
	if crackFile.CompressedSize != 0 && compressedSize != crackFile.CompressedSize {
		return fmt.Errorf("compressed size is %d, expected %d", compressedSize, crackFile.CompressedSize)
	}
	chunkReader := &sequentialCrackChunkReader{paths: paths}
	defer func() { _ = chunkReader.Close() }()
	var reader io.Reader = chunkReader
	var decoder *zstd.Decoder
	if crackFile.IsCompressed {
		var err error
		decoder, err = zstd.NewReader(reader)
		if err != nil {
			return err
		}
		defer decoder.Close()
		reader = decoder
	}
	digest := sha256.New()
	verificationLimit := crackFile.UncompressedSize
	if verificationLimit < math.MaxInt64 {
		verificationLimit++
	}
	uncompressedSize, err := io.Copy(digest, io.LimitReader(reader, verificationLimit))
	if err != nil {
		return err
	}
	if uncompressedSize != crackFile.UncompressedSize {
		return fmt.Errorf("uncompressed size is %d, expected %d", uncompressedSize, crackFile.UncompressedSize)
	}
	actualSHA256 := hex.EncodeToString(digest.Sum(nil))
	if !strings.EqualFold(actualSHA256, expectedSHA256) {
		return fmt.Errorf("sha2-256 is %s, expected %s", actualSHA256, expectedSHA256)
	}
	return nil
}

func (rpc *Server) CrackFileComplete(ctx context.Context, req *clientpb.CrackFile) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file")
	}
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	crackFileID := models.ParseUUIDOrNil(req.ID)
	if crackFileID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid crack file id")
	}
	if matched, err := regexp.MatchString(`^[a-fA-F0-9]{64}$`, req.Sha2_256); !matched || err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sha2-256")
	}
	crackFile := &models.CrackFile{ID: crackFileID}
	err := db.Session().Where(crackFile).Preload("Chunks").First(crackFile).Error
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to create crack file id")
	}
	if crackFile.IsComplete {
		if strings.EqualFold(crackFile.Sha2_256, req.Sha2_256) {
			return &commonpb.Empty{}, nil
		}
		return nil, status.Error(codes.FailedPrecondition, "crack file is already complete with a different digest")
	}
	if err := verifyCrackFileUpload(crackFile, req.Sha2_256); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	crackFile.Sha2_256 = strings.ToLower(req.Sha2_256)
	crackFile.IsComplete = true
	crackFile.LastModified = time.Now()
	result := db.Session().Model(&models.CrackFile{}).Where("id = ? AND is_complete = ?", crackFile.ID, false).Updates(map[string]interface{}{
		"sha2_256":      crackFile.Sha2_256,
		"is_complete":   true,
		"last_modified": crackFile.LastModified,
	})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, "failed to complete crack file")
	}
	if result.RowsAffected != 1 {
		return nil, status.Error(codes.Aborted, "crack file completion raced another request")
	}
	core.EventBroker.Publish(core.Event{EventType: consts.CrackFileUpdated, Data: []byte(crackFile.ID.String())})
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackFileChunkDownload(ctx context.Context, req *clientpb.CrackFileChunk) (*clientpb.CrackFileChunk, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file chunk")
	}
	crackFile, err := db.GetByCrackFileByID(req.CrackFileID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "crack file not found")
	}
	if !crackFile.IsComplete {
		return nil, status.Error(codes.FailedPrecondition, "crack file upload is not complete")
	}
	chunkID := models.ParseUUIDOrNil(req.ID)
	if chunkID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid chunk id")
	}

	fileChunk := &models.CrackFileChunk{ID: chunkID}
	err = db.Session().Where(fileChunk).First(fileChunk).Error
	if err != nil {
		rpcLog.Errorf("Failed to get crack file chunk: %s", err)
		return nil, status.Error(codes.Internal, "failed to get crack file chunk (db)")
	}
	if fileChunk.CrackFileID.String() != req.CrackFileID {
		return nil, status.Error(codes.InvalidArgument, "chunk does not belong to specified crack file")
	}

	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		rpcLog.Errorf("Failed to get chunk data directory")
		return nil, status.Error(codes.Internal, "failed to get crack file chunk (fs)")
	}
	chunkDataPath := filepath.Join(chunkDataDir, fileChunk.ID.String())
	rpcLog.Infof("Reading chunk %s data from %s", chunkID.String(), chunkDataPath)
	data, err := os.ReadFile(chunkDataPath)
	if err != nil {
		rpcLog.Errorf("Failed to read chunk data from %s: %s", chunkDataPath, err)
		return nil, status.Error(codes.Internal, "failed to get crack file chunk (fs)")
	}
	return &clientpb.CrackFileChunk{
		CrackFileID: fileChunk.CrackFileID.String(),
		N:           fileChunk.N,
		Data:        data,
	}, nil
}

func (rpc *Server) CrackFileDelete(ctx context.Context, req *clientpb.CrackFile) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing crack file")
	}
	crackFileLifecycleMu.Lock()
	defer crackFileLifecycleMu.Unlock()
	crackFileID := models.ParseUUIDOrNil(req.ID)
	if crackFileID == models.NilUUID() {
		return nil, status.Error(codes.InvalidArgument, "invalid crack file id")
	}
	crackFile, err := db.GetByCrackFileByID(req.ID)
	if err != nil {
		rpcLog.Errorf("Failed to get crack file: %s", err)
		return nil, status.Error(codes.Internal, "failed to get crack file (db)")
	}
	referenced, err := activeCrackJobReferencesManagedFile(db.Session().WithContext(ctx), crackFile)
	if err != nil {
		crackRPCLog.Errorf("Failed to check crack file references: %s", err)
		return nil, status.Error(codes.Internal, "failed to check crack file references")
	}
	if referenced {
		return nil, status.Error(codes.FailedPrecondition, "crack file is referenced by active crack work")
	}
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		rpcLog.Errorf("Failed to get chunk data directory")
		return nil, status.Error(codes.Internal, "failed to delete crack file (fs)")
	}
	chunkPaths := make([]string, 0, len(crackFile.Chunks))
	for _, chunk := range crackFile.Chunks {
		chunkPaths = append(chunkPaths, filepath.Join(chunkDataDir, chunk.ID.String()))
	}
	err = db.Session().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chunks := tx.Where("crack_file_id = ?", crackFile.ID).Delete(&models.CrackFileChunk{})
		if chunks.Error != nil {
			return chunks.Error
		}
		if chunks.RowsAffected != int64(len(crackFile.Chunks)) {
			return fmt.Errorf("deleted %d of %d crack file chunk records", chunks.RowsAffected, len(crackFile.Chunks))
		}
		file := tx.Where("id = ?", crackFile.ID).Delete(&models.CrackFile{})
		if file.Error != nil {
			return file.Error
		}
		if file.RowsAffected != 1 {
			return fmt.Errorf("deleted %d of 1 crack file records", file.RowsAffected)
		}
		return nil
	})
	if err != nil {
		rpcLog.Errorf("Failed to delete crack file: %s", err)
		return nil, status.Error(codes.Internal, "failed to delete crack file (db)")
	}
	rpcLog.Infof("Deleted crack file %s record; cleaning up %d chunk(s)", crackFile.ID, len(chunkPaths))
	for _, chunkDataPath := range chunkPaths {
		if err := removeCrackFileChunk(chunkDataPath); err != nil && !os.IsNotExist(err) {
			rpcLog.Warnf("Failed to clean up deleted crack file chunk %s: %s", chunkDataPath, err)
		}
	}
	core.EventBroker.Publish(core.Event{EventType: consts.CrackFileUpdated, Data: []byte(crackFile.ID.String())})
	return &commonpb.Empty{}, nil
}
