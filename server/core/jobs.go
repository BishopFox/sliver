package core

import "sync"

var (
	// Jobs - Holds pointers to all the current jobs
	Jobs = &jobs{
		Active: &map[int]*Job{},
		mutex:  &sync.RWMutex{},
	}
	jobID = new(int)
)

// Job - Manages background jobs
type Job struct {
	ID          int
	Name        string
	Description string
	Protocol    string
	Port        uint16
	JobCtrl     chan bool
}

// Jobs - Holds refs to all active jobs
type jobs struct {
	Active *map[int]*Job
	mutex  *sync.RWMutex
}

// AddJob - Add a job to the hive (atomically)
func (j *jobs) AddJob(job *Job) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	(*j.Active)[job.ID] = job
}

// RemoveJob - Remove a job (atomically)
func (j *jobs) RemoveJob(job *Job) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	delete((*j.Active), job.ID)
}

// GetJobID - Returns an incremental nonce as an id
func GetJobID() int {
	newID := (*jobID) + 1
	(*jobID)++
	return newID
}
