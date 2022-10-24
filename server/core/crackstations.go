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
	// ClientID -> *clientpb.CrackStation
	crackers = &sync.Map{}

	ErrDuplicateExternalCrackerName = errors.New("cracker name must be unique, this name is already in use")
)

func AddCrackstation(cracker *clientpb.Crackstation) error {
	_, loaded := crackers.LoadOrStore(cracker.Name, cracker)
	if loaded {
		return ErrDuplicateExternalCrackerName
	}
	EventBroker.Publish(Event{
		EventType: consts.CrackstationConnected,
		Data:      []byte(cracker.Name),
	})
	return nil
}

func GetCrackstation(crackerName string) *clientpb.Crackstation {
	cracker, ok := crackers.Load(crackerName)
	if !ok {
		return nil
	}
	return cracker.(*clientpb.Crackstation)
}

func AllCrackstations() []*clientpb.Crackstation {
	externalCrackers := []*clientpb.Crackstation{}
	crackers.Range(func(key, value interface{}) bool {
		externalCrackers = append(externalCrackers, value.(*clientpb.Crackstation))
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
