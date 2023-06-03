package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

var (
	beaconRpcLog = log.NamedLogger("rpc", "beacons")
)

// GetBeacons - Get a list of beacons from the database
func (rpc *Server) GetBeacons(ctx context.Context, req *commonpb.Empty) (*clientpb.Beacons, error) {
	dbBeacons, err := db.ListBeacons()
	if err != nil {
		beaconRpcLog.Errorf("Failed to find db beacons: %s", err)
		return nil, ErrDatabaseFailure
	}
	beacons := []*clientpb.Beacon{}
	for _, beacon := range dbBeacons {
		pbBeacon := beacon.ToProtobuf()
		all, completed, err := db.CountTasksByBeaconID(beacon.ID)
		if err != nil {
			beaconRpcLog.Errorf("Task count failed: %s", err)
		}
		pbBeacon.TasksCount = all
		pbBeacon.TasksCountCompleted = completed
		beacons = append(beacons, pbBeacon)
	}
	return &clientpb.Beacons{Beacons: beacons}, nil
}

// GetBeacon - Get a list of beacons from the database
func (rpc *Server) GetBeacon(ctx context.Context, req *clientpb.Beacon) (*clientpb.Beacon, error) {
	beacon, err := db.BeaconByID(req.ID)
	if err != nil {
		beaconRpcLog.Error(err)
		return nil, ErrDatabaseFailure
	}
	return beacon.ToProtobuf(), nil
}

// RmBeacon - Delete a beacon and any related tasks
func (rpc *Server) RmBeacon(ctx context.Context, req *clientpb.Beacon) (*commonpb.Empty, error) {
	beacon, err := db.BeaconByID(req.ID)
	if err != nil {
		beaconRpcLog.Error(err)
		return nil, ErrInvalidBeaconID
	}
	err = db.Session().Where(&models.BeaconTask{
		BeaconID: beacon.ID},
	).Delete(&models.BeaconTask{}).Error
	if err != nil {
		beaconRpcLog.Errorf("Database error: %s", err)
		return nil, ErrDatabaseFailure
	}
	err = db.Session().Delete(beacon).Error
	if err != nil {
		beaconRpcLog.Errorf("Database error: %s", err)
		return nil, ErrDatabaseFailure
	}
	return &commonpb.Empty{}, nil
}

// GetBeaconTasks - Get a list of tasks for a specific beacon
func (rpc *Server) GetBeaconTasks(ctx context.Context, req *clientpb.Beacon) (*clientpb.BeaconTasks, error) {
	beacon, err := db.BeaconByID(req.ID)
	if err != nil {
		return nil, ErrInvalidBeaconID
	}
	dbTasks, err := db.BeaconTasksByBeaconID(beacon.ID.String())
	tasks := []*clientpb.BeaconTask{}
	for _, task := range dbTasks {
		tasks = append(tasks, task.ToProtobuf(false))
	}
	return &clientpb.BeaconTasks{Tasks: tasks}, err
}

// GetBeaconTaskContent - Get the content of a specific task
func (rpc *Server) GetBeaconTaskContent(ctx context.Context, req *clientpb.BeaconTask) (*clientpb.BeaconTask, error) {
	task, err := db.BeaconTaskByID(req.ID)
	if err != nil {
		return nil, ErrInvalidBeaconTaskID
	}
	return task.ToProtobuf(true), nil
}

// CancelBeaconTask - Cancel a beacon task
func (rpc *Server) CancelBeaconTask(ctx context.Context, req *clientpb.BeaconTask) (*clientpb.BeaconTask, error) {
	task, err := db.BeaconTaskByID(req.ID)
	if err != nil {
		return nil, ErrInvalidBeaconTaskID
	}
	if task.State == models.PENDING {
		task.State = models.CANCELED
		err = db.Session().Save(task).Error
		if err != nil {
			beaconRpcLog.Errorf("Database error: %s", err)
			return nil, ErrDatabaseFailure
		}
	} else {
		// No real point to cancel the task if it's already been sent
		return task.ToProtobuf(false), ErrInvalidBeaconTaskCancelState
	}
	task, err = db.BeaconTaskByID(req.ID)
	if err != nil {
		return nil, ErrInvalidBeaconTaskID
	}
	return task.ToProtobuf(false), nil
}
