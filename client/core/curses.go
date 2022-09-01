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
	"fmt"
	"net/url"
	"sync"
)

var (
	// SessionID -> CursedProcess
	CursedProcesses = &sync.Map{}
)

type CursedProcess struct {
	SessionID         string
	PID               uint32
	BindTCPPort       int
	PortFwd           *Portfwd
	Platform          string
	ExePath           string
	ChromeUserDataDir string
}

func (c *CursedProcess) DebugURL() *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", c.BindTCPPort),
		Path:   "/json",
	}
}

func CursedProcessBySessionID(sessionID string) []*CursedProcess {
	var cursedProcesses []*CursedProcess
	CursedProcesses.Range(func(key, value interface{}) bool {
		cursedProcess := value.(*CursedProcess)
		if cursedProcess.SessionID == sessionID {
			cursedProcesses = append(cursedProcesses, cursedProcess)
		}
		return true
	})
	return cursedProcesses
}

func CloseCursedProcesses(sessionID string) {
	CursedProcesses.Range(func(key, value interface{}) bool {
		cursedProcess := value.(*CursedProcess)
		if cursedProcess.SessionID == sessionID {
			defer func() {
				value, loaded := CursedProcesses.LoadAndDelete(key)
				if loaded {
					curse := value.(*CursedProcess)
					Portfwds.Remove(curse.PortFwd.ID)
				}
			}()
		}
		return true
	})
}

func CloseCursedProcessesByBindPort(sessionID string, bindPort int) {
	CursedProcesses.Range(func(key, value interface{}) bool {
		cursedProcess := value.(*CursedProcess)
		if cursedProcess.SessionID == sessionID && cursedProcess.BindTCPPort == bindPort {
			defer func() {
				value, loaded := CursedProcesses.LoadAndDelete(key)
				if loaded {
					curse := value.(*CursedProcess)
					Portfwds.Remove(curse.PortFwd.ID)
				}
			}()
		}
		return true
	})
}
