package core

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
	"errors"
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var (
	// ClientID -> core.CrackStation
	crackers = &sync.Map{}

	ErrDuplicateExternalCrackerName = errors.New("cracker name must be unique, this name is already in use")
)

func NewCrackstation(station *clientpb.Crackstation) *Crackstation {
	return &Crackstation{
		Station:    station,
		Events:     make(chan *clientpb.Event, 8),
		status:     clientpb.Statuses_INITIALIZING,
		statusLock: &sync.RWMutex{},
	}
}

type Crackstation struct {
	Station *clientpb.Crackstation
	Events  chan *clientpb.Event

	status     clientpb.Statuses
	statusLock *sync.RWMutex
}

func (c *Crackstation) UpdateStatus(status *clientpb.CrackstationStatus) {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()
	c.status = status.Status
}

func (c *Crackstation) GetStatus() clientpb.Statuses {
	c.statusLock.RLock()
	defer c.statusLock.RUnlock()
	return c.status
}

func AddCrackstation(crack *Crackstation) error {
	_, loaded := crackers.LoadOrStore(crack.Station.Name, crack)
	if loaded {
		return ErrDuplicateExternalCrackerName
	}
	EventBroker.Publish(Event{
		EventType: consts.CrackstationConnected,
		Data:      []byte(crack.Station.Name),
	})
	return nil
}

func GetCrackstation(crackerName string) *Crackstation {
	cracker, ok := crackers.Load(crackerName)
	if !ok {
		return nil
	}
	return cracker.(*Crackstation)
}

func AllCrackstations() []*clientpb.Crackstation {
	externalCrackers := []*clientpb.Crackstation{}
	crackers.Range(func(key, value interface{}) bool {
		crackStation := value.(*Crackstation)
		externalCrackers = append(externalCrackers, crackStation.Station)
		return true
	})
	return externalCrackers
}

func RemoveCrackstation(crackerName string) {
	_, loaded := crackers.LoadAndDelete(crackerName)
	if loaded {
		EventBroker.Publish(Event{
			EventType: consts.CrackstationDisconnected,
			Data:      []byte(crackerName),
		})
	}
}
