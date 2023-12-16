package client

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
	"os/user"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
)

// HomeDir returns the root application directory (~/.app/ by default).
// This directory can be set with the environment variable <APP>_ROOT_DIR.
// This directory is not to be confused with the ~/.app/teamclient directory
// returned by the client.TeamDir(), which is specific to the app teamclient.
func (tc *Client) HomeDir() string {
	var dir string

	// Note: very important not to combine the nested if here.
	if !tc.opts.inMemory {
		if tc.homeDir == "" {
			user, _ := user.Current()
			dir = filepath.Join(user.HomeDir, "."+tc.name)
		} else {
			dir = tc.homeDir
		}
	} else {
		dir = "." + tc.name
	}

	err := tc.fs.MkdirAll(dir, assets.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// TeamDir returns the teamclient directory of the app (named ~/.<app>/teamclient/),
// creating the directory if needed, or logging an error event if failing to create it.
// This directory is used to store teamclient logs and remote server configs.
func (tc *Client) TeamDir() string {
	dir := filepath.Join(tc.HomeDir(), tc.opts.teamDir)

	err := tc.fs.MkdirAll(dir, assets.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// LogsDir returns the directory of the teamclient logs (~/.app/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (tc *Client) LogsDir() string {
	logsDir := filepath.Join(tc.TeamDir(), assets.DirLogs)

	err := tc.fs.MkdirAll(logsDir, assets.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s root dir: %s", logsDir, err)
	}

	return logsDir
}

// ConfigsDir returns the path to the remote teamserver configs directory
// for this application (~/.app/teamclient/configs), creating the directory
// if needed, or logging a fatal event if failing to create it.
func (tc *Client) ConfigsDir() string {
	rootDir, _ := filepath.Abs(tc.TeamDir())
	dir := filepath.Join(rootDir, assets.DirConfigs)

	err := tc.fs.MkdirAll(dir, assets.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s configs dir: %s", dir, err)
	}

	return dir
}
