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
	"os/user"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
)

// HomeDir returns the root application directory (~/.app/ by default).
// This directory can be set with the environment variable <APP>_ROOT_DIR.
// This directory is not to be confused with the ~/.app/teamserver directory
// returned by the server.TeamDir(), which is specific to the app teamserver.
func (ts *Server) HomeDir() string {
	var dir string

	// Note: very important not to combine the nested if here.
	if !ts.opts.inMemory {
		if ts.homeDir == "" {
			user, _ := user.Current()
			dir = filepath.Join(user.HomeDir, "."+ts.name)
		} else {
			dir = ts.homeDir
		}
	} else {
		dir = "." + ts.name
	}

	err := ts.fs.MkdirAll(dir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// TeamDir returns the teamserver directory of the app (named ~/.<app>/teamserver/),
// creating the directory if needed, or logging an error event if failing to create it.
// This directory is used to store teamserver certificates, database, logs, and configs.
func (ts *Server) TeamDir() string {
	dir := path.Join(ts.HomeDir(), ts.opts.teamDir)

	err := ts.fs.MkdirAll(dir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// LogsDir returns the log directory of the server (~/.app-server/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) LogsDir() string {
	logDir := path.Join(ts.TeamDir(), assets.DirLogs)

	err := ts.fs.MkdirAll(logDir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", logDir, err)
	}

	return logDir
}

// Configs returns the configs directory of the server (~/.app-server/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) ConfigsDir() string {
	logDir := path.Join(ts.TeamDir(), assets.DirConfigs)

	err := ts.fs.MkdirAll(logDir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", logDir, err)
	}

	return logDir
}

// CertificatesDir returns the directory storing users CA PEM files as backup,
// (~/.app/teamserver/certs), either on-disk or in-memory if the teamserver is.
func (ts *Server) CertificatesDir() string {
	certDir := path.Join(ts.TeamDir(), assets.DirCerts)

	err := ts.fs.MkdirAll(certDir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", certDir, err)
	}

	return certDir
}
