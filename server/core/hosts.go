package core

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
	"errors"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

var (
	coreLog = log.NamedLogger("core", "hosts")
)

// StartEventAutomation - Starts an event automation goroutine
func StartEventAutomation() {
	go func() {
		for event := range EventBroker.Subscribe() {
			switch event.EventType {

			case consts.BeaconRegisteredEvent:
				if event.Beacon != nil {
					hostsBeaconCallback(event.Beacon)
				}
			case consts.SessionOpenedEvent:
				if event.Session != nil {
					hostsSessionCallback(event.Session)
				}
			}

		}
	}()
}

// Triggered on new session events, checks to see if the host is in
// the database and adds it if not.
func hostsSessionCallback(session *Session) {
	coreLog.Debugf("Hosts session callback for %v", session.UUID)
	dbSession := db.Session()
	host, err := db.HostByHostUUID(session.UUID)
	coreLog.Debugf("Hosts query result: %v %v", host, err)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		coreLog.Error(err)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		coreLog.Infof("Session %v is from a new host", session.ID)
		err := dbSession.Create(&models.Host{
			HostUUID:      uuid.FromStringOrNil(session.UUID),
			Hostname:      session.Hostname,
			OSVersion:     session.OS,
			Locale:        session.Locale,
			IOCs:          []models.IOC{},
			ExtensionData: []models.ExtensionData{},
		}).Error
		if err != nil {
			coreLog.Error(err)
			return
		}
	}
}

// Triggered on new beacon events, checks to see if the host is in
// the database and adds it if not.
func hostsBeaconCallback(beacon *models.Beacon) {
	coreLog.Debugf("Hosts beacon callback for %v", beacon.UUID)
	dbSession := db.Session()
	host, err := db.HostByHostUUID(beacon.UUID.String())
	coreLog.Debugf("Hosts query result: %v %v", host, err)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		coreLog.Error(err)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		coreLog.Infof("Beacon %v is from a new host", beacon.ID)
		err := dbSession.Create(&models.Host{
			HostUUID:      uuid.FromStringOrNil(beacon.UUID.String()),
			Hostname:      beacon.Hostname,
			OSVersion:     beacon.OS,
			Locale:        beacon.Locale,
			IOCs:          []models.IOC{},
			ExtensionData: []models.ExtensionData{},
		}).Error
		if err != nil {
			coreLog.Error(err)
			return
		}
	}
}
