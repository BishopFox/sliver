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
	"path/filepath"
	"runtime"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/db"
	"github.com/reeflective/team/internal/version"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Server is the core driver of an application teamserver.
// It is the counterpart to and plays a similar role than that
// of the team/client.Client type, ie. that it provides tools
// to any application/program to become a teamserver of itself.
//
// The server object can run on its own, without any teamclient attached
// or connected: it fulfills the reeflective/team.Client interface, and
// any teamserver can also be a client of itself, without configuration.
//
// The core job of the Server is to, non-exhaustively:
//   - Store and manage a list of users with a zero-trust identity system
//     (ie. public-key based), with ways to import/export these lists.
//   - Register, start and control teamserver listener/server stacks, for
//     many application clients to connect and consume the teamserver app.
//   - Offer version and user information to all teamclients.
//
// Additionally and similarly to the team/client.Client, it gives:
//   - Pre-configured loggers that listener stacks and server consumers
//     can use at any step of their application.
//   - Various options to configure its backends and behaviors.
//   - A builtin, app-specific abstracted filesystem (in-memory or on-disk).
//   - Additionally, an API to further register and control listeners.
//
// Various combinations of teamclient/teamserver usage are possible.
// Please see the Go module example/ directory for a list of them.
type Server struct {
	// Core
	name     string     // Name of the application using the teamserver.
	homeDir  string     // APP_ROOT_DIR var, evaluated once when creating the server.
	opts     *opts      // Server options
	fs       *assets.FS // Server filesystem, on-disk or embedded
	initOpts sync.Once  // Some options can only be set once when creating the server.

	// Logging
	fileLog  *logrus.Logger // Can be in-memory if the teamserver is configured.
	stdioLog *logrus.Logger // Logging level independent from the file logger.

	// Users
	userTokens *sync.Map      // Refreshed entirely when a user is kicked.
	certs      *certs.Manager // Manages all the certificate infrastructure.
	db         *gorm.DB       // Stores certificates and users data.
	dbInit     sync.Once      // A single database can be used in a teamserver lifetime.

	// Listeners and job control
	initServe sync.Once           // Some options can only have an effect at first start.
	self      Listener            // The default listener stack used by the teamserver.
	handlers  map[string]Listener // Other listeners available by name.
	jobs      *jobs               // Listeners job control
}

// New creates a new teamserver for the provided application name.
// Only one such teamserver should be created an application of any given name.
// Since by default, any teamserver can have any number of runtime clients, you
// are not required to provide any specific server.Listener type to serve clients.
//
// This call to create the server only creates the application default directory.
// No files, logs, connections or any interaction with the os/filesystem are made.
//
// Errors:
//   - All errors returned from this call are critical, in that the server could not
//     run properly in its most basic state, which may happen if the teamclient cannot
//     use and write to its on-disk directories/backends and log files.
//     No server is returned if the error is not nil.
//   - All methods of the teamserver core which return an error will always log this
//     error to the various teamserver log files/output streams, so that all actions
//     of teamserver can be recorded and watched out in various places.
func New(application string, options ...Options) (*Server, error) {
	server := &Server{
		name:       application,
		opts:       newDefaultOpts(),
		userTokens: &sync.Map{},
		jobs:       newJobs(),
		handlers:   make(map[string]Listener),
	}

	server.apply(options...)

	// Filesystem
	user, _ := user.Current()
	root := filepath.Join(user.HomeDir, "."+server.name)
	server.fs = assets.NewFileSystem(root, server.opts.inMemory)

	// Logging (if allowed)
	if err := server.initLogging(); err != nil {
		return nil, err
	}

	// Ensure we have a working database configuration,
	// and at least an in-memory sqlite database.
	if server.opts.dbConfig == nil {
		server.opts.dbConfig = server.getDefaultDatabaseConfig()
	}

	if server.opts.dbConfig.Database == db.SQLiteInMemoryHost && server.db == nil {
		if err := server.initDatabase(); err != nil {
			return nil, server.errorf("%w: %w", ErrDatabase, err)
		}
	}

	return server, nil
}

// Name returns the name of the application handled by the teamserver.
// Since you can embed multiple teamservers (one for each application)
// into a single binary, this is different from the program binary name
// running this teamserver.
func (ts *Server) Name() string {
	return ts.name
}

// Self returns a new application team/client.Client (with the same app name),
// using any provided options for client behavior.
//
// This teamclient implements by default the root team/Teamclient interface
// (directly through the server), but passing user-specific dialer stack options
// to this function and by providing the corresponding server options for your
// pair, you can use in-memory clients which will use the complete RPC stack
// of the application using this teamserver.
// See the server.Listener and client.Dialer types documentation for more.
func (ts *Server) Self(opts ...client.Options) *client.Client {
	teamclient, _ := client.New(ts.Name(), ts, opts...)

	return teamclient
}

// VersionClient implements team.Client.VersionClient() interface
// method, so that the teamserver can be a teamclient of itself.
// This simply returns the server.VersionServer() output.
func (ts *Server) VersionClient() (team.Version, error) {
	return ts.VersionServer()
}

// VersionServe returns the teamserver binary version information.
func (ts *Server) VersionServer() (team.Version, error) {
	semVer := version.Semantic()
	compiled, _ := version.Compiled()

	var major, minor, patch int32

	if len(semVer) == 3 {
		major = int32(semVer[0])
		minor = int32(semVer[1])
		patch = int32(semVer[2])
	}

	return team.Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Commit:     version.GitCommit(),
		Dirty:      version.GitDirty(),
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// Users returns the list of users in the teamserver database, and their information.
// Any error raised during querying the database is returned, along with all users.
func (ts *Server) Users() ([]team.User, error) {
	if err := ts.initDatabase(); err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	usersDB := []*db.User{}
	err := ts.dbSession().Find(&usersDB).Error

	users := make([]team.User, len(usersDB))

	if err != nil && len(usersDB) == 0 {
		return users, ts.errorf("%w: %w", ErrDatabase, err)
	}

	for i, user := range usersDB {
		users[i] = team.User{
			Name:     user.Name,
			LastSeen: user.LastSeen,
		}

		if _, ok := ts.userTokens.Load(user.Token); ok {
			users[i].Online = true
		}
	}

	return users, nil
}

// Filesystem returns an abstract filesystem used by the teamserver.
// This filesystem can be either of two things:
//   - By default, the on-disk filesystem, without any specific bounds.
//   - If the teamserver was created with the InMemory() option, a full
//     in-memory filesystem (with root `.app/`).
//
// Use cases for this filesystem might include:
//   - The wish to have a fully abstracted filesystem to work for testing
//   - Ensuring that the filesystem code in your application remains the
//     same regardless of the underlying, actual filesystem.
//
// The type returned is currently an internal type because it wraps some
// os.Filesystem methods for working more transparently: this may change
// in the future if the Go stdlib offers write support to its new io/fs.FS.
//
// SERVER note: Runtime clients can run with the client.InMemory() option,
// without any impact on the teamserver filesystem and its behavior.
func (ts *Server) Filesystem() *assets.FS {
	return ts.fs
}
