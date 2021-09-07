package handlers

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
	------------------------------------------------------------------------

	WARNING: These functions can be invoked by remote implants without user interaction

*/

import (
	"errors"

	sliverpb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

var (
	beaconHandlerLog = log.NamedLogger("handlers", "beacons")
)

func beaconRegisterHandler(implantConn *core.ImplantConnection, data []byte) {
	beaconReg := &sliverpb.BeaconRegister{}
	err := proto.Unmarshal(data, beaconReg)
	if err != nil {
		beaconHandlerLog.Errorf("Error decoding beacon registration message: %s", err)
		return
	}
	beaconHandlerLog.Infof("Beacon registration from %s", beaconReg.ID)
	beacon, err := db.BeaconByID(beaconReg.ID)
	beaconHandlerLog.Debugf("Found %v err = %s", beacon, err)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		beaconHandlerLog.Errorf("Database query error %s", err)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		beacon = &models.Beacon{
			ID: uuid.FromStringOrNil(beaconReg.ID),
		}
	}
	beacon.Name = beaconReg.Register.Name
	beacon.Hostname = beaconReg.Register.Hostname
	beacon.UUID = uuid.FromStringOrNil(beaconReg.Register.Uuid)
	beacon.Username = beaconReg.Register.Username
	beacon.UID = beaconReg.Register.Uid
	beacon.GID = beaconReg.Register.Gid
	beacon.OS = beaconReg.Register.Os
	beacon.Arch = beaconReg.Register.Arch
	beacon.Transport = implantConn.Transport
	beacon.RemoteAddress = implantConn.RemoteAddress
	beacon.PID = beaconReg.Register.Pid
	beacon.Filename = beaconReg.Register.Filename
	beacon.LastCheckin = implantConn.LastMessage
	beacon.Version = beaconReg.Register.Version
	beacon.ReconnectInterval = beaconReg.Register.ReconnectInterval
	beacon.ProxyURL = beaconReg.Register.ProxyURL
	beacon.PollTimeout = beaconReg.Register.PollTimeout
	// beacon.ConfigID = uuid.FromStringOrNil(beaconReg.Register.ConfigID)

	beacon.Interval = beaconReg.Interval
	beacon.Jitter = beaconReg.Jitter
	beacon.NextCheckin = beaconReg.NextCheckin

	err = db.Session().Save(beacon).Error
	if err != nil {
		beaconHandlerLog.Errorf("Database write %s", err)
	}
}

func beaconTasksHandler(implantConn *core.ImplantConnection, data []byte) {
	beaconTasks := &sliverpb.BeaconTasks{}
	err := proto.Unmarshal(data, beaconTasks)
	if err != nil {
		beaconHandlerLog.Errorf("Error decoding beacon tasks message: %s", err)
		return
	}

	beaconHandlerLog.Infof("Beacon tasks from %s", beaconTasks.ID)
	pendingTasks, err := db.PendingBeaconTasksByBeaconID(beaconTasks.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		beaconHandlerLog.Errorf("Beacon task database error: %s", err)
		return
	}
	tasks := []*sliverpb.Envelope{}
	for _, pendingTask := range pendingTasks {
		envelope := &sliverpb.Envelope{}
		err = proto.Unmarshal(pendingTask.Request, envelope)
		if err != nil {
			beaconHandlerLog.Errorf("Error decoding pending task: %s", err)
			continue
		}
		tasks = append(tasks, envelope)
		pendingTask.State = models.SENT
	}
	taskData, err := proto.Marshal(&sliverpb.BeaconTasks{Tasks: tasks})
	if err != nil {
		beaconHandlerLog.Errorf("Error marshaling beacon tasks message: %s", err)
		return
	}
	beaconHandlerLog.Infof("Sending %d task(s) to beacon %s", len(pendingTasks), beaconTasks.ID)
	implantConn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgBeaconTasks,
		Data: taskData,
	}
	db.Session().Save(pendingTasks)
}
