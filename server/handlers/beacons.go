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
			ID:                uuid.FromStringOrNil(beaconReg.ID),
			Name:              beaconReg.Register.Name,
			Hostname:          beaconReg.Register.Hostname,
			UUID:              uuid.FromStringOrNil(beaconReg.Register.Uuid),
			Username:          beaconReg.Register.Username,
			UID:               beaconReg.Register.Uid,
			GID:               beaconReg.Register.Gid,
			OS:                beaconReg.Register.Os,
			Arch:              beaconReg.Register.Arch,
			Transport:         implantConn.Transport,
			RemoteAddress:     implantConn.RemoteAddress,
			PID:               beaconReg.Register.Pid,
			Filename:          beaconReg.Register.Filename,
			LastCheckin:       implantConn.LastMessage,
			Version:           beaconReg.Register.Version,
			ReconnectInterval: beaconReg.Register.ReconnectInterval,
			ProxyURL:          beaconReg.Register.ProxyURL,
			PollTimeout:       beaconReg.Register.PollTimeout,
			ConfigID:          uuid.FromStringOrNil(beaconReg.Register.ConfigID),

			Interval:    beaconReg.Interval,
			Jitter:      beaconReg.Jitter,
			NextCheckin: beacon.NextCheckin,
		}
	} else {
		// Found existing Beacon, update specific values if they've changed
		if beaconReg.NextCheckin != beacon.NextCheckin {
			beacon.NextCheckin = beaconReg.NextCheckin
		}
		if implantConn.Transport != beacon.Transport {
			beacon.Transport = implantConn.Transport
		}
		if implantConn.RemoteAddress != beacon.RemoteAddress {
			beacon.RemoteAddress = implantConn.RemoteAddress
		}
	}
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
}
