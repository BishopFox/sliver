package extension

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
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

const (
	Success = 0
	Failure = 1
)

var (
	extensions     map[string]Extension
	extensionLoads map[string]*extensionLoad
	extensionsMu   sync.RWMutex
)

type extensionLoad struct {
	done    chan struct{}
	err     error
	waiters int
}

type Extension interface {
	Load() error
	Call(exportName string, arguments []byte, callback func([]byte)) error
	GetID() string
	GetArch() string
}

func Add(e Extension) {
	extensionsMu.Lock()
	defer extensionsMu.Unlock()
	extensions[e.GetID()] = e
}

// Register loads an extension once per ID and publishes it only after loading
// succeeds. Concurrent registrations for the same ID wait for the in-flight
// load instead of mapping duplicate libraries and allocating duplicate callback
// trampolines.
func Register(e Extension) error {
	id := e.GetID()
	extensionsMu.Lock()
	if _, found := extensions[id]; found {
		extensionsMu.Unlock()
		return nil
	}
	if loading, found := extensionLoads[id]; found {
		loading.waiters++
		extensionsMu.Unlock()
		<-loading.done
		return loading.err
	}
	loading := &extensionLoad{done: make(chan struct{})}
	extensionLoads[id] = loading
	extensionsMu.Unlock()

	err := e.Load()
	extensionsMu.Lock()
	if err == nil {
		extensions[id] = e
	}
	loading.err = err
	close(loading.done)
	delete(extensionLoads, id)
	extensionsMu.Unlock()
	return err
}

func List() []string {
	extensionsMu.RLock()
	defer extensionsMu.RUnlock()
	var extList []string
	for id := range extensions {
		extList = append(extList, id)
	}
	return extList
}

func Run(extID string, funcName string, arguments []byte, callback func([]byte)) error {
	extensionsMu.RLock()
	ext, found := extensions[extID]
	extensionsMu.RUnlock()
	if found {
		return ext.Call(funcName, arguments, callback)
	}
	// {{if .Config.Debug}}
	extensionsMu.RLock()
	defer extensionsMu.RUnlock()
	for id, ext := range extensions {
		log.Printf("Extension '%s' (%s)", id, ext.GetArch())
	}
	//{{end}}
	return errors.New("{{if .Config.Debug}} extension not found{{end}}")
}

func init() {
	extensions = make(map[string]Extension)
	extensionLoads = make(map[string]*extensionLoad)
}
