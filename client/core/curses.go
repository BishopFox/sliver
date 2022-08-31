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
	BindTCPPort       int
	Platform          string
	ChromeExePath     string
	ChromeUserDataDir string
}

func (c *CursedProcess) DebugURL() *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", c.BindTCPPort),
		Path:   "/json",
	}
}
