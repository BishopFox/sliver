package server

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"fmt"
	"net"
	"sync"
)

// job - Manages background jobs.
type job struct {
	ID          string
	Name        string
	Description string
	kill        chan bool
	Persistent  bool
}

// jobs - Holds refs to all active jobs.
type jobs struct {
	active *sync.Map
}

func newJobs() *jobs {
	return &jobs{
		active: &sync.Map{},
	}
}

// Add - Add a job to the hive (atomically).
func (j *jobs) Add(listener *job) {
	j.active.Store(listener.ID, listener)
}

// Get - Get a Job.
func (j *jobs) Get(jobID string) *job {
	if jobID == "" {
		return nil
	}

	val, ok := j.active.Load(jobID)
	if ok {
		return val.(*job)
	}

	return nil
}

// Listeners returns a list of all running listener jobs.
// If you also want the list of the non-running, persistent
// ones, use the teamserver Config().
func (ts *Server) Listeners() []*job {
	all := []*job{}

	// Active listeners
	ts.jobs.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*job))
		return true
	})

	return all
}

// ListenerAdd adds a teamserver listener job to the teamserver configuration.
// This function does not start the given listener, and you must call the server
// ServeAddr(name, host, port) function for this.
func (ts *Server) ListenerAdd(name, host string, port uint16) error {
	listener := struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}{
		Name: name,
		Host: host,
		Port: port,
		ID:   getRandomID(),
	}

	if listener.Name == "" && ts.self != nil {
		listener.Name = ts.self.Name()
	}

	ts.opts.config.Listeners = append(ts.opts.config.Listeners, listener)

	return ts.SaveConfig(ts.opts.config)
}

// ListenerRemove removes a server listener job from the configuration.
// This function does not stop any running listener for the given ID: you
// must call server.CloseListener(id) for this.
func (ts *Server) ListenerRemove(listenerID string) {
	if ts.opts.config.Listeners == nil {
		return
	}

	defer ts.SaveConfig(ts.opts.config)

	var listeners []struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}

	for _, listener := range ts.opts.config.Listeners {
		if listener.ID != listenerID {
			listeners = append(listeners, listener)
		}
	}

	ts.opts.config.Listeners = listeners
}

// ListenerClose closes/stops an active teamserver listener by ID.
// This function can only return an ErrListenerNotFound if the ID
// is invalid: all listener-specific options are logged instead.
func (ts *Server) ListenerClose(id string) error {
	listener := ts.jobs.Get(id)
	if listener == nil {
		return ts.errorf("%w: %s", ErrListenerNotFound, id)
	}

	listener.kill <- true

	return nil
}

// ListenerStartPersistents attempts to start all listeners saved in the teamserver
// configuration file, looking up the listener stacks in its map and starting them
// for each bind target.
// If the teamserver has been passed the WithContinueOnError() option at some point,
// it will log all errors raised by listener stacks will still try to start them all.
func (ts *Server) ListenerStartPersistents() error {
	var listenerErrors error

	log := ts.NamedLogger("teamserver", "listeners")

	if ts.opts.config.Listeners == nil {
		return nil
	}

	for _, ln := range ts.opts.config.Listeners {
		handler := ts.handlers[ln.Name]
		if handler == nil {
			handler = ts.self
		}

		if handler == nil {
			if !ts.opts.continueOnError {
				return ts.errorf("Failed to find handler for `%s` listener (%s:%d)", ln.Name, ln.Host, ln.Port)
			}

			continue
		}

		err := ts.serve(handler, ln.ID, ln.Host, ln.Port)

		if err == nil {
			continue
		}

		log.Errorf("Failed to start %s listener (%s:%d): %s", ln.Name, ln.Host, ln.Port, err)

		if !ts.opts.continueOnError {
			return err
		}

		listenerErrors = errors.Join(listenerErrors, err)
	}

	return nil
}

func (ts *Server) addListenerJob(listenerID, name, host string, port int, ln net.Listener) {
	log := ts.NamedLogger("teamserver", "listeners")

	if listenerID == "" {
		listenerID = getRandomID()
	}

	laddr := host
	if port != 0 {
		laddr = fmt.Sprintf("%s:%d", laddr, port)
	}

	if laddr == "" {
		laddr = "runtime"
	}

	listener := &job{
		ID:          listenerID,
		Name:        name,
		Description: laddr,
		kill:        make(chan bool),
	}

	go func() {
		<-listener.kill

		// Kills listener goroutines but NOT connections.
		log.Infof("Stopping teamserver %s listener (%s)", name, listener.ID)
		ln.Close()

		ts.jobs.active.LoadAndDelete(listener.ID)
	}()

	ts.jobs.active.Store(listener.ID, listener)
}
