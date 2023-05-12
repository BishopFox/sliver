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
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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
	"github.com/bishopfox/sliver/util"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	crackRpcLog = log.NamedLogger("rpc", "crackstations")
)

func (rpc *Server) Crackstations(ctx context.Context, req *commonpb.Empty) (*clientpb.Crackstations, error) {
	crackstations := core.AllCrackstations()
	for _, crackstation := range crackstations {
		crackstation.Benchmarks = map[int32]uint64{}
		dbCrackstation, err := db.CrackstationByHostUUID(crackstation.HostUUID)
		if err != nil {
			crackRpcLog.Errorf("Failed to get crackstation by host UUID: %s", err)
			return nil, status.Errorf(codes.NotFound, "Failed to find crackstation by host UUID")
		}
		for _, benchmark := range dbCrackstation.Benchmarks {
			crackstation.Benchmarks[benchmark.HashType] = benchmark.PerSecondRate
		}
	}
	return &clientpb.Crackstations{Crackstations: crackstations}, nil
}

func (rpc *Server) CrackstationTrigger(ctx context.Context, req *clientpb.Event) (*commonpb.Empty, error) {
	switch req.EventType {

	case consts.CrackStatusEvent:
		statusUpdate := &clientpb.CrackstationStatus{}
		err := proto.Unmarshal(req.Data, statusUpdate)
		if err != nil {
			crackRpcLog.Errorf("Failed to unmarshal crackstation status update: %s", err)
			return nil, status.Errorf(codes.InvalidArgument, "Failed to unmarshal status update")
		}
		crackStation := core.GetCrackstation(statusUpdate.HostUUID)
		if crackStation == nil {
			crackRpcLog.Errorf("Received status update for unknown crackstation: %s", statusUpdate.Name)
			return nil, status.Errorf(codes.InvalidArgument, "Unknown crackstation")
		}
		crackStation.UpdateStatus(statusUpdate)

	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackTaskByID(ctx context.Context, req *clientpb.CrackTask) (*clientpb.CrackTask, error) {
	task, err := db.GetCrackTaskByID(req.ID)
	if err != nil {
		crackRpcLog.Errorf("Failed to get crack task by ID: %s", err)
		return nil, status.Errorf(codes.NotFound, "Failed to get crack task by ID")
	}
	return task.ToProtobuf(), nil
}

func (rpc *Server) CrackTaskUpdate(ctx context.Context, req *clientpb.CrackTask) (*commonpb.Empty, error) {
	taskUpdate := models.CrackTask{}.FromProtobuf(req)
	err := db.Session().Save(&taskUpdate).Error
	if err != nil {
		crackRpcLog.Errorf("Failed to update crack task: %s", err)
		return nil, status.Errorf(codes.Internal, "Failed to update crack task")
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackstationBenchmark(ctx context.Context, req *clientpb.CrackBenchmark) (*commonpb.Empty, error) {
	hostUUID := uuid.FromStringOrNil(req.HostUUID)
	if hostUUID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid host uuid")
	}
	crackstation, err := db.CrackstationByHostUUID(req.HostUUID)
	if err != nil {
		crackRpcLog.Errorf("Failed to get crackstation by host UUID: %s", err)
		return nil, status.Errorf(codes.NotFound, "Failed to find crackstation by host UUID")
	}
	crackstation.Benchmarks = []models.Benchmark{}
	for hashType, speed := range req.Benchmarks {
		crackstation.Benchmarks = append(crackstation.Benchmarks, models.Benchmark{HashType: hashType, PerSecondRate: speed})
	}
	dbSession := db.Session()
	err = dbSession.Save(&crackstation).Error
	if err != nil {
		crackRpcLog.Errorf("Failed to save crackstation benchmarks: %s", err)
		return nil, status.Errorf(codes.Internal, "Failed to save crackstation benchmarks")
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackstationRegister(req *clientpb.Crackstation, stream rpcpb.SliverRPC_CrackstationRegisterServer) error {
	hostUUID := uuid.FromStringOrNil(req.HostUUID)
	if hostUUID == uuid.Nil {
		return status.Error(codes.InvalidArgument, "invalid host uuid")
	}
	crackStation := core.NewCrackstation(req)
	err := core.AddCrackstation(crackStation)
	if err == core.ErrDuplicateHosts {
		status.Error(codes.AlreadyExists, "crackstation already running on host")
	}
	if err != nil {
		return err
	}

	dbCrackstation, err := db.CrackstationByHostUUID(req.HostUUID)
	if err != nil {
		crackRpcLog.Infof("Registering new crackstation: %s (%s)", req.Name, hostUUID.String())
		dbSession := db.Session()
		dbSession.Create(&models.Crackstation{ID: hostUUID})
		dbCrackstation, err = db.CrackstationByHostUUID(req.HostUUID)
	}
	if err != nil {
		crackRpcLog.Errorf("Failed to query crackstation record: %s", err)
		return status.Error(codes.Internal, "failed to register crackstation")
	}

	crackRpcLog.Infof("Crackstation %s (%s) connected", req.Name, req.OperatorName)
	events := core.EventBroker.Subscribe()
	defer func() {
		crackRpcLog.Infof("Crackstation %s disconnected", req.Name)
		core.EventBroker.Unsubscribe(events)
		core.RemoveCrackstation(req.HostUUID)
	}()

	if len(dbCrackstation.Benchmarks) == 0 {
		crackRpcLog.Infof("No benchmark information for '%s', requesting benchmark...", req.Name)
		err = stream.Send(&clientpb.Event{EventType: consts.CrackBenchmark, Data: []byte{}})
		if err != nil {
			crackRpcLog.Errorf("Failed to send benchmark task to crackstation: %s", err)
			return status.Error(codes.Internal, "failed to send benchmark task")
		}
	}

	// Only forward these event types
	crackingEvents := []string{
		consts.CrackBenchmark,
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case msg := <-crackStation.Events: // This event stream is specific to this crackstation
			err := stream.Send(msg)
			if err != nil {
				crackRpcLog.Warnf(err.Error())
				return err
			}
		case event := <-events: // All server-side events
			if !util.Contains(crackingEvents, event.EventType) {
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
				crackRpcLog.Warnf(err.Error())
				return err
			}
		}
	}
}

// ----------------------------------------------------------------------------------
// CrackFile APIs - Synchronize wordlists, rules, etc. with all the crackstation(s)
// ----------------------------------------------------------------------------------
func (rpc *Server) CrackFilesList(ctx context.Context, req *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
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
		crackRpcLog.Errorf("Failed to query crack files: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack files")
	}

	crackCfg, _ := configs.LoadCrackConfig()
	currentUsage, err := db.CrackFilesDiskUsage()
	if err != nil {
		crackRpcLog.Errorf("Failed to query crack file usage: %s", err)
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
	if len(req.Name) < 1 || 64 < len(req.Name) {
		return nil, status.Error(codes.InvalidArgument, "invalid name length")
	}
	duplicateCrackFile, err := db.CrackWordlistByName(req.Name)
	if err != db.ErrRecordNotFound || duplicateCrackFile != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("duplicate name '%s'", req.Name))
	}
	usage, err := db.CrackFilesDiskUsage()
	if err != nil {
		crackRpcLog.Errorf("Failed to query crack files' disk quota: %s", err)
		return nil, status.Error(codes.Internal, "failed to query crack files' disk quota")
	}

	// Slight TOCTOU here, but disk limit is a soft limit
	crackCfg, _ := configs.LoadCrackConfig()
	if req.UncompressedSize < 1 || crackCfg.MaxFileSize < req.UncompressedSize {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid file size %d", req.UncompressedSize))
	}
	if crackCfg.MaxDiskUsage < usage+req.UncompressedSize {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("disk quota exceeded: %d/%d", usage+req.UncompressedSize, crackCfg.MaxDiskUsage))
	}

	newCrackFile := &models.CrackFile{
		Name:             req.Name,
		Type:             int32(req.Type),
		UncompressedSize: req.UncompressedSize,
		IsCompressed:     req.IsCompressed,
		IsComplete:       false,
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
	crackCfg, _ := configs.LoadCrackConfig()
	if len(req.Data) < 1 || crackCfg.ChunkSize < int64(len(req.Data)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid data size %d", len(req.Data)))
	}
	crackFile, err := db.GetByCrackFileByID(req.CrackFileID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("crack file not found '%s'", req.ID))
	}
	if crackFile.MaxN(crackCfg.ChunkSize) < req.N {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid chunk number (%d of %d)", req.N, crackFile.MaxN(crackCfg.ChunkSize)))
	}
	fileChunk := &models.CrackFileChunk{
		CrackFileID: uuid.FromStringOrNil(req.CrackFileID),
		N:           req.N,
	}
	err = db.Session().Create(fileChunk).Error
	if err != nil {
		rpcLog.Errorf("Failed to create crack file chunk: %s", err)
		return nil, status.Error(codes.Internal, "failed to create crack file chunk (db)")
	}
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		rpcLog.Errorf("Failed to get chunk data directory")
		return nil, status.Error(codes.Internal, "failed to create crack file chunk (fs)")
	}
	if fileChunk.ID == uuid.Nil {
		return nil, status.Error(codes.Internal, "nil file chunk id")
	}
	chunkDataPath := filepath.Join(chunkDataDir, fileChunk.ID.String())
	err = os.WriteFile(chunkDataPath, req.Data, 0600)
	if err != nil {
		rpcLog.Errorf("Failed to write chunk data to %s: %s", chunkDataPath, err)
		return nil, status.Error(codes.Internal, "failed to create crack file chunk (fs)")
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackFileComplete(ctx context.Context, req *clientpb.CrackFile) (*commonpb.Empty, error) {
	crackFileID := uuid.FromStringOrNil(req.ID)
	if crackFileID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid crack file id")
	}
	if matched, err := regexp.MatchString(`^[a-fA-F0-9]{64}$`, req.Sha2_256); !matched || err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sha2-256")
	}
	crackFile := &models.CrackFile{ID: crackFileID}
	err := db.Session().Where(crackFile).First(crackFile).Error
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to create crack file id")
	}
	crackFile.Sha2_256 = req.Sha2_256
	crackFile.IsComplete = true
	err = db.Session().Save(crackFile).Error
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to complete crack file")
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackFileChunkDownload(ctx context.Context, req *clientpb.CrackFileChunk) (*clientpb.CrackFileChunk, error) {
	crackFile, err := db.GetByCrackFileByID(req.CrackFileID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "crack file not found")
	}
	if !crackFile.IsComplete {
		return nil, status.Error(codes.FailedPrecondition, "crack file upload is not complete")
	}
	chunkID := uuid.FromStringOrNil(req.ID)
	if chunkID == uuid.Nil {
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
	crackFileID := uuid.FromStringOrNil(req.ID)
	if crackFileID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid crack file id")
	}
	crackFile, err := db.GetByCrackFileByID(req.ID)
	if err != nil {
		rpcLog.Errorf("Failed to get crack file: %s", err)
		return nil, status.Error(codes.Internal, "failed to get crack file (db)")
	}
	chunkDataDir := assets.GetChunkDataDir()
	if chunkDataDir == "" {
		rpcLog.Errorf("Failed to get chunk data directory")
		return nil, status.Error(codes.Internal, "failed to delete crack file (fs)")
	}
	rpcLog.Infof("Deleting crack file %s with %d chunk(s)", crackFile.ID, len(crackFile.Chunks))
	for _, chunk := range crackFile.Chunks {
		rpcLog.Infof("Deleting chunk: %s", chunk.ID)
		chunkDataPath := filepath.Join(chunkDataDir, chunk.ID.String())
		err = os.Remove(chunkDataPath)
		if err != nil {
			rpcLog.Errorf("Failed to delete chunk data from %s: %s", chunkDataPath, err)
			return nil, status.Error(codes.Internal, "failed to delete crack file (fs)")
		}
		db.Session().Delete(chunk)
	}
	err = db.Session().Delete(crackFile).Error
	if err != nil {
		rpcLog.Errorf("Failed to delete crack file: %s", err)
		return nil, status.Error(codes.Internal, "failed to delete crack file (db)")
	}
	return &commonpb.Empty{}, nil
}
