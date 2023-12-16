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

import "errors"

var (
	//
	// Filesystem errors.
	//

	// ErrDirectory is an error related to directories used by the teamserver.
	ErrDirectory = errors.New("teamserver directory")

	// ErrDirectoryUnwritable is an error returned when the teamserver checked for write permissions
	// on a directory path it needs, and that the Go code of the teamserver has determined that the
	// file is really non-user writable. This error is NEVER returned "because the path does not exist".
	ErrDirectoryUnwritable = errors.New("The directory seems to be unwritable (to the app runtime)")

	// ErrLogging is an error related with the logging backend.
	// Some errors can be about writable files/directories.
	ErrLogging = errors.New("logging")

	// ErrSecureRandFailed indicates that the teamserver could not read from the system secure random source.
	ErrSecureRandFailed = errors.New("failed to read from secure rand")

	//
	// Teamserver core errors.
	//

	// ErrConfig is an error related to the teamserver configuration.
	ErrConfig = errors.New("teamserver config")

	// ErrDatabaseConfig is an error related to the database configuration.
	ErrDatabaseConfig = errors.New("teamserver database configuration")

	// ErrDatabase is an error raised by the database backend.
	ErrDatabase = errors.New("database")

	// ErrTeamServer is an error raised by the teamserver core code.
	ErrTeamServer = errors.New("teamserver")

	// ErrCertificate is an error related to the certificate infrastructure.
	ErrCertificate = errors.New("certificates")

	// ErrUserConfig is an error related to users (teamclients) configuration files.
	ErrUserConfig = errors.New("user configuration")

	// ErrUnauthenticated indicates that a client user could not authenticate itself,
	// whether at connection time, or when requesting server-side features/info.
	ErrUnauthenticated = errors.New("User authentication failure")

	//
	// Listener errors.
	//

	// ErrNoListener indicates that the server could not find any listener/server
	// stack to run when one of its .Serve*() methods were invoked. If such an error
	// is raised, make sure you passed a server.Listener type with WithListener() option.
	ErrNoListener = errors.New("the teamserver has no listeners to start")

	// ErrListenerNotFound indicates that for a given ID, no running or persistent listener could be found.
	ErrListenerNotFound = errors.New("no listener exists with ID")

	// ErrListener indicates an error raised by a listener stack/implementation.
	ErrListener = errors.New("teamserver listener")
)
