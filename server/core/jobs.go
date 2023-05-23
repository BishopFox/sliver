package core

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var (
	// Jobs - Holds pointers to all the current jobs
	Jobs = &jobs{
		// ID -> *Job
		active: &sync.Map{},
	}
	jobID = 0
)

// Job - Manages background jobs
type Job struct {
	ID           int
	Name         string
	Description  string
	Protocol     string
	Port         uint16
	Domains      []string
	JobCtrl      chan bool
	PersistentID string
	ProfileName  string
}

// ToProtobuf - Get the protobuf version of the object
func (j *Job) ToProtobuf() *clientpb.Job {
	return &clientpb.Job{
		ID:          uint32(j.ID),
		Name:        j.Name,
		Description: j.Description,
		Protocol:    j.Protocol,
		Port:        uint32(j.Port),
		Domains:     j.Domains,
		ProfileName: j.ProfileName,
	}
}

// jobs - Holds refs to all active jobs
type jobs struct {
	active *sync.Map
}

// All - Return a list of all jobs
func (j *jobs) All() []*Job {
	all := []*Job{}
	j.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Job))
		return true
	})
	return all
}

// Add - Add a job to the hive (atomically)
func (j *jobs) Add(job *Job) {
	j.active.Store(job.ID, job)
	EventBroker.Publish(Event{
		Job:       job,
		EventType: consts.JobStartedEvent,
	})
}

// Remove - Remove a job
func (j *jobs) Remove(job *Job) {
	_, ok := j.active.LoadAndDelete(job.ID)
	if ok {
		EventBroker.Publish(Event{
			Job:       job,
			EventType: consts.JobStoppedEvent,
		})
	}
}

// Get - Get a Job
func (j *jobs) Get(jobID int) *Job {
	if jobID <= 0 {
		return nil
	}
	val, ok := j.active.Load(jobID)
	if ok {
		return val.(*Job)
	}
	return nil
}

// NextJobID - Returns an incremental nonce as an id
func NextJobID() int {
	newID := jobID + 1
	jobID++
	return newID
}
