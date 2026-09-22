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
	"google.golang.org/protobuf/proto"
)

var (
	// ClientID -> core.CrackStation
	crackers = &sync.Map{}

	ErrDuplicateHosts = errors.New("only one crackstation instance per host")
)

func NewCrackstation(station *clientpb.Crackstation) *Crackstation {
	return &Crackstation{
		HostUUID:    station.HostUUID,
		Station:     station,
		Events:      make(chan *clientpb.Event, 8),
		stationLock: &sync.RWMutex{},
		statusLock:  &sync.RWMutex{},
	}
}

type Crackstation struct {
	HostUUID string
	Station  *clientpb.Crackstation
	Events   chan *clientpb.Event

	stationLock *sync.RWMutex
	status      *clientpb.CrackstationStatus
	statusLock  *sync.RWMutex
}

func (c *Crackstation) UpdateStatus(status *clientpb.CrackstationStatus) {
	c.statusLock.Lock()
	defer c.statusLock.Unlock()
	if status == nil {
		c.status = nil
		return
	}
	c.status = proto.Clone(status).(*clientpb.CrackstationStatus)
}

func (c *Crackstation) GetStatus() *clientpb.CrackstationStatus {
	c.statusLock.RLock()
	defer c.statusLock.RUnlock()
	if c.status == nil {
		return nil
	}
	return proto.Clone(c.status).(*clientpb.CrackstationStatus)
}

// Snapshot returns an isolated copy of the crackstation and its current status.
func (c *Crackstation) Snapshot() *clientpb.Crackstation {
	c.stationLock.RLock()
	defer c.stationLock.RUnlock()
	station := proto.Clone(c.Station).(*clientpb.Crackstation)
	station.Status = c.GetStatus()
	return station
}

// UpdateBenchmarks replaces the crackstation's benchmark results.
func (c *Crackstation) UpdateBenchmarks(benchmarks map[int32]uint64) {
	c.stationLock.Lock()
	defer c.stationLock.Unlock()
	c.Station.Benchmarks = make(map[int32]uint64, len(benchmarks))
	for hashType, rate := range benchmarks {
		c.Station.Benchmarks[hashType] = rate
	}
}

func AddCrackstation(crack *Crackstation) error {
	station := crack.Snapshot()
	_, loaded := crackers.LoadOrStore(station.HostUUID, crack)
	if loaded {
		return ErrDuplicateHosts
	}
	EventBroker.Publish(Event{
		EventType: consts.CrackstationConnected,
		Data:      []byte(station.HostUUID),
	})
	return nil
}

func GetCrackstation(hostUUID string) *Crackstation {
	cracker, ok := crackers.Load(hostUUID)
	if !ok {
		return nil
	}
	return cracker.(*Crackstation)
}

func AllCrackstations() []*clientpb.Crackstation {
	externalCrackers := []*clientpb.Crackstation{}
	crackers.Range(func(key, value interface{}) bool {
		crackStation := value.(*Crackstation)
		externalCrackers = append(externalCrackers, crackStation.Snapshot())
		return true
	})
	return externalCrackers
}

func RemoveCrackstation(hostUUID string) {
	_, loaded := crackers.LoadAndDelete(hostUUID)
	if loaded {
		EventBroker.Publish(Event{
			EventType: consts.CrackstationDisconnected,
			Data:      []byte(hostUUID),
		})
	}
}
