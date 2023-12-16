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

import "errors"

var (
	// ErrNoTeamclient indicates that the client cannot remotely query a server
	// to get its version or user information, because there is no client RPC
	// to do it. Make sure that your team/client.Client has been given one.
	ErrNoTeamclient = errors.New("this teamclient has no client implementation")

	// ErrConfig is an error related to the teamclient connection configuration.
	ErrConfig = errors.New("client config error")

	// ErrNoConfig indicates that no configuration, default or on file system, was found.
	ErrNoConfig = errors.New("no client configuration was selected or parsed")

	// ErrConfigNoUser says that the configuration has no user,
	// which is not possible even if the client is an in-memory one.
	ErrConfigNoUser = errors.New("client config with empty user")

	// ErrClient indicates an error raised by the client when igniting or connecting.
	// Most errors are raised by the underlying transport stack, which can be user-specific,
	// so users of this library should unwrap ErrClient errors to check against their owns.
	ErrClient = errors.New("teamclient dialer")
)
