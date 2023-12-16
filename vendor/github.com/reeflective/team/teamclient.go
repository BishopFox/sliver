package team

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

import "time"

// Client is the smallest interface which should be implemented by all
// teamclients of any sort, regardless of their use of the client/server
// packages in the reeflective/team Go module.
// This interface has been declared with various aims in mind:
//   - To provide a base reference/hint about what minimum functionality
//     is to be provided by the teamclients and teamservers alike.
//   - To harmonize the use of team/client and team/server core drivers.
type Client interface {
	// Users returns the list of teamserver users and their status.
	Users() ([]User, error)
	// VersionClient returns the compilation/version information for the client.
	VersionClient() (Version, error)
	// VersionServer returns the compilation/version information from a connected teamserver.
	VersionServer() (Version, error)
}

// User represents a teamserver user.
// This user shall be registered to a teamserver (ie. the teamserver should
// be in possession of the user cryptographic materials required to serve him)
// This type is returned by both team/clients and team/servers.
type User struct {
	Name     string
	Online   bool
	LastSeen time.Time
	Clients  int
}

// Version returns complete version/compilation information for a given binary.
// Therefore, two distinct version information can be provided by a teamclient
// connected to a remote (distinct runtime) server: the client binary version,
// and the server binary version.
// When a teamserver is serving itself in-memory, both versions will thus be identical.
//
// Note to developers: updating your teamserver/teamclient version information
// requires you to use `go generate ./...` at the root of your Go module code.
// The team/server and team/client will thus embed their respective version
// informations thanks to an automatic shell script generation.
// See the https://github.com/reeflective/team README/doc for more details.
type Version struct {
	Major      int32
	Minor      int32
	Patch      int32
	Commit     string
	Dirty      bool
	CompiledAt int64
	OS         string
	Arch       string
}
